package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/mapper"
	"block-explorer-backend/internal/service/model"
	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils/ethutils"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockRPC interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error)
	GetChainID(ctx context.Context) (*big.Int, error)
}

// BlockRepository call interface
type BlockRepository interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*models.Block, bool, error)
	InsertBlockWithTransactions(ctx context.Context, block *models.Block, txs []*models.Transaction) error
}

type TransactionReceiptSyncer interface {
	SyncBlockTransactionReceipts(ctx context.Context, blockNumber uint64) error
}

type BlockService struct {
	blockRPC                 BlockRPC
	blockRepo                BlockRepository
	transactionReceiptSyncer TransactionReceiptSyncer
}

func NewBlockService(
	blockRPC BlockRPC,
	blockRepo BlockRepository,
	transactionReceiptSyncer TransactionReceiptSyncer,
) *BlockService {
	return &BlockService{
		blockRPC:                 blockRPC,
		blockRepo:                blockRepo,
		transactionReceiptSyncer: transactionReceiptSyncer,
	}
}

func (s *BlockService) getRawBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	rpcBlock, err := s.blockRPC.GetBlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return rpcBlock, nil
}

func (s *BlockService) GetBlockByNumber(ctx context.Context, number uint64) (model.BlockQueryResult, error) {
	dbBlock, found, err := s.blockRepo.GetBlockByNumber(ctx, number)
	if err != nil {
		return model.BlockQueryResult{}, fmt.Errorf("get block %d from db: %w", number, err)
	}
	if found {
		return mapper.MapBlockEntityToQueryResult(dbBlock), nil
	}

	rpcBlock, err := s.getRawBlockByNumber(ctx, number)
	if err != nil {
		if errors.Is(err, types.ErrBlockNotFound) {
			return model.BlockQueryResult{}, types.ErrBlockNotFound
		}
		return model.BlockQueryResult{}, fmt.Errorf("get block detail by number %d: %w", number, err)
	}
	return mapper.MapRPCBlockToQueryResult(rpcBlock), nil
}

func (s *BlockService) validateBlockForSync(ctx context.Context, blockModel *models.Block) error {
	existingBlock, found, err := s.blockRepo.GetBlockByNumber(ctx, blockModel.Number)
	if err != nil {
		return fmt.Errorf("query block %d from db: %w", blockModel.Number, err)
	}
	if found {
		if strings.EqualFold(existingBlock.Hash, blockModel.Hash) {
			return nil
		}
		return fmt.Errorf(
			"reorg detected at block %d: db_hash=%s rpc_hash=%s: %w",
			blockModel.Number,
			existingBlock.Hash,
			blockModel.Hash,
			types.ErrReorgDetected,
		)
	}

	if blockModel.Number > 0 {
		parentBlock, found, err := s.blockRepo.GetBlockByNumber(ctx, blockModel.Number-1)
		if err != nil {
			return fmt.Errorf("query parent block %d from db: %w", blockModel.Number-1, err)
		}
		if found && !strings.EqualFold(blockModel.ParentHash, parentBlock.Hash) {
			return fmt.Errorf(
				"chain discontinuity at block %d: rpc_parent_hash=%s db_parent_hash=%s: %w",
				blockModel.Number,
				blockModel.ParentHash,
				parentBlock.Hash,
				types.ErrChainDiscontinuity,
			)
		}
	}
	return nil
}

func (s *BlockService) buildTransactionModelsFromBlock(ctx context.Context, block *ethtypes.Block) ([]*models.Transaction, error) {
	ethTxs := block.Transactions()
	if len(ethTxs) == 0 {
		return []*models.Transaction{}, nil
	}

	chainID, err := s.blockRPC.
		GetChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain id: %w", err)
	}

	txModels := make([]*models.Transaction, 0, len(ethTxs))
	signer := ethtypes.LatestSignerForChainID(chainID)

	for i, ethTx := range ethTxs {
		from, err := ethutils.RecoverSender(signer, ethTx)
		if err != nil {
			return nil, fmt.Errorf("recover sender for tx %s: %w", ethTx.Hash().Hex(), err)
		}
		txModel := mapper.ToTransactionModel(block, ethTx, uint(i), from)
		txModels = append(txModels, txModel)
	}
	return txModels, nil
}

func (s *BlockService) SyncBlockToDB(ctx context.Context, number uint64) error {
	block, err := s.getRawBlockByNumber(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch block %d from rpc: %w", number, err)
	}

	blockModel := mapper.ToBlockModel(block)

	if err := s.validateBlockForSync(ctx, blockModel); err != nil {
		return err
	}

	txModels, err := s.buildTransactionModelsFromBlock(ctx, block)
	if err != nil {
		return err
	}

	if err := s.blockRepo.InsertBlockWithTransactions(ctx, blockModel, txModels); err != nil {
		return fmt.Errorf("insert block %d into db: %w", number, err)
	}

	if err := s.transactionReceiptSyncer.SyncBlockTransactionReceipts(ctx, number); err != nil {
		return fmt.Errorf("sync transaction receipts for block %d: %w", number, err)
	}

	return nil
}

// SyncBlockRangetoDB handles manual block sync for debugging or recovery.
// Not used by automatic indexing logic.
func (s *BlockService) SyncBlockRangeToDB(ctx context.Context, start, end uint64) (*types.BlockRangeSyncResult, error) {
	if start > end {
		return nil, types.ErrInvalidBlockRange
	}

	const maxBlockRange uint64 = 100
	if end-start+1 > maxBlockRange {
		return nil, types.ErrBlockRangeTooLarge
	}

	result := &types.BlockRangeSyncResult{
		Start:     start,
		End:       end,
		Requested: end - start + 1,
	}

	for number := start; number <= end; number++ {
		select {
		case <-ctx.Done():
			return result, types.ErrRequestCanceled
		default:
		}

		err := s.SyncBlockToDB(ctx, number)
		if err != nil {
			result.Failed++
			result.FailedBlocks = append(result.FailedBlocks, number)
			if errors.Is(err, types.ErrReorgDetected) || errors.Is(err, types.ErrChainDiscontinuity) {
				return result, fmt.Errorf("sync block %d: %w", number, err)
			}
			continue
		}
		result.Succeeded++
	}
	return result, nil
}

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

// ChainBlockReader from  block_rpc
type ChainBlockReader interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error)
}

// BlockSyncStore from block_repo
type BlockSyncStore interface {
	GetBlockByNumber(ctx context.Context, chainID int64, number uint64) (*models.Block, bool, error)
	InsertBlockWithTransactions(ctx context.Context, block *models.Block, txs []*models.Transaction) error
	MarkBlockReceiptsSynced(ctx context.Context, chainID int64, blockNumber uint64) error
	MarkBlockReceiptsSyncFailed(ctx context.Context, chainID int64, blockNumber uint64, reason string) error
}

// BlockReceiptSyncer from tx_service
type BlockReceiptSyncer interface {
	SyncBlockTransactionReceipts(ctx context.Context, blockNumber uint64) error
}

// TransactionConflictReader from tx_repo
type TransactionConflictReader interface {
	GetTransactionsByHashes(ctx context.Context, chainID int64, hashes []string) (map[string]*models.Transaction, error)
}

type BlockService struct {
	chainID                   int64
	chainBlockReader          ChainBlockReader
	blockSyncStore            BlockSyncStore
	blockReceiptSyncer        BlockReceiptSyncer
	transactionConflictReader TransactionConflictReader

	startBlock uint64
}

func NewBlockService(
	chainID int64,
	chainBlockReader ChainBlockReader,
	blockSyncStore BlockSyncStore,
	blockReceiptSyncer BlockReceiptSyncer,
	transactionConflictReader TransactionConflictReader,
	startBlock uint64,
) *BlockService {
	return &BlockService{
		chainID:                   chainID,
		chainBlockReader:          chainBlockReader,
		blockSyncStore:            blockSyncStore,
		blockReceiptSyncer:        blockReceiptSyncer,
		transactionConflictReader: transactionConflictReader,
		startBlock:                startBlock,
	}
}

func (s *BlockService) getRawBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	rpcBlock, err := s.chainBlockReader.GetBlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return rpcBlock, nil
}

func (s *BlockService) GetBlockByNumber(ctx context.Context, number uint64) (model.BlockQueryResult, error) {
	dbBlock, found, err := s.blockSyncStore.GetBlockByNumber(ctx, s.chainID, number)
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
	return mapper.MapRPCBlockToQueryResult(s.chainID, rpcBlock), nil
}

func (s *BlockService) validateBlockForSync(ctx context.Context, blockModel *models.Block) error {
	existingBlock, found, err := s.blockSyncStore.GetBlockByNumber(ctx, s.chainID, blockModel.Number)
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
		parentBlock, found, err := s.blockSyncStore.GetBlockByNumber(ctx, s.chainID, blockModel.Number-1)
		if err != nil {
			return fmt.Errorf("query parent block %d from db: %w", blockModel.Number-1, err)
		}

		if !found {
			if blockModel.Number == s.startBlock {
				return nil
			}
			return fmt.Errorf(
				"parent block %d missing for block %d: %w",
				blockModel.Number-1,
				blockModel.Number,
				types.ErrChainDiscontinuity,
			)
		}
		if !strings.EqualFold(blockModel.ParentHash, parentBlock.Hash) {
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

func (s *BlockService) buildTransactionModelsFromBlock(block *ethtypes.Block) ([]*models.Transaction, error) {
	ethTxs := block.Transactions()
	if len(ethTxs) == 0 {
		return []*models.Transaction{}, nil
	}

	txModels := make([]*models.Transaction, 0, len(ethTxs))
	signer := ethtypes.LatestSignerForChainID(big.NewInt(s.chainID))

	for i, ethTx := range ethTxs {
		from, err := ethutils.RecoverSender(signer, ethTx)
		if err != nil {
			return nil, fmt.Errorf("recover sender for tx %s: %w", ethTx.Hash().Hex(), err)
		}
		txModel := mapper.ToTransactionModel(s.chainID, block, ethTx, uint(i), from)
		txModels = append(txModels, txModel)
	}
	return txModels, nil
}

func (s *BlockService) validateTransactionsForSync(
	ctx context.Context,
	txModels []*models.Transaction,
) error {
	if len(txModels) == 0 {
		return nil
	}

	hashes := make([]string, 0, len(txModels))
	for _, txModel := range txModels {
		hashes = append(hashes, txModel.Hash)
	}

	existingTxs, err := s.transactionConflictReader.GetTransactionsByHashes(ctx, s.chainID, hashes)
	if err != nil {
		return fmt.Errorf("query existing transactions by hashes: %w", err)
	}

	for _, txModel := range txModels {
		existingTx, exists := existingTxs[txModel.Hash]
		if !exists {
			continue
		}

		if existingTx.BlockNumber != txModel.BlockNumber ||
			!strings.EqualFold(existingTx.BlockHash, txModel.BlockHash) ||
			existingTx.TxIndex != txModel.TxIndex {
			return fmt.Errorf(
				"transaction conflict: tx_hash=%s existing_block_number=%d existing_block_hash=%s existing_tx_index=%d new_block_number=%d new_block_hash=%s new_tx_index=%d: %w",
				txModel.Hash,
				existingTx.BlockNumber,
				existingTx.BlockHash,
				existingTx.TxIndex,
				txModel.BlockNumber,
				txModel.BlockHash,
				txModel.TxIndex,
				types.ErrChainDataConflict,
			)
		}
	}
	return nil
}

func (s *BlockService) SyncBlockToDB(ctx context.Context, number uint64) error {
	block, err := s.getRawBlockByNumber(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch block %d from rpc: %w", number, err)
	}

	blockModel := mapper.ToBlockModel(s.chainID, block)

	if err := s.validateBlockForSync(ctx, blockModel); err != nil {
		return err
	}

	txModels, err := s.buildTransactionModelsFromBlock(block)
	if err != nil {
		return err
	}

	if err := s.validateTransactionsForSync(ctx, txModels); err != nil {
		return fmt.Errorf("validate transactions for block %d: %w", blockModel.Number, err)
	}

	blockModel.TransactionsSynced = true
	if len(txModels) == 0 {
		blockModel.ReceiptsSynced = true
		blockModel.SyncStatus = models.BlockSyncStatusCompleted
	} else {
		blockModel.ReceiptsSynced = false
		blockModel.SyncStatus = models.BlockSyncStatusTransactionsSynced
	}

	if err := s.blockSyncStore.InsertBlockWithTransactions(ctx, blockModel, txModels); err != nil {
		return fmt.Errorf("insert block %d into db: %w", number, err)
	}

	if len(txModels) == 0 {
		return nil
	}

	if err := s.blockReceiptSyncer.SyncBlockTransactionReceipts(ctx, number); err != nil {
		if markErr := s.blockSyncStore.MarkBlockReceiptsSyncFailed(ctx, s.chainID, number, err.Error()); markErr != nil {
			return fmt.Errorf(
				"sync receipts for block %d failed: %w; additionally failed to mark receipts sync failed: %v",
				number,
				err,
				markErr,
			)
		}
		return fmt.Errorf("sync receipts for block %d: %w", number, err)
	}
	if err := s.blockSyncStore.MarkBlockReceiptsSynced(ctx, s.chainID, number); err != nil {
		return fmt.Errorf("mark block %d receipts synced: %w", number, err)
	}
	return nil
}

// SyncBlockRangeToDB handles manual block sync for debugging or recovery.
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
			if errors.Is(err, types.ErrReorgDetected) ||
				errors.Is(err, types.ErrChainDiscontinuity) ||
				errors.Is(err, types.ErrChainDataConflict) {
				return result, fmt.Errorf("sync block %d: %w", number, err)
			}
			continue
		}
		result.Succeeded++
	}
	return result, nil
}

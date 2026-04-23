package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/mapper"
	"block-explorer-backend/internal/service/model"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"
	"log"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockRPC interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error)
}

// BlockRepository call interface
type BlockRepository interface {
	InsertBlock(ctx context.Context, block *models.Block) error
	GetBlockByNumber(ctx context.Context, number uint64) (*models.Block, bool, error)
}

type BlockService struct {
	blockRPC  BlockRPC
	blockRepo BlockRepository
}

func NewBlockService(blockRPC BlockRPC, blockRepo BlockRepository) *BlockService {
	return &BlockService{
		blockRPC:  blockRPC,
		blockRepo: blockRepo,
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
		log.Printf("[error] block query db failed: number=%d err=%v", number, err)
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
		log.Printf("[error] block query rpc failed: number=%d err=%v", number, err)
		return model.BlockQueryResult{}, fmt.Errorf("get block detail by number %d: %w", number, err)
	}
	return mapper.MapRPCBlockToQueryResult(rpcBlock), nil
}

func (s *BlockService) SyncBlockToDB(ctx context.Context, number uint64) error {
	block, err := s.getRawBlockByNumber(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch block %d from rpc: %w", number, err)
	}

	blockModel := toBlockModel(block)

	if err := s.blockRepo.InsertBlock(ctx, blockModel); err != nil {
		return fmt.Errorf("insert block %d into db: %w", number, err)
	}
	return nil
}

func toBlockModel(block *ethtypes.Block) *models.Block {

	return &models.Block{
		Number:     block.NumberU64(),
		Hash:       block.Hash().Hex(),
		ParentHash: block.ParentHash().Hex(),
		Timestamp:  block.Time(),
		Miner:      block.Coinbase().Hex(),
		TxCount:    len(block.Transactions()),
		GasUsed:    block.GasUsed(),
		GasLimit:   block.GasLimit(),
	}
}

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
			continue
		}
		result.Succeeded++
	}
	return result, nil
}

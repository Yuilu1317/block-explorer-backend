package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockRPC interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error)
}

// BlockRepository call interface
type BlockRepository interface {
	InsertBlock(ctx context.Context, block *models.Block) error
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
	block, err := s.blockRPC.GetBlockByNumber(ctx, number)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, types.ErrBlockNotFound
		}
		return nil, mapRPCError(err)
	}
	return block, nil
}

func (s *BlockService) GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockDetailDTO, error) {
	block, err := s.getRawBlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return toBlockDetailDTO(block), nil
}

func toBlockDetailDTO(block *ethtypes.Block) *types.BlockDetailDTO {
	return &types.BlockDetailDTO{
		Number:     block.NumberU64(),
		Hash:       block.Hash().Hex(),
		ParentHash: block.ParentHash().Hex(),
		Timestamp:  block.Time(),
		TxCount:    len(block.Transactions()),
		GasUsed:    block.GasUsed(),
		GasLimit:   block.GasLimit(),
	}
}

func (s *BlockService) SyncBlockToDB(ctx context.Context, number uint64) error {
	block, err := s.getRawBlockByNumber(ctx, number)
	if err != nil {
		return err
	}

	blockModel := toBlockModel(block)

	if err := s.blockRepo.InsertBlock(ctx, blockModel); err != nil {
		return fmt.Errorf("insert block to db: %w", err)
	}
	return nil
}

func toBlockModel(block *ethtypes.Block) *models.Block {
	var miner string
	if block.Coinbase() != (common.Address{}) {
		miner = block.Coinbase().Hex()
	}

	return &models.Block{
		Number:     block.NumberU64(),
		Hash:       block.Hash().Hex(),
		ParentHash: block.ParentHash().Hex(),
		Timestamp:  block.Time(),
		Miner:      miner,
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
			return result, ctx.Err()
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

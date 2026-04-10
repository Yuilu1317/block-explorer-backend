package service

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"

	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockRPC interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error)
}

type BlockService struct {
	blockRPC BlockRPC
}

func NewBlockService(blockRPC BlockRPC) *BlockService {
	return &BlockService{
		blockRPC: blockRPC,
	}
}

func (s *BlockService) GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockDetailDTO, error) {
	block, err := s.blockRPC.GetBlockByNumber(ctx, number)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded),
			errors.Is(err, context.Canceled):
			return nil, types.ErrRPCTimeout

		case errors.Is(err, ethereum.NotFound):
			return nil, types.ErrBlockNotFound

		default:
			return nil, err
		}
	}
	return s.toBlockRawDTO(block), nil
}
func (s *BlockService) toBlockRawDTO(block *ethtypes.Block) *types.BlockDetailDTO {
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

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
		if errors.Is(err, ethereum.NotFound) {
			return nil, types.ErrBlockNotFound
		}
		return nil, mapRPCError(err)
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

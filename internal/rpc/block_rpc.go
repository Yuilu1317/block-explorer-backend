package rpc

import (
	"context"
	"fmt"
	"math/big"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockRPC struct {
	BaseRPC
}

func NewBlockRPC(client *ethclient.Client, timeoutSeconds int) *BlockRPC {
	return &BlockRPC{
		BaseRPC: NewBaseRPC(client, timeoutSeconds),
	}
}

func (r *BlockRPC) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	blockNumber := new(big.Int).SetUint64(number)

	block, err := r.client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("get block by number from ethereum rpc: %w", err)
	}

	return block, nil
}

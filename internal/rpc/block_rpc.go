package rpc

import (
	"context"
	"fmt"
	"math/big"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockRPC struct {
	client         *ethclient.Client
	timeoutSeconds int
}

func NewBlockRPC(client *ethclient.Client, timeoutSeconds int) *BlockRPC {
	return &BlockRPC{
		client:         client,
		timeoutSeconds: timeoutSeconds,
	}
}

func (r *BlockRPC) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeoutSeconds)*time.Second)
	defer cancel()

	blockNumber := new(big.Int).SetUint64(number)

	block, err := r.client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("get block by number from ethereum rpc: %w", err)
	}

	return block, nil
}

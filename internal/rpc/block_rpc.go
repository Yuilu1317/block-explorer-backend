package rpc

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type BlockRPC struct {
	*BaseRPC
}

func NewBlockRPC(client *ethclient.Client, rpcClient *rpc.Client, timeoutSeconds int) *BlockRPC {
	return &BlockRPC{
		BaseRPC: NewBaseRPC(client, rpcClient, timeoutSeconds),
	}
}

func (r *BlockRPC) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	blockNumber := new(big.Int).SetUint64(number)

	block, err := r.client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, types.ErrBlockNotFound
		}
		mapped := mapRPCError(err)
		if mapped != err {
			return nil, mapped
		}
		return nil, fmt.Errorf("get block by number %d: %w", number, err)
	}
	return block, nil
}

func (r *BlockRPC) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	number, err := r.client.BlockNumber(ctx)
	if err != nil {
		mapped := mapRPCError(err)
		if mapped != err {
			return 0, mapped
		}
		return 0, fmt.Errorf("get latest block number: %w", err)
	}
	return number, nil
}

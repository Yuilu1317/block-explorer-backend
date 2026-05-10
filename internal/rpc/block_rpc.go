package rpc

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

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
		if mapped := mapRPCError(err); mapped != nil {
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
		if mapped := mapRPCError(err); mapped != nil {
			return 0, mapped
		}
		return 0, fmt.Errorf("get latest block number: %w", err)
	}
	return number, nil
}

type rpcBlockNumberResult struct {
	Number string `json:"number"`
}

func (r *BlockRPC) GetBlockNumberByTag(ctx context.Context, tag string) (uint64, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	if tag != "latest" && tag != "safe" && tag != "finalized" {
		return 0, fmt.Errorf("invalid block tag: %s", tag)
	}

	var result *rpcBlockNumberResult

	err := r.rpcClient.CallContext(
		ctx,
		&result,
		"eth_getBlockByNumber",
		tag,
		false,
	)
	if err != nil {
		if mapped := mapRPCError(err); mapped != nil {
			return 0, mapped
		}
		return 0, fmt.Errorf("get block number by tag %s: %w", tag, err)
	}
	if result == nil {
		return 0, types.ErrBlockNotFound
	}

	if result.Number == "" {
		return 0, fmt.Errorf("empty block number for tag %s", tag)
	}

	if !strings.HasPrefix(result.Number, "0x") {
		return 0, fmt.Errorf("invalid block number format for tag %s: %s", tag, result.Number)
	}
	number, err := strconv.ParseUint(strings.TrimPrefix(result.Number, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parse block number by tag %s: %w", tag, err)
	}

	return number, nil
}

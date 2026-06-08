package rpc

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type BaseRPC struct {
	client    *ethclient.Client
	rpcClient *rpc.Client
	timeout   time.Duration
}

func NewBaseRPC(client *ethclient.Client, rpcClient *rpc.Client, timeout time.Duration) *BaseRPC {
	return &BaseRPC{
		client:    client,
		rpcClient: rpcClient,
		timeout:   timeout,
	}
}

func (r *BaseRPC) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.timeout)
}

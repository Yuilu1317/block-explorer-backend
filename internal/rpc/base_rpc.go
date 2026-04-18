package rpc

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type BaseRPC struct {
	client         *ethclient.Client
	rpcClient      *rpc.Client
	timeoutSeconds int
}

func NewBaseRPC(client *ethclient.Client, rpcClient *rpc.Client, timeoutSeconds int) *BaseRPC {
	return &BaseRPC{
		client:         client,
		rpcClient:      rpcClient,
		timeoutSeconds: timeoutSeconds,
	}
}

func (r *BaseRPC) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(r.timeoutSeconds)*time.Second)
}

package rpc

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

type BaseRPC struct {
	client         *ethclient.Client
	timeoutSeconds int
}

func NewBaseRPC(client *ethclient.Client, timeoutSeconds int) BaseRPC {
	return BaseRPC{
		client:         client,
		timeoutSeconds: timeoutSeconds,
	}
}

func (r *BaseRPC) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(r.timeoutSeconds)*time.Second)
}

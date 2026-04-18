package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type AddressRPC struct {
	*BaseRPC
}

func NewAddressRPC(client *ethclient.Client, rpcClient *rpc.Client, timeoutSeconds int) *AddressRPC {
	return &AddressRPC{
		BaseRPC: NewBaseRPC(client, rpcClient, timeoutSeconds),
	}
}

func (r *AddressRPC) GetBalance(ctx context.Context, address string) (string, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()
	addr := common.HexToAddress(address)

	balance, err := r.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return "", err
	}
	return balance.String(), nil
}

func (r *AddressRPC) GetNonce(ctx context.Context, address string) (uint64, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	addr := common.HexToAddress(address)

	nonce, err := r.client.NonceAt(ctx, addr, nil)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}
func (r *AddressRPC) GetCode(ctx context.Context, address string) (string, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	addr := common.HexToAddress(address)

	code, err := r.client.CodeAt(ctx, addr, nil)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(code), nil
}

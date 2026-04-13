package rpc

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TxRPC provides transaction-related Ethereum RPC operations.
type TxRPC struct {
	BaseRPC
}

// NewTxRPC creates a transaction RPC helper with the configured timeout.
func NewTxRPC(client *ethclient.Client, timeoutSeconds int) *TxRPC {
	return &TxRPC{
		BaseRPC: NewBaseRPC(client, timeoutSeconds),
	}
}

// GetTransactionByHash fetches a transaction and its receipt data by hash.
func (r *TxRPC) GetTransactionByHash(ctx context.Context, hash string) (*types.TxRaw, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	txHash := common.HexToHash(hash)

	tx, isPending, err := r.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ethereum.NotFound
	}

	chainID, err := r.client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch network id: %w", err)
	}

	signer := gethtypes.LatestSignerForChainID(chainID)
	from, err := signer.Sender(tx)
	if err != nil {
		return nil, fmt.Errorf("derive sender from tx: %w", err)
	}

	var receipt *gethtypes.Receipt
	if !isPending {
		receipt, err = r.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			if !errors.Is(err, ethereum.NotFound) {
				return nil, fmt.Errorf("fetch receipt: %w", err)
			}
		}
	}
	return &types.TxRaw{
		Tx:        tx,
		From:      from.Hex(),
		IsPending: isPending,
		Receipt:   receipt,
	}, nil
}

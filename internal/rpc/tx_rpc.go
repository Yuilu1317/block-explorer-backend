package rpc

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"block-explorer-backend/internal/types"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TxRPC provides transaction-related Ethereum RPC operations.
type TxRPC struct {
	client         *ethclient.Client
	timeoutSeconds int
}

// NewTxRPC creates a transaction RPC helper with the configured timeout.
func NewTxRPC(client *ethclient.Client, timeoutSeconds int) *TxRPC {
	return &TxRPC{
		client:         client,
		timeoutSeconds: timeoutSeconds,
	}
}

// GetTransactionByHash fetches a transaction and its receipt data by hash.
func (r *TxRPC) GetTransactionByHash(ctx context.Context, hash string) (*types.TxDetailDTO, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeoutSeconds)*time.Second)
	defer cancel()

	txHash := common.HexToHash(hash)

	tx, isPending, err := r.client.TransactionByHash(ctx, txHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, types.ErrTxNotFound
		}
		return nil, fmt.Errorf("fetch transaction by hash: %w", err)
	}
	if tx == nil {
		return nil, types.ErrTxNotFound
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

	var receiptStatus *uint64
	var gasUsed *uint64
	var blockNumber *uint64

	if !isPending {
		receipt, err := r.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, fmt.Errorf("fetch transaction receipt: %w", err)
		}

		status := receipt.Status
		receiptStatus = &status

		used := receipt.GasUsed
		gasUsed = &used

		if receipt.BlockNumber != nil {
			bn := receipt.BlockNumber.Uint64()
			blockNumber = &bn
		}
	}

	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	valueWei := tx.Value()
	gasPriceWei := tx.GasPrice()
	if gasPriceWei == nil {
		gasPriceWei = big.NewInt(0)
	}

	dto := &types.TxDetailDTO{
		Hash:        tx.Hash().Hex(),
		From:        from.Hex(),
		To:          to,
		ValueWei:    valueWei.String(),
		Nonce:       tx.Nonce(),
		GasLimit:    tx.Gas(),
		GasPriceWei: gasPriceWei.String(),
		Data:        fmt.Sprintf("0x%x", tx.Data()),
		IsPending:   isPending,
		BlockNumber: blockNumber,
		Status:      receiptStatus,
		GasUsed:     gasUsed,
	}

	return dto, nil
}

package service

import (
	"context"
	"strings"

	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"
)

// TxRPC defines the RPC capability required by the service layer.
// The concrete implementation is provided by the rpc layer.
type TxRPC interface {
	GetTransactionByHash(ctx context.Context, hash string) (*types.TxDetailDTO, error)
}

// TxService handles transaction-related business logic.
// It depends on the RPC abstraction rather than a concrete implementation.
type TxService struct {
	// txRPC is the RPC dependency used to fetch transaction data from the blockchain
	txRPC TxRPC
}

// NewTxService creates and initializes a TxService instance.
func NewTxService(txRPC TxRPC) *TxService {
	return &TxService{
		txRPC: txRPC,
	}
}

// GetTxByHash validates the input hash, calls the RPC layer,
// and returns the transaction details if found.
func (s *TxService) GetTxByHash(ctx context.Context, hash string) (*types.TxDetailDTO, error) {
	// Remove leading and trailing whitespace from the input hash.
	hash = strings.TrimSpace(hash)

	// Validate whether the transaction hash format is correct.
	if err := utils.ValidateTxHash(hash); err != nil {
		return nil, err
	}

	// Query the transaction details from the RPC layer.
	tx, err := s.txRPC.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	// Return the transaction details to the upper layer.
	return tx, nil
}

package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/mapper"
	"context"
	"fmt"
	"strings"

	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"
)

type TxRPC interface {
	GetTransactionByHash(ctx context.Context, hash string) (*types.TxRaw, error)
}

type TxRepo interface {
	GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, bool, error)
}

type TxService struct {
	// txRPC is the RPC dependency used to fetch transaction data from the blockchain
	txRPC  TxRPC
	txRepo TxRepo
}

func NewTxService(txRPC TxRPC, txRepo TxRepo) *TxService {
	return &TxService{
		txRPC:  txRPC,
		txRepo: txRepo,
	}
}

func (s *TxService) GetTxDetailByHashFromRPC(ctx context.Context, hash string) (*types.TxDetailDTO, error) {
	hash = strings.TrimSpace(hash)
	hash = strings.ToLower(hash)

	if err := utils.ValidateTxHash(hash); err != nil {
		return nil, types.ErrInvalidTxHash
	}

	raw, err := s.txRPC.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get transaction by hash %s: %w", hash, err)
	}

	return mapper.ToTxDetailDTO(raw), nil
}

func (s *TxService) GetIndexedTransactionByHash(ctx context.Context, hash string) (*types.IndexedTransactionDTO, error) {
	hash = strings.TrimSpace(hash)
	hash = strings.ToLower(hash)

	if err := utils.ValidateTxHash(hash); err != nil {
		return nil, types.ErrInvalidTxHash
	}

	tx, found, err := s.txRepo.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get indexed transaction by hash %s from db: %w", hash, err)
	}

	if !found {
		return nil, types.ErrTxNotFound
	}

	return mapper.ToIndexedTransactionDTO(tx), nil
}

package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/mapper"
	"context"
	"errors"
	"fmt"
	"strings"

	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

type ChainTransactionReader interface {
	GetTransactionByHash(ctx context.Context, hash string) (*types.TxRaw, error)
	GetTransactionReceipt(ctx context.Context, hash string) (*gethtypes.Receipt, error)
}

type IndexedTransactionReader interface {
	GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, bool, error)
}

type TransactionReceiptStore interface {
	UpdateTransactionReceiptByHash(
		ctx context.Context,
		hash string,
		status *uint64,
		gasUsed *uint64,
	) error

	ListTransactionsMissingReceiptByBlockNumber(
		ctx context.Context,
		blockNumber uint64,
	) ([]*models.Transaction, error)
}

type TxService struct {
	chainTransactionReader   ChainTransactionReader
	indexedTransactionReader IndexedTransactionReader
	transactionReceiptStore  TransactionReceiptStore
}

func NewTxService(
	chainTransactionReader ChainTransactionReader,
	indexedTransactionReader IndexedTransactionReader,
	transactionReceiptStore TransactionReceiptStore,
) *TxService {
	return &TxService{
		chainTransactionReader:   chainTransactionReader,
		indexedTransactionReader: indexedTransactionReader,
		transactionReceiptStore:  transactionReceiptStore,
	}
}

func (s *TxService) GetTxDetailByHashFromRPC(ctx context.Context, hash string) (*types.TxDetailDTO, error) {
	hash = strings.TrimSpace(hash)
	hash = strings.ToLower(hash)

	if err := utils.ValidateTxHash(hash); err != nil {
		return nil, types.ErrInvalidTxHash
	}

	raw, err := s.chainTransactionReader.GetTransactionByHash(ctx, hash)
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

	tx, found, err := s.indexedTransactionReader.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get indexed transaction by hash %s from db: %w", hash, err)
	}

	if !found {
		return nil, types.ErrTxNotFound
	}

	return mapper.ToIndexedTransactionDTO(tx), nil
}

func (s *TxService) validateReceiptMatchesTransaction(tx *models.Transaction, receipt *gethtypes.Receipt) error {
	if receipt == nil {
		return fmt.Errorf("receipt is nil: tx=%s", tx.Hash)
	}
	if receipt.TxHash != common.HexToHash(tx.Hash) {
		return fmt.Errorf("receipt tx hash mismatch: tx=%s receipt=%s", tx.Hash, receipt.TxHash.Hex())
	}
	if receipt.BlockNumber == nil {
		return fmt.Errorf("receipt block number is nil: tx=%s", tx.Hash)
	}
	if receipt.BlockNumber.Uint64() != tx.BlockNumber {
		return fmt.Errorf(
			"receipt block number mismatch: tx=%s expected=%d got=%d",
			tx.Hash,
			tx.BlockNumber,
			receipt.BlockNumber.Uint64(),
		)
	}
	if receipt.BlockHash != common.HexToHash(tx.BlockHash) {
		return fmt.Errorf("receipt block hash mismatch: tx=%s", tx.Hash)
	}
	return nil
}

func (s *TxService) SyncBlockTransactionReceipts(ctx context.Context, blockNumber uint64) error {
	txs, err := s.transactionReceiptStore.ListTransactionsMissingReceiptByBlockNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("service: list missing receipt transactions for block %d: %w", blockNumber, err)
	}

	for _, tx := range txs {
		receipt, err := s.chainTransactionReader.GetTransactionReceipt(ctx, tx.Hash)
		if err != nil {
			if errors.Is(err, types.ErrTxReceiptNotFound) {
				continue
			}
			return fmt.Errorf("service: get transaction receipt for tx %s: %w", tx.Hash, err)
		}
		if err := s.validateReceiptMatchesTransaction(tx, receipt); err != nil {
			return fmt.Errorf("service: validate transaction receipt for tx %s: %w", tx.Hash, err)
		}
		status := receipt.Status
		gasUsed := receipt.GasUsed

		if err := s.transactionReceiptStore.UpdateTransactionReceiptByHash(ctx, tx.Hash, &status, &gasUsed); err != nil {
			return fmt.Errorf("service: update transaction receipt for tx %s: %w", tx.Hash, err)
		}
	}
	return nil
}

package repo

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) InsertTransaction(ctx context.Context, tx *models.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(tx).Error; err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("insert transaction %s: %w", tx.Hash, err)
	}
	return nil
}

func (r *TransactionRepository) InsertTransactions(ctx context.Context, txs []*models.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&txs).Error; err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("insert transactions: %w", err)
	}
	return nil
}

func (r *TransactionRepository) GetTransactionByHash(ctx context.Context, chainID int64, hash string) (*models.Transaction, bool, error) {
	var tx models.Transaction

	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Where("hash = ?", hash).
		Take(&tx).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		if mapped := mapDBError(err); mapped != nil {
			return nil, false, mapped
		}
		return nil, false, fmt.Errorf("query transaction by hash %s on chain %d: %w", hash, chainID, err)
	}

	return &tx, true, nil
}

func (r *TransactionRepository) GetTransactionsByHashes(
	ctx context.Context,
	chainID int64,
	hashes []string,
) (map[string]*models.Transaction, error) {
	result := make(map[string]*models.Transaction, len(hashes))

	if len(hashes) == 0 {
		return result, nil
	}

	var txs []models.Transaction
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Where("hash IN ?", hashes).
		Find(&txs).
		Error
	if err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return nil, mapped
		}
		return nil, fmt.Errorf("query transactions by hashes: %w", err)
	}

	for i := range txs {
		result[txs[i].Hash] = &txs[i]
	}

	return result, nil
}

func (r *TransactionRepository) ListTransactionsByAddress(
	ctx context.Context,
	chainID int64,
	address string,
	limit int,
	offset int,
) ([]models.Transaction, error) {
	txs := make([]models.Transaction, 0)

	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Where("from_address_lower = ? OR to_address_lower = ?", address, address).
		Order("block_number DESC, tx_index DESC").
		Limit(limit).
		Offset(offset).
		Find(&txs).
		Error

	if err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return nil, mapped
		}
		return nil, fmt.Errorf("list transactions by address %s: %w", address, err)
	}
	return txs, nil
}

func (r *TransactionRepository) UpdateTransactionReceiptByHash(
	ctx context.Context,
	chainID int64,
	hash string,
	status *uint64,
	gasUsed *uint64,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("chain_id = ?", chainID).
		Where("hash = ?", hash).
		Updates(map[string]any{
			"receipt_status":   status,
			"receipt_gas_used": gasUsed,
		})
	if result.Error != nil {
		if mapped := mapDBError(result.Error); mapped != nil {
			return mapped
		}
		return fmt.Errorf("update transaction receipt by hash %s on chain %d: %w", hash, chainID, result.Error)
	}

	if result.RowsAffected == 0 {
		return types.ErrTxNotFound
	}
	return nil
}

func (r *TransactionRepository) ListTransactionsMissingReceiptByBlockNumber(
	ctx context.Context,
	chainID int64,
	blockNumber uint64,
) ([]*models.Transaction, error) {
	txs := make([]*models.Transaction, 0)

	result := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("chain_id = ?", chainID).
		Where("block_number = ? AND (receipt_status IS NULL OR receipt_gas_used IS NULL)", blockNumber).
		Order("tx_index ASC").
		Find(&txs)

	if result.Error != nil {
		return nil, fmt.Errorf("list transactions missing receipt by block number %d on chain %d: %w", blockNumber, chainID, result.Error)
	}

	return txs, nil
}

func (r *TransactionRepository) ListWalletCompletedTransactionRows(
	ctx context.Context,
	chainID int64,
	blockNumbers []uint64,
) ([]models.Transaction, error) {
	if len(blockNumbers) == 0 {
		return []models.Transaction{}, nil
	}
	var txs []models.Transaction
	err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("chain_id = ?", chainID).
		Where("block_number IN ?", blockNumbers).
		Order("block_number ASC").
		Order("tx_index ASC").
		Find(&txs).
		Error
	if err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return nil, fmt.Errorf("list wallet completed transaction rows: %w", mapped)
		}
		return nil, fmt.Errorf("list wallet completed transaction rows: %w", err)
	}
	return txs, nil
}

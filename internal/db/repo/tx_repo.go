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

func (r *TransactionRepository) GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, bool, error) {
	var tx models.Transaction

	err := r.db.WithContext(ctx).Where("hash = ?", hash).Take(&tx).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		if mapped := mapDBError(err); mapped != nil {
			return nil, false, mapped
		}
		return nil, false, fmt.Errorf("query transaction by hash %s: %w", hash, err)
	}

	return &tx, true, nil
}

func (r *TransactionRepository) GetTransactionsByHashes(
	ctx context.Context,
	hashes []string,
) (map[string]*models.Transaction, error) {
	result := make(map[string]*models.Transaction, len(hashes))

	if len(hashes) == 0 {
		return result, nil
	}

	var txs []models.Transaction
	err := r.db.WithContext(ctx).
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
	address string,
	limit int,
	offset int,
) ([]models.Transaction, error) {
	txs := make([]models.Transaction, 0)

	err := r.db.WithContext(ctx).
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
	hash string,
	status *uint64,
	gasUsed *uint64,
) error {
	result := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("hash = ?", hash).Updates(map[string]any{
		"receipt_status":   status,
		"receipt_gas_used": gasUsed,
	})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return types.ErrTxNotFound
	}
	return nil
}

func (r *TransactionRepository) ListTransactionsMissingReceiptByBlockNumber(
	ctx context.Context,
	blockNumber uint64,
) ([]*models.Transaction, error) {
	txs := make([]*models.Transaction, 0)

	result := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("block_number = ? AND (receipt_status IS NULL OR receipt_gas_used IS NULL)", blockNumber).
		Order("tx_index ASC").
		Find(&txs)

	if result.Error != nil {
		return nil, result.Error
	}

	return txs, nil
}

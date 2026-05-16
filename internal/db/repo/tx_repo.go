package repo

import (
	"block-explorer-backend/internal/db/models"
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

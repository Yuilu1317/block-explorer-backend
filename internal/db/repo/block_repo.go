package repo

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BlockRepository struct {
	db *gorm.DB
}

func NewBlockRepository(db *gorm.DB) *BlockRepository {
	return &BlockRepository{db: db}
}

func (r *BlockRepository) InsertBlock(ctx context.Context, block *models.Block) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(block).Error; err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("insert block: %w", err)
	}
	return nil
}

func (r *BlockRepository) GetLatestFullySyncedBlockNumber(ctx context.Context, chainID int64) (uint64, bool, error) {
	if chainID <= 0 {
		return 0, false, fmt.Errorf("chain_id must be positive")
	}
	var latestFullyNumber sql.NullInt64

	err := r.db.WithContext(ctx).
		Model(&models.Block{}).
		Where("chain_id = ?", chainID).
		Where("sync_status = ?", models.BlockSyncStatusCompleted).
		Where("transactions_synced = ? AND receipts_synced = ?", true, true).
		Where("last_sync_error IS NULL").
		Select("MAX(number)").
		Scan(&latestFullyNumber).
		Error
	if err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return 0, false, mapped
		}
		return 0, false, fmt.Errorf("query latest fully synced block number: %w", err)
	}

	if !latestFullyNumber.Valid {
		return 0, false, nil
	}

	return uint64(latestFullyNumber.Int64), true, nil
}

func (r *BlockRepository) GetBlockByNumber(ctx context.Context, chainID int64, number uint64) (*models.Block, bool, error) {
	var block models.Block

	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Where("number = ?", number).
		Take(&block).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		if mapped := mapDBError(err); mapped != nil {
			return nil, false, mapped
		}
		return nil, false, fmt.Errorf("query block by number %d on chain %d: %w", number, chainID, err)
	}

	return &block, true, nil
}

func (r *BlockRepository) InsertBlockWithTransactions(ctx context.Context, block *models.Block, txs []*models.Transaction) error {
	return r.db.WithContext(ctx).Transaction(func(txDB *gorm.DB) error {
		if err := txDB.Clauses(clause.OnConflict{DoNothing: true}).Create(block).Error; err != nil {
			if mapped := mapDBError(err); mapped != nil {
				return mapped
			}
			return fmt.Errorf("insert block %d: %w", block.Number, err)
		}
		if len(txs) == 0 {
			return nil
		}

		if err := txDB.Clauses(clause.OnConflict{DoNothing: true}).Create(&txs).Error; err != nil {
			if mapped := mapDBError(err); mapped != nil {
				return mapped
			}
			return fmt.Errorf("insert transactions for block %d: %w", block.Number, err)
		}

		return nil
	})
}

func (r *BlockRepository) MarkBlockReceiptsSynced(ctx context.Context, chainID int64, blockNumber uint64) error {
	result := r.db.WithContext(ctx).
		Model(&models.Block{}).
		Where("chain_id = ?", chainID).
		Where("number = ?", blockNumber).
		Updates(map[string]any{
			"receipts_synced": true,
			"sync_status":     models.BlockSyncStatusCompleted,
			"last_sync_error": nil,
		})

	if result.Error != nil {
		if mapped := mapDBError(result.Error); mapped != nil {
			return mapped
		}
		return fmt.Errorf("mark block %d receipts synced: %w", blockNumber, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("mark block %d receipts synced: %w", blockNumber, types.ErrBlockNotFound)
	}

	return nil
}

func (r *BlockRepository) MarkBlockReceiptsSyncFailed(ctx context.Context, chainID int64, blockNumber uint64, reason string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Block{}).
		Where("chain_id = ?", chainID).
		Where("number = ?", blockNumber).
		Updates(map[string]any{
			"receipts_synced": false,
			"sync_status":     models.BlockSyncStatusReceiptsFailed,
			"last_sync_error": reason,
		})
	if result.Error != nil {
		if mapped := mapDBError(result.Error); mapped != nil {
			return mapped
		}
		return fmt.Errorf("mark block %d receipts sync failed: %w", blockNumber, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("mark block %d receipts sync failed: %w", blockNumber, types.ErrBlockNotFound)
	}
	return nil
}

func (r *BlockRepository) ListWalletCompletedBlockRows(
	ctx context.Context,
	chainID int64,
	fromBlock int64,
	limit int,
) ([]models.Block, error) {
	if fromBlock < 0 {
		return nil, fmt.Errorf("from_block must be non-negative")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	fromBlockNumber := uint64(fromBlock)
	var blocks []models.Block

	err := r.db.WithContext(ctx).
		Model(&models.Block{}).
		Where("chain_id = ?", chainID).
		Where("number >= ?", fromBlockNumber).
		Where("transactions_synced = ?", true).
		Where("receipts_synced = ?", true).
		Where("sync_status = ?", models.BlockSyncStatusCompleted).
		Where("last_sync_error IS NULL").
		Order("number ASC").
		Limit(limit).
		Find(&blocks).
		Error
	if err != nil {
		if mapped := mapDBError(err); mapped != nil {
			return nil, mapped
		}
		return nil, fmt.Errorf("list wallet completed block rows: %w", err)
	}
	return blocks, nil
}

func (r *BlockRepository) GetLatestCompletedBlock(
	ctx context.Context,
	chainID int64,
) (*models.Block, bool, error) {
	var block models.Block

	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Where("sync_status = ?", models.BlockSyncStatusCompleted).
		Where("transactions_synced = ?", true).
		Where("receipts_synced = ?", true).
		Where("last_sync_error IS NULL").
		Order("number DESC").
		First(&block).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		if mapped := mapDBError(err); mapped != nil {
			return nil, false, mapped
		}
		return nil, false, fmt.Errorf("get latest completed block: %w", err)
	}
	return &block, true, nil
}

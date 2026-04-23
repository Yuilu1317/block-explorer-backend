package repo

import (
	"block-explorer-backend/internal/db/models"
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
		mapped := mapDBError(err)
		if mapped != err {
			return mapped
		}
		return fmt.Errorf("insert block: %w", err)
	}
	return nil
}

func (r *BlockRepository) GetLatestBlockNumber(ctx context.Context) (uint64, bool, error) {
	var number sql.NullInt64

	err := r.db.WithContext(ctx).Model(&models.Block{}).Select("MAX(number)").Scan(&number).Error
	if err != nil {
		mapped := mapDBError(err)
		if mapped != err {
			return 0, false, mapped
		}
		return 0, false, fmt.Errorf("query latest block number: %w", err)
	}

	if !number.Valid {
		return 0, false, nil
	}

	return uint64(number.Int64), true, nil
}

func (r *BlockRepository) GetBlockByNumber(ctx context.Context, number uint64) (*models.Block, bool, error) {
	var block models.Block

	err := r.db.WithContext(ctx).Where("number = ?", number).Take(&block).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		mapped := mapDBError(err)
		if mapped != err {
			return nil, false, mapped
		}
		return nil, false, fmt.Errorf("query block by number %d: %w", number, err)
	}

	return &block, false, nil
}

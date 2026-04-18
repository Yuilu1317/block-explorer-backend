package repo

import (
	"block-explorer-backend/internal/db/models"
	"context"
	"database/sql"
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
		return fmt.Errorf("insert block failed: %w", err)
	}
	return nil
}

func (r *BlockRepository) GetLatestBlockNumber(ctx context.Context) (uint64, bool, error) {
	var number sql.NullInt64

	err := r.db.WithContext(ctx).Model(&models.Block{}).Select("MAX(number)").Scan(&number).Error
	if err != nil {
		return 0, false, err
	}

	if !number.Valid {
		return 0, false, nil
	}

	return uint64(number.Int64), true, nil
}

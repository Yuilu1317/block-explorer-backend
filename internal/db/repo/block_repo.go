package repo

import (
	"block-explorer-backend/internal/db/models"
	"context"
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

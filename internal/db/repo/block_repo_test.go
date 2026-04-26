package repo

import (
	"block-explorer-backend/internal/db/models"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.Block{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func TestBlockRepository_InsertBlock_Success(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	err := r.InsertBlock(context.Background(), &models.Block{
		Number: 20,
		Hash:   "0x123",
	})
	if err != nil {
		t.Fatalf("insert block: %v", err)
	}

	var block models.Block
	err = db.First(&block, "number = ?", 20).Error
	if err != nil {
		t.Fatalf("expected block inserted, got error: %v", err)
	}
	if block.Hash != "0x123" {
		t.Fatalf("expected hash=0x123, got %s", block.Hash)
	}
}

func TestBlockRepository_InsertBlock_DuplicateIgnored(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	ctx := context.Background()
	block := &models.Block{
		Number: 20,
		Hash:   "0x123",
	}

	err := r.InsertBlock(ctx, block)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	err = r.InsertBlock(ctx, block)
	if err != nil {
		t.Fatalf("second insert failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.Block{}).Count(&count).Error; err != nil {
		t.Fatalf("count query failed: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 record, got %d", count)
	}
}

func TestBlockRepository_InsertBlock_DBError(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlDB: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	err = r.InsertBlock(ctx, &models.Block{
		Number: 1,
		Hash:   "0xabc",
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBlockRepository_GetLatestBlockNumber_EmptyTable(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	number, found, err := r.GetLatestBlockNumber(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if found {
		t.Fatalf("expected found=false, got true")
	}
	if number != 0 {
		t.Fatalf("expected number=0, got %d", number)
	}
}

func TestBlockRepository_GetLatestBlockNumber_WithBlocks(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	err := db.Create(&models.Block{
		Number: 25,
		Hash:   "0xabc",
	}).Error
	if err != nil {
		t.Fatalf("seed block: %v", err)
	}

	number, found, err := r.GetLatestBlockNumber(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !found {
		t.Fatalf("expected found=true, got false")
	}
	if number != 25 {
		t.Fatalf("expected number=25, got %d", number)
	}
}

func TestBlockRepository_GetLatestBlockNumber_MultipleBlocks(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	err := db.Create(&[]models.Block{
		{Number: 40, Hash: "0xzxc"},
		{Number: 30, Hash: "0xdef"},
		{Number: 50, Hash: "0xghi"},
	}).Error
	if err != nil {
		t.Fatalf("seed block: %v", err)
	}

	number, found, err := r.GetLatestBlockNumber(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !found {
		t.Fatalf("expected found=true, got false")
	}
	if number != 50 {
		t.Fatalf("expected number=50, got %d", number)
	}
}

func TestBlockRepository_GetBlockByNumber_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	block, found, err := r.GetBlockByNumber(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if found {
		t.Fatalf("expected found=false, got true")
	}
	if block != nil {
		t.Fatalf("expected block=nil, got %+v", block)
	}
}

func TestBlockRepository_GetBlockByHash_Found(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)
	ctx := context.Background()

	err := db.Create(&models.Block{
		Number: 20,
		Hash:   "0x123",
	}).Error
	if err != nil {
		t.Fatalf("seed block: %v", err)
	}

	block, found, err := r.GetBlockByNumber(ctx, 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !found {
		t.Fatalf("expected found=true, got false")
	}
	if block == nil {
		t.Fatalf("expected block, got nil")
	}
	if block.Number != 20 {
		t.Fatalf("expected number=20, got %d", block.Number)
	}
	if block.Hash != "0x123" {
		t.Fatalf("expected hash=0x123, got %s", block.Hash)
	}
}

package repo

import (
	"block-explorer-backend/internal/db/models"
	"context"
	"testing"

	"gorm.io/gorm"
)

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

func TestBlockRepository_GetBlockByNumber_Found(t *testing.T) {
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

func setupBlockWithTransactionsRepo(t *testing.T) (*BlockRepository, *gorm.DB) {
	t.Helper()

	db := SetupTestDB(t, &models.Block{}, &models.Transaction{})
	return NewBlockRepository(db), db
}

func newBlockWithTxTestBlock(number uint64, hash string) *models.Block {
	return &models.Block{
		Number:     number,
		Hash:       hash,
		ParentHash: "0xparenthash",
		Timestamp:  1700000000,
		Miner:      "0x1111111111111111111111111111111111111111",
		GasLimit:   30000000,
		GasUsed:    21000,
		TxCount:    3,
	}
}

func newBlockWithTxTestTransaction(hash string, blockNumber uint64, blockHash string, txIndex uint) *models.Transaction {
	return &models.Transaction{
		Hash:        hash,
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		TxIndex:     txIndex,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		Nonce:       uint64(txIndex),
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0x",
	}
}

func TestBlockRepository_InsertBlockWithTransactions_Success(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	block := newBlockWithTxTestBlock(100, "0xblockhash100")
	txs := []*models.Transaction{
		newBlockWithTxTestTransaction("0xtxhash1", 100, "0xblockhash100", 0),
		newBlockWithTxTestTransaction("0xtxhash2", 100, "0xblockhash100", 1),
		newBlockWithTxTestTransaction("0xtxhash3", 100, "0xblockhash100", 2),
	}
	if err := r.InsertBlockWithTransactions(ctx, block, txs); err != nil {
		t.Fatalf("insert block with transactions: %v", err)
	}
	var blockCount int64
	if err := db.Model(&models.Block{}).Where("number = ?", 100).Count(&blockCount).Error; err != nil {
		t.Fatalf("count blocks: %v", err)
	}
	if blockCount != 1 {
		t.Fatalf("expected 1 block, got %d", blockCount)
	}

	var txCount int64
	if err := db.Model(&models.Transaction{}).Where("block_number = ?", 100).Count(&txCount).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if txCount != 3 {
		t.Fatalf("expected 3 transactions, got %d", txCount)
	}
}

func TestBlockRepository_InsertBlockWithTransactions_EmptyTransactions(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	block := newBlockWithTxTestBlock(100, "0xblockhash100")

	if err := r.InsertBlockWithTransactions(ctx, block, nil); err != nil {
		t.Fatalf("insert block with empty transactions: %v", err)
	}

	var blockCount int64
	if err := db.Model(&models.Block{}).Where("number = ?", 100).Count(&blockCount).Error; err != nil {
		t.Fatalf("count blocks: %v", err)
	}
	if blockCount != 1 {
		t.Fatalf("expected 1 block, got %d", blockCount)
	}

	var txCount int64
	if err := db.Model(&models.Transaction{}).Where("block_number = ?", 100).Count(&txCount).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if txCount != 0 {
		t.Fatalf("expected 0 transactions, got %d", txCount)
	}
}

func TestBlockRepository_InsertBlockWithTransactions_DuplicateIgnored(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	block := newBlockWithTxTestBlock(100, "0xblockhash100")
	txs := []*models.Transaction{
		newBlockWithTxTestTransaction("0xtxhash1", 100, "0xblockhash100", 0),
		newBlockWithTxTestTransaction("0xtxhash2", 100, "0xblockhash100", 1),
		newBlockWithTxTestTransaction("0xtxhash3", 100, "0xblockhash100", 2),
	}

	if err := r.InsertBlockWithTransactions(ctx, block, txs); err != nil {
		t.Fatalf("first insert block with transactions: %v", err)
	}

	if err := r.InsertBlockWithTransactions(ctx, block, txs); err != nil {
		t.Fatalf("second insert block with transactions: %v", err)
	}

	var blockCount int64
	if err := db.Model(&models.Block{}).Where("number = ?", 100).Count(&blockCount).Error; err != nil {
		t.Fatalf("count blocks: %v", err)
	}
	if blockCount != 1 {
		t.Fatalf("expected 1 block, got %d", blockCount)
	}

	var txCount int64
	if err := db.Model(&models.Transaction{}).Where("block_number = ?", 100).Count(&txCount).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if txCount != 3 {
		t.Fatalf("expected 3 transactions, got %d", txCount)
	}
}

func TestBlockRepository_InsertBlockWithTransactions_DBError(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	block := newBlockWithTxTestBlock(100, "0xblockhash100")
	txs := []*models.Transaction{
		newBlockWithTxTestTransaction("0xtxhash1", 100, "0xblockhash100", 0),
	}

	err = r.InsertBlockWithTransactions(ctx, block, txs)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

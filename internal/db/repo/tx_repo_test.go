package repo

import (
	"block-explorer-backend/internal/db/models"
	"context"
	"testing"

	"gorm.io/gorm"
)

func setupTransactionRepo(t *testing.T) (*TransactionRepository, *gorm.DB) {
	t.Helper()

	db := SetupTestDB(t, &models.Transaction{})
	return NewTransactionRepository(db), db
}

func newTestTransaction(hash string, blockNumber uint64, txIndex uint) *models.Transaction {
	return &models.Transaction{
		Hash:        hash,
		BlockNumber: blockNumber,
		BlockHash:   "0xblockhash",
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

func TestTransactionRepository_InsertTransaction_Success(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)

	if err := r.InsertTransaction(ctx, tx); err != nil {
		t.Fatalf("insert transaction: %v", err)
	}

	var got models.Transaction
	if err := db.First(&got, "hash = ?", "0xtxhash1").Error; err != nil {
		t.Fatalf("expected transaction inserted, got error: %v", err)
	}
	if got.BlockNumber != 100 {
		t.Fatalf("expected block number=100, got %d", got.BlockNumber)
	}
	if got.BlockHash != "0xblockhash" {
		t.Fatalf("expected block hash=0xblockhash, got %s", got.BlockHash)
	}
	if got.TxIndex != 0 {
		t.Fatalf("expected tx index=0, got %d", got.TxIndex)
	}
	if got.FromAddress != tx.FromAddress {
		t.Fatalf("expected from address=%s, got %s", tx.FromAddress, got.FromAddress)
	}
	if got.ToAddress != tx.ToAddress {
		t.Fatalf("expected to address=%s, got %s", tx.ToAddress, got.ToAddress)
	}
}

func TestTransactionRepository_InsertTransaction_DuplicateIgnored(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)

	if err := r.InsertTransaction(ctx, tx); err != nil {
		t.Fatalf("first insert transaction: %v", err)
	}

	if err := r.InsertTransaction(ctx, tx); err != nil {
		t.Fatalf("second insert transaction: %v", err)
	}

	var count int64
	if err := db.Model(&models.Transaction{}).Where("hash = ?", "0xtxhash1").Count(&count).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 transaction, got %d", count)
	}
}

func TestTransactionRepository_InsertTransaction_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err = r.InsertTransaction(ctx, newTestTransaction("0xtxhash1", 100, 0))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTransactionRepository_InsertTransactions_Success(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	txs := []*models.Transaction{
		newTestTransaction("0xtxhash1", 100, 0),
		newTestTransaction("0xtxhash2", 100, 1),
		newTestTransaction("0xtxhash3", 100, 2),
	}

	if err := r.InsertTransactions(ctx, txs); err != nil {
		t.Fatalf("insert transactions: %v", err)
	}

	var count int64
	if err := db.Model(&models.Transaction{}).Where("block_number = ?", 100).Count(&count).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}

	if count != 3 {
		t.Fatalf("expected 3 transactions, got %d", count)
	}
}

func TestTransactionRepository_InsertTransactions_EmptyInput(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	if err := r.InsertTransactions(ctx, nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if err := r.InsertTransactions(ctx, []*models.Transaction{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var count int64
	if err := db.Model(&models.Transaction{}).Count(&count).Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 transactions, got %d", count)
	}
}

func TestTransactionRepository_InsertTransactions_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	txs := []*models.Transaction{
		newTestTransaction("0xtxhash1", 100, 0),
		newTestTransaction("0xtxhash2", 100, 1),
	}

	err = r.InsertTransactions(ctx, txs)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTransactionRepository_GetTransactionByHash_Found(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)

	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, found, err := r.GetTransactionByHash(ctx, "0xtxhash1")
	if err != nil {
		t.Fatalf("get transaction by hash: %v", err)
	}

	if !found {
		t.Fatalf("expected found=true, got false")
	}

	if got == nil {
		t.Fatalf("expected transaction, got nil")
	}

	if got.Hash != "0xtxhash1" {
		t.Fatalf("expected hash=0xtxhash1, got %s", got.Hash)
	}

	if got.BlockNumber != 100 {
		t.Fatalf("expected block number=100, got %d", got.BlockNumber)
	}
}

func TestTransactionRepository_GetTransactionByHash_NotFound(t *testing.T) {
	r, _ := setupTransactionRepo(t)
	ctx := context.Background()

	tx, found, err := r.GetTransactionByHash(ctx, "0xmissing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if found {
		t.Fatalf("expected found=false, got true")
	}

	if tx != nil {
		t.Fatalf("expected tx=nil, got %+v", tx)
	}
}

func TestTransactionRepository_GetTransactionByHash_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	tx, found, err := r.GetTransactionByHash(ctx, "0xtxhash1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if found {
		t.Fatalf("expected found=false, got true")
	}

	if tx != nil {
		t.Fatalf("expected tx=nil, got %+v", tx)
	}
}

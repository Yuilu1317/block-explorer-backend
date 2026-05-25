package repo

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func setupTransactionRepo(t *testing.T) (*TransactionRepository, *gorm.DB) {
	t.Helper()

	db := SetupTestDB(t, &models.Transaction{})
	return NewTransactionRepository(db), db
}

func newTestTransaction(hash string, blockNumber uint64, txIndex uint) *models.Transaction {
	from := "0x1111111111111111111111111111111111111111"
	to := "0x2222222222222222222222222222222222222222"

	return &models.Transaction{
		Hash:        hash,
		BlockNumber: blockNumber,
		BlockHash:   "0xblockhash",
		TxIndex:     txIndex,

		FromAddress:      from,
		FromAddressLower: strings.ToLower(from),
		ToAddress:        to,
		ToAddressLower:   strings.ToLower(to),

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
	if got.FromAddressLower != tx.FromAddressLower {
		t.Fatalf("expected from address lower=%s, got %s", tx.FromAddressLower, got.FromAddressLower)
	}

	if got.ToAddressLower != tx.ToAddressLower {
		t.Fatalf("expected to address lower=%s, got %s", tx.ToAddressLower, got.ToAddressLower)
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

func TestTransactionRepository_GetTransactionsByHashes_EmptyInput(t *testing.T) {
	r, _ := setupTransactionRepo(t)
	ctx := context.Background()

	got, err := r.GetTransactionsByHashes(ctx, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if got == nil {
		t.Fatalf("expected empty map, got nil")
	}

	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d items", len(got))
	}

	got, err = r.GetTransactionsByHashes(ctx, []string{})
	if err != nil {
		t.Fatalf("expected nil error for empty slice, got %v", err)
	}

	if got == nil {
		t.Fatalf("expected empty map for empty slice, got nil")
	}

	if len(got) != 0 {
		t.Fatalf("expected empty map for empty slice, got %d items", len(got))
	}
}

func TestTransactionRepository_GetTransactionsByHashes_ReturnsOnlyMatchedTransactions(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx1 := newTestTransaction("0xtxhash1", 100, 0)
	tx2 := newTestTransaction("0xtxhash2", 101, 1)
	tx3 := newTestTransaction("0xtxhash3", 102, 2)

	for _, tx := range []*models.Transaction{tx1, tx2, tx3} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.GetTransactionsByHashes(ctx, []string{
		"0xtxhash1",
		"0xmissing",
		"0xtxhash3",
	})
	if err != nil {
		t.Fatalf("get transactions by hashes: %v", err)
	}

	if got == nil {
		t.Fatalf("expected result map, got nil")
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 matched transactions, got %d", len(got))
	}

	gotTx1, exists := got["0xtxhash1"]
	if !exists {
		t.Fatalf("expected 0xtxhash1 to exist")
	}
	if gotTx1 == nil {
		t.Fatalf("expected 0xtxhash1 transaction, got nil")
	}
	if gotTx1.BlockNumber != 100 {
		t.Fatalf("expected 0xtxhash1 block number=100, got %d", gotTx1.BlockNumber)
	}
	if gotTx1.TxIndex != 0 {
		t.Fatalf("expected 0xtxhash1 tx index=0, got %d", gotTx1.TxIndex)
	}

	gotTx3, exists := got["0xtxhash3"]
	if !exists {
		t.Fatalf("expected 0xtxhash3 to exist")
	}
	if gotTx3 == nil {
		t.Fatalf("expected 0xtxhash3 transaction, got nil")
	}
	if gotTx3.BlockNumber != 102 {
		t.Fatalf("expected 0xtxhash3 block number=102, got %d", gotTx3.BlockNumber)
	}
	if gotTx3.TxIndex != 2 {
		t.Fatalf("expected 0xtxhash3 tx index=2, got %d", gotTx3.TxIndex)
	}

	if _, exists := got["0xmissing"]; exists {
		t.Fatalf("expected 0xmissing not to exist")
	}

	if _, exists := got["0xtxhash2"]; exists {
		t.Fatalf("expected 0xtxhash2 not to exist because it was not requested")
	}
}

func TestTransactionRepository_GetTransactionsByHashes_ReturnsEmptyMapWhenNoHashesMatch(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, err := r.GetTransactionsByHashes(ctx, []string{
		"0xmissing1",
		"0xmissing2",
	})
	if err != nil {
		t.Fatalf("get transactions by hashes: %v", err)
	}

	if got == nil {
		t.Fatalf("expected empty map, got nil")
	}

	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d items", len(got))
	}
}

func TestTransactionRepository_GetTransactionsByHashes_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	got, err := r.GetTransactionsByHashes(ctx, []string{"0xtxhash1"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if got != nil {
		t.Fatalf("expected nil result on error, got %+v", got)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_ReturnsTxsWhenFromAddressMatches(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx1 := newTestTransaction("0xtxhash1", 100, 0)

	tx2 := newTestTransaction("0xtxhash2", 101, 0)
	tx2.FromAddress = "0x3333333333333333333333333333333333333333"
	tx2.FromAddressLower = strings.ToLower(tx2.FromAddress)
	tx2.ToAddress = "0x4444444444444444444444444444444444444444"
	tx2.ToAddressLower = strings.ToLower(tx2.ToAddress)

	if err := db.Create(tx1).Error; err != nil {
		t.Fatalf("seed tx1: %v", err)
	}
	if err := db.Create(tx2).Error; err != nil {
		t.Fatalf("seed tx2: %v", err)
	}

	got, err := r.ListTransactionsByAddress(ctx, tx1.FromAddressLower, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(got))
	}

	if got[0].Hash != "0xtxhash1" {
		t.Fatalf("expected hash=0xtxhash1, got %s", got[0].Hash)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_ReturnsTxsWhenToAddressMatches(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx1 := newTestTransaction("0xtxhash1", 100, 0)

	tx2 := newTestTransaction("0xtxhash2", 101, 0)
	tx2.FromAddress = "0x3333333333333333333333333333333333333333"
	tx2.FromAddressLower = strings.ToLower(tx2.FromAddress)
	tx2.ToAddress = "0x4444444444444444444444444444444444444444"
	tx2.ToAddressLower = strings.ToLower(tx2.ToAddress)

	if err := db.Create(tx1).Error; err != nil {
		t.Fatalf("seed tx1: %v", err)
	}
	if err := db.Create(tx2).Error; err != nil {
		t.Fatalf("seed tx2: %v", err)
	}

	got, err := r.ListTransactionsByAddress(ctx, tx1.ToAddressLower, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(got))
	}

	if got[0].Hash != "0xtxhash1" {
		t.Fatalf("expected hash=0xtxhash1, got %s", got[0].Hash)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_MatchesLowercaseAddressKey(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	checksumAddress := "0x39fA8c5f2793459D6622857E7D9FbB4BD91766d3"
	lowerAddress := strings.ToLower(checksumAddress)

	tx := newTestTransaction("0xtxhash1", 100, 0)
	tx.FromAddress = checksumAddress
	tx.FromAddressLower = lowerAddress

	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, err := r.ListTransactionsByAddress(ctx, lowerAddress, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(got))
	}

	if got[0].Hash != "0xtxhash1" {
		t.Fatalf("expected hash=0xtxhash1, got %s", got[0].Hash)
	}

	if got[0].FromAddress != checksumAddress {
		t.Fatalf("expected display address=%s, got %s", checksumAddress, got[0].FromAddress)
	}

	if got[0].FromAddressLower != lowerAddress {
		t.Fatalf("expected lower address=%s, got %s", lowerAddress, got[0].FromAddressLower)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_OrdersByBlockNumberAndTxIndexDesc(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"

	txs := []*models.Transaction{
		newTestTransaction("0xtxhash1", 100, 1),
		newTestTransaction("0xtxhash2", 101, 0),
		newTestTransaction("0xtxhash3", 101, 2),
		newTestTransaction("0xtxhash4", 99, 9),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsByAddress(ctx, address, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 transactions, got %d", len(got))
	}

	expectedHashes := []string{
		"0xtxhash3", // block 101, tx_index 2
		"0xtxhash2", // block 101, tx_index 0
		"0xtxhash1", // block 100, tx_index 1
		"0xtxhash4", // block 99, tx_index 9
	}

	for i, expectedHash := range expectedHashes {
		if got[i].Hash != expectedHash {
			t.Fatalf("expected got[%d].Hash=%s, got %s", i, expectedHash, got[i].Hash)
		}
	}
}

func TestTransactionRepository_ListTransactionsByAddress_AppliesLimitAndOffset(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"

	txs := []*models.Transaction{
		newTestTransaction("0xtxhash1", 100, 0),
		newTestTransaction("0xtxhash2", 101, 0),
		newTestTransaction("0xtxhash3", 102, 0),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsByAddress(ctx, address, 1, 1)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(got))
	}

	// Sorted order should be:
	// 0xtxhash3 block 102
	// 0xtxhash2 block 101
	// 0xtxhash1 block 100
	//
	// limit=1 offset=1 should return the second item.
	if got[0].Hash != "0xtxhash2" {
		t.Fatalf("expected hash=0xtxhash2, got %s", got[0].Hash)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_ReturnsEmptySliceWhenNoMatch(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)

	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, err := r.ListTransactionsByAddress(
		ctx,
		"0x9999999999999999999999999999999999999999",
		10,
		0,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(got))
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_SuccessStatusOne(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	if err := r.UpdateTransactionReceiptByHash(ctx, "0xtxhash1", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var got models.Transaction
	if err := db.First(&got, "hash = ?", "0xtxhash1").Error; err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if got.ReceiptStatus == nil {
		t.Fatalf("expected receipt status, got nil")
	}
	if *got.ReceiptStatus != uint64(1) {
		t.Fatalf("expected receipt status=1, got %d", *got.ReceiptStatus)
	}

	if got.ReceiptGasUsed == nil {
		t.Fatalf("expected receipt gas used, got nil")
	}
	if *got.ReceiptGasUsed != uint64(21000) {
		t.Fatalf("expected receipt gas used=21000, got %d", *got.ReceiptGasUsed)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_SuccessStatusZero(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(0)
	gasUsed := uint64(21000)

	if err := r.UpdateTransactionReceiptByHash(ctx, "0xtxhash1", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var got models.Transaction
	if err := db.First(&got, "hash = ?", "0xtxhash1").Error; err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if got.ReceiptStatus == nil {
		t.Fatalf("expected receipt status, got nil")
	}
	if *got.ReceiptStatus != uint64(0) {
		t.Fatalf("expected receipt status=0, got %d", *got.ReceiptStatus)
	}

	if got.ReceiptGasUsed == nil {
		t.Fatalf("expected receipt gas used, got nil")
	}
	if *got.ReceiptGasUsed != uint64(21000) {
		t.Fatalf("expected receipt gas used=21000, got %d", *got.ReceiptGasUsed)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_NotFound(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	err := r.UpdateTransactionReceiptByHash(ctx, "0xmissing", &status, &gasUsed)
	if !errors.Is(err, types.ErrTxNotFound) {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}

	var got models.Transaction
	if err := db.First(&got, "hash = ?", "0xtxhash1").Error; err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if got.ReceiptStatus != nil {
		t.Fatalf("expected receipt status nil, got %d", *got.ReceiptStatus)
	}
	if got.ReceiptGasUsed != nil {
		t.Fatalf("expected receipt gas used nil, got %d", *got.ReceiptGasUsed)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_DoesNotModifyTransactionFields(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction("0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	originalBlockNumber := tx.BlockNumber
	originalBlockHash := tx.BlockHash
	originalTxIndex := tx.TxIndex
	originalFromAddress := tx.FromAddress
	originalFromAddressLower := tx.FromAddressLower
	originalToAddress := tx.ToAddress
	originalToAddressLower := tx.ToAddressLower
	originalNonce := tx.Nonce
	originalValueWei := tx.ValueWei
	originalGasLimit := tx.GasLimit
	originalGasPriceWei := tx.GasPriceWei
	originalInputData := tx.InputData

	status := uint64(1)
	gasUsed := uint64(21000)

	if err := r.UpdateTransactionReceiptByHash(ctx, "0xtxhash1", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var got models.Transaction
	if err := db.First(&got, "hash = ?", "0xtxhash1").Error; err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if got.BlockNumber != originalBlockNumber {
		t.Fatalf("expected block number=%d, got %d", originalBlockNumber, got.BlockNumber)
	}
	if got.BlockHash != originalBlockHash {
		t.Fatalf("expected block hash=%s, got %s", originalBlockHash, got.BlockHash)
	}
	if got.TxIndex != originalTxIndex {
		t.Fatalf("expected tx index=%d, got %d", originalTxIndex, got.TxIndex)
	}
	if got.FromAddress != originalFromAddress {
		t.Fatalf("expected from address=%s, got %s", originalFromAddress, got.FromAddress)
	}
	if got.FromAddressLower != originalFromAddressLower {
		t.Fatalf("expected from address lower=%s, got %s", originalFromAddressLower, got.FromAddressLower)
	}
	if got.ToAddress != originalToAddress {
		t.Fatalf("expected to address=%s, got %s", originalToAddress, got.ToAddress)
	}
	if got.ToAddressLower != originalToAddressLower {
		t.Fatalf("expected to address lower=%s, got %s", originalToAddressLower, got.ToAddressLower)
	}
	if got.Nonce != originalNonce {
		t.Fatalf("expected nonce=%d, got %d", originalNonce, got.Nonce)
	}
	if got.ValueWei != originalValueWei {
		t.Fatalf("expected value wei=%s, got %s", originalValueWei, got.ValueWei)
	}
	if got.GasLimit != originalGasLimit {
		t.Fatalf("expected gas limit=%d, got %d", originalGasLimit, got.GasLimit)
	}
	if got.GasPriceWei != originalGasPriceWei {
		t.Fatalf("expected gas price wei=%s, got %s", originalGasPriceWei, got.GasPriceWei)
	}
	if got.InputData != originalInputData {
		t.Fatalf("expected input data=%s, got %s", originalInputData, got.InputData)
	}

	if got.ReceiptStatus == nil {
		t.Fatalf("expected receipt status, got nil")
	}
	if *got.ReceiptStatus != uint64(1) {
		t.Fatalf("expected receipt status=1, got %d", *got.ReceiptStatus)
	}

	if got.ReceiptGasUsed == nil {
		t.Fatalf("expected receipt gas used, got nil")
	}
	if *got.ReceiptGasUsed != uint64(21000) {
		t.Fatalf("expected receipt gas used=21000, got %d", *got.ReceiptGasUsed)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	err = r.UpdateTransactionReceiptByHash(ctx, "0xtxhash1", &status, &gasUsed)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func findTransactionByHash(txs []*models.Transaction, hash string) *models.Transaction {
	for _, tx := range txs {
		if tx.Hash == hash {
			return tx
		}
	}
	return nil
}

func TestTransactionRepository_ListTransactionsMissingReceiptByBlockNumber_FiltersCorrectly(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	statusOne := uint64(1)
	statusZero := uint64(0)
	gasUsed := uint64(21000)

	missingBoth := newTestTransaction("0xmissingboth", 100, 0)
	// ReceiptStatus nil, ReceiptGasUsed nil

	alreadySuccess := newTestTransaction("0xalreadysuccess", 100, 1)
	alreadySuccess.ReceiptStatus = &statusOne
	alreadySuccess.ReceiptGasUsed = &gasUsed

	alreadyFailed := newTestTransaction("0xalreadyfailed", 100, 2)
	alreadyFailed.ReceiptStatus = &statusZero
	alreadyFailed.ReceiptGasUsed = &gasUsed

	missingGasUsed := newTestTransaction("0xmissinggasused", 100, 3)
	missingGasUsed.ReceiptStatus = &statusOne
	// ReceiptGasUsed nil

	missingStatus := newTestTransaction("0xmissingstatus", 100, 4)
	// ReceiptStatus nil
	missingStatus.ReceiptGasUsed = &gasUsed

	otherBlockMissing := newTestTransaction("0xotherblockmissing", 101, 0)
	// ReceiptStatus nil, ReceiptGasUsed nil, but block_number is different

	txs := []*models.Transaction{
		missingBoth,
		alreadySuccess,
		alreadyFailed,
		missingGasUsed,
		missingStatus,
		otherBlockMissing,
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, 100)
	if err != nil {
		t.Fatalf("list transactions missing receipt: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 missing receipt transactions, got %d", len(got))
	}

	gotMissingBoth := findTransactionByHash(got, "0xmissingboth")
	if gotMissingBoth == nil {
		t.Fatalf("expected missing-both transaction to be returned")
	}
	if gotMissingBoth.ReceiptStatus != nil {
		t.Fatalf("expected missing-both receipt status nil, got %d", *gotMissingBoth.ReceiptStatus)
	}
	if gotMissingBoth.ReceiptGasUsed != nil {
		t.Fatalf("expected missing-both receipt gas used nil, got %d", *gotMissingBoth.ReceiptGasUsed)
	}

	gotMissingGasUsed := findTransactionByHash(got, "0xmissinggasused")
	if gotMissingGasUsed == nil {
		t.Fatalf("expected missing-gas-used transaction to be returned")
	}
	if gotMissingGasUsed.ReceiptStatus == nil {
		t.Fatalf("expected missing-gas-used receipt status, got nil")
	}
	if *gotMissingGasUsed.ReceiptStatus != uint64(1) {
		t.Fatalf("expected missing-gas-used receipt status=1, got %d", *gotMissingGasUsed.ReceiptStatus)
	}
	if gotMissingGasUsed.ReceiptGasUsed != nil {
		t.Fatalf("expected missing-gas-used receipt gas used nil, got %d", *gotMissingGasUsed.ReceiptGasUsed)
	}

	gotMissingStatus := findTransactionByHash(got, "0xmissingstatus")
	if gotMissingStatus == nil {
		t.Fatalf("expected missing-status transaction to be returned")
	}
	if gotMissingStatus.ReceiptStatus != nil {
		t.Fatalf("expected missing-status receipt status nil, got %d", *gotMissingStatus.ReceiptStatus)
	}
	if gotMissingStatus.ReceiptGasUsed == nil {
		t.Fatalf("expected missing-status receipt gas used, got nil")
	}
	if *gotMissingStatus.ReceiptGasUsed != uint64(21000) {
		t.Fatalf("expected missing-status receipt gas used=21000, got %d", *gotMissingStatus.ReceiptGasUsed)
	}

	if findTransactionByHash(got, "0xalreadysuccess") != nil {
		t.Fatalf("expected already-success transaction not to be returned")
	}

	if findTransactionByHash(got, "0xalreadyfailed") != nil {
		t.Fatalf("expected already-failed transaction not to be returned")
	}

	if findTransactionByHash(got, "0xotherblockmissing") != nil {
		t.Fatalf("expected other-block missing transaction not to be returned")
	}
}

func TestTransactionRepository_ListTransactionsMissingReceiptByBlockNumber_OrdersByTxIndexAsc(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	txs := []*models.Transaction{
		newTestTransaction("0xtxhash2", 100, 2),
		newTestTransaction("0xtxhash0", 100, 0),
		newTestTransaction("0xtxhash1", 100, 1),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, 100)
	if err != nil {
		t.Fatalf("list transactions missing receipt: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(got))
	}

	expectedHashes := []string{
		"0xtxhash0",
		"0xtxhash1",
		"0xtxhash2",
	}

	for i, expectedHash := range expectedHashes {
		if got[i].Hash != expectedHash {
			t.Fatalf("expected got[%d].Hash=%s, got %s", i, expectedHash, got[i].Hash)
		}
	}
}

func TestTransactionRepository_ListTransactionsMissingReceiptByBlockNumber_ReturnsEmptySliceWhenNoMissingReceipt(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	statusOne := uint64(1)
	statusZero := uint64(0)
	gasUsed := uint64(21000)

	successTx := newTestTransaction("0xsuccess", 100, 0)
	successTx.ReceiptStatus = &statusOne
	successTx.ReceiptGasUsed = &gasUsed

	failedTx := newTestTransaction("0xfailed", 100, 1)
	failedTx.ReceiptStatus = &statusZero
	failedTx.ReceiptGasUsed = &gasUsed

	txs := []*models.Transaction{
		successTx,
		failedTx,
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, 100)
	if err != nil {
		t.Fatalf("list transactions missing receipt: %v", err)
	}

	if got == nil {
		t.Fatalf("expected empty slice, got nil")
	}

	if len(got) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(got))
	}
}

func TestTransactionRepository_ListTransactionsMissingReceiptByBlockNumber_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, 100)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if got != nil {
		t.Fatalf("expected nil transactions, got %+v", got)
	}
}

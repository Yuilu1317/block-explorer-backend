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

const (
	txRepoTestChainID      int64 = 11155111
	txRepoOtherTestChainID int64 = 1

	txRepoFromAddressLower = "0x1111111111111111111111111111111111111111"
	txRepoToAddressLower   = "0x2222222222222222222222222222222222222222"
)

func setupTransactionRepo(t *testing.T) (*TransactionRepository, *gorm.DB) {
	t.Helper()

	db := SetupTestDB(t, &models.Transaction{})
	return NewTransactionRepository(db), db
}

func newTestTransaction(
	chainID int64,
	hash string,
	blockNumber uint64,
	txIndex uint,
) *models.Transaction {
	return newTestTransactionWithAddresses(
		chainID,
		hash,
		blockNumber,
		txIndex,
		txRepoFromAddressLower,
		txRepoToAddressLower,
	)
}

func newTestTransactionWithAddresses(
	chainID int64,
	hash string,
	blockNumber uint64,
	txIndex uint,
	fromAddressLower string,
	toAddressLower string,
) *models.Transaction {
	return &models.Transaction{
		ChainID: chainID,

		Hash:        hash,
		BlockNumber: blockNumber,
		BlockHash:   "0xblockhash",
		TxIndex:     txIndex,

		FromAddress:      fromAddressLower,
		FromAddressLower: strings.ToLower(fromAddressLower),

		ToAddress:      toAddressLower,
		ToAddressLower: strings.ToLower(toAddressLower),

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

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)

	if err := r.InsertTransaction(ctx, tx); err != nil {
		t.Fatalf("insert transaction: %v", err)
	}

	var got models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xtxhash1").First(&got).Error; err != nil {
		t.Fatalf("expected transaction inserted, got error: %v", err)
	}

	if got.ChainID != txRepoTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", txRepoTestChainID, got.ChainID)
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

func TestTransactionRepository_InsertTransaction_NilReturnsError(t *testing.T) {
	r, _ := setupTransactionRepo(t)

	err := r.InsertTransaction(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTransactionRepository_InsertTransaction_DuplicateIgnored(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx1 := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	tx2 := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)

	if err := r.InsertTransaction(ctx, tx1); err != nil {
		t.Fatalf("first insert transaction: %v", err)
	}

	if err := r.InsertTransaction(ctx, tx2); err != nil {
		t.Fatalf("second insert transaction: %v", err)
	}

	var count int64
	if err := db.Model(&models.Transaction{}).
		Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xtxhash1").
		Count(&count).
		Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 transaction, got %d", count)
	}
}

func TestTransactionRepository_InsertTransaction_AllowsSameHashOnDifferentChains(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx1 := newTestTransaction(txRepoTestChainID, "0xsamehash", 100, 0)
	tx2 := newTestTransaction(txRepoOtherTestChainID, "0xsamehash", 200, 0)

	if err := r.InsertTransaction(ctx, tx1); err != nil {
		t.Fatalf("insert target chain transaction: %v", err)
	}

	if err := r.InsertTransaction(ctx, tx2); err != nil {
		t.Fatalf("insert other chain transaction: %v", err)
	}

	var count int64
	if err := db.Model(&models.Transaction{}).
		Where("hash = ?", "0xsamehash").
		Count(&count).
		Error; err != nil {
		t.Fatalf("count transactions: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected 2 transactions with same hash on different chains, got %d", count)
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

	err = r.InsertTransaction(ctx, newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTransactionRepository_InsertTransactions_Success(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	txs := []*models.Transaction{
		newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash2", 100, 1),
		newTestTransaction(txRepoTestChainID, "0xtxhash3", 100, 2),
	}

	if err := r.InsertTransactions(ctx, txs); err != nil {
		t.Fatalf("insert transactions: %v", err)
	}

	var count int64
	if err := db.Model(&models.Transaction{}).
		Where("chain_id = ? AND block_number = ?", txRepoTestChainID, 100).
		Count(&count).
		Error; err != nil {
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
		newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash2", 100, 1),
	}

	err = r.InsertTransactions(ctx, txs)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTransactionRepository_GetTransactionByHash_Found(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)

	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xtxhash1")
	if err != nil {
		t.Fatalf("get transaction by hash: %v", err)
	}

	if !found {
		t.Fatalf("expected found=true, got false")
	}

	if got == nil {
		t.Fatalf("expected transaction, got nil")
	}

	if got.ChainID != txRepoTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", txRepoTestChainID, got.ChainID)
	}

	if got.Hash != "0xtxhash1" {
		t.Fatalf("expected hash=0xtxhash1, got %s", got.Hash)
	}

	if got.BlockNumber != 100 {
		t.Fatalf("expected block number=100, got %d", got.BlockNumber)
	}
}

func TestTransactionRepository_GetTransactionByHash_FiltersByChainID(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	targetTx := newTestTransaction(txRepoTestChainID, "0xsamehash", 100, 0)
	otherTx := newTestTransaction(txRepoOtherTestChainID, "0xsamehash", 200, 0)

	if err := db.Create(targetTx).Error; err != nil {
		t.Fatalf("seed target chain transaction: %v", err)
	}
	if err := db.Create(otherTx).Error; err != nil {
		t.Fatalf("seed other chain transaction: %v", err)
	}

	got, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xsamehash")
	if err != nil {
		t.Fatalf("get transaction by hash: %v", err)
	}

	if !found {
		t.Fatalf("expected found=true, got false")
	}

	if got == nil {
		t.Fatalf("expected transaction, got nil")
	}

	if got.ChainID != txRepoTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", txRepoTestChainID, got.ChainID)
	}

	if got.BlockNumber != 100 {
		t.Fatalf("expected target chain block number=100, got %d", got.BlockNumber)
	}
}

func TestTransactionRepository_GetTransactionByHash_PreservesReceiptStatusNilZeroOne(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	statusZero := uint64(0)
	statusOne := uint64(1)
	gasUsed := uint64(21000)

	pendingTx := newTestTransaction(txRepoTestChainID, "0xpending", 100, 0)

	failedTx := newTestTransaction(txRepoTestChainID, "0xfailed", 101, 0)
	failedTx.ReceiptStatus = &statusZero
	failedTx.ReceiptGasUsed = &gasUsed

	successTx := newTestTransaction(txRepoTestChainID, "0xsuccess", 102, 0)
	successTx.ReceiptStatus = &statusOne
	successTx.ReceiptGasUsed = &gasUsed

	for _, tx := range []*models.Transaction{pendingTx, failedTx, successTx} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	gotPending, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xpending")
	if err != nil {
		t.Fatalf("get pending transaction by hash: %v", err)
	}
	if !found {
		t.Fatalf("expected pending transaction found=true, got false")
	}
	if gotPending == nil {
		t.Fatalf("expected pending transaction, got nil")
	}
	if gotPending.ReceiptStatus != nil {
		t.Fatalf("expected pending receipt status nil, got %d", *gotPending.ReceiptStatus)
	}
	if gotPending.ReceiptGasUsed != nil {
		t.Fatalf("expected pending receipt gas used nil, got %d", *gotPending.ReceiptGasUsed)
	}

	gotFailed, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xfailed")
	if err != nil {
		t.Fatalf("get failed transaction by hash: %v", err)
	}
	if !found {
		t.Fatalf("expected failed transaction found=true, got false")
	}
	if gotFailed == nil {
		t.Fatalf("expected failed transaction, got nil")
	}
	if gotFailed.ReceiptStatus == nil {
		t.Fatalf("expected failed receipt status, got nil")
	}
	if *gotFailed.ReceiptStatus != uint64(0) {
		t.Fatalf("expected failed receipt status=0, got %d", *gotFailed.ReceiptStatus)
	}
	if gotFailed.ReceiptGasUsed == nil {
		t.Fatalf("expected failed receipt gas used, got nil")
	}
	if *gotFailed.ReceiptGasUsed != gasUsed {
		t.Fatalf("expected failed receipt gas used=%d, got %d", gasUsed, *gotFailed.ReceiptGasUsed)
	}

	gotSuccess, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xsuccess")
	if err != nil {
		t.Fatalf("get success transaction by hash: %v", err)
	}
	if !found {
		t.Fatalf("expected success transaction found=true, got false")
	}
	if gotSuccess == nil {
		t.Fatalf("expected success transaction, got nil")
	}
	if gotSuccess.ReceiptStatus == nil {
		t.Fatalf("expected success receipt status, got nil")
	}
	if *gotSuccess.ReceiptStatus != uint64(1) {
		t.Fatalf("expected success receipt status=1, got %d", *gotSuccess.ReceiptStatus)
	}
	if gotSuccess.ReceiptGasUsed == nil {
		t.Fatalf("expected success receipt gas used, got nil")
	}
	if *gotSuccess.ReceiptGasUsed != gasUsed {
		t.Fatalf("expected success receipt gas used=%d, got %d", gasUsed, *gotSuccess.ReceiptGasUsed)
	}
}

func TestTransactionRepository_GetTransactionByHash_NotFound(t *testing.T) {
	r, _ := setupTransactionRepo(t)
	ctx := context.Background()

	tx, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xmissing")
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

	tx, found, err := r.GetTransactionByHash(ctx, txRepoTestChainID, "0xtxhash1")
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

	got, err := r.GetTransactionsByHashes(ctx, txRepoTestChainID, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if got == nil {
		t.Fatalf("expected empty map, got nil")
	}

	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d items", len(got))
	}

	got, err = r.GetTransactionsByHashes(ctx, txRepoTestChainID, []string{})
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

func TestTransactionRepository_GetTransactionsByHashes_ReturnsOnlyMatchedTransactionsForChain(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	txs := []*models.Transaction{
		newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash2", 101, 1),
		newTestTransaction(txRepoTestChainID, "0xtxhash3", 102, 2),
		newTestTransaction(txRepoOtherTestChainID, "0xtxhash1", 200, 0),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.GetTransactionsByHashes(ctx, txRepoTestChainID, []string{
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
	if gotTx1.ChainID != txRepoTestChainID {
		t.Fatalf("expected 0xtxhash1 chain_id=%d, got %d", txRepoTestChainID, gotTx1.ChainID)
	}
	if gotTx1.BlockNumber != 100 {
		t.Fatalf("expected 0xtxhash1 block number=100, got %d", gotTx1.BlockNumber)
	}

	gotTx3, exists := got["0xtxhash3"]
	if !exists {
		t.Fatalf("expected 0xtxhash3 to exist")
	}
	if gotTx3 == nil {
		t.Fatalf("expected 0xtxhash3 transaction, got nil")
	}
	if gotTx3.ChainID != txRepoTestChainID {
		t.Fatalf("expected 0xtxhash3 chain_id=%d, got %d", txRepoTestChainID, gotTx3.ChainID)
	}
	if gotTx3.BlockNumber != 102 {
		t.Fatalf("expected 0xtxhash3 block number=102, got %d", gotTx3.BlockNumber)
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

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, err := r.GetTransactionsByHashes(ctx, txRepoTestChainID, []string{
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

	got, err := r.GetTransactionsByHashes(ctx, txRepoTestChainID, []string{"0xtxhash1"})
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

	tx1 := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)

	tx2 := newTestTransaction(txRepoTestChainID, "0xtxhash2", 101, 0)
	tx2.FromAddress = "0x3333333333333333333333333333333333333333"
	tx2.FromAddressLower = strings.ToLower(tx2.FromAddress)
	tx2.ToAddress = "0x4444444444444444444444444444444444444444"
	tx2.ToAddressLower = strings.ToLower(tx2.ToAddress)

	tx3 := newTestTransaction(txRepoOtherTestChainID, "0xtxhash3", 102, 0)

	if err := db.Create(tx1).Error; err != nil {
		t.Fatalf("seed tx1: %v", err)
	}
	if err := db.Create(tx2).Error; err != nil {
		t.Fatalf("seed tx2: %v", err)
	}
	if err := db.Create(tx3).Error; err != nil {
		t.Fatalf("seed tx3: %v", err)
	}

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, tx1.FromAddressLower, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(got))
	}

	if got[0].Hash != "0xtxhash1" {
		t.Fatalf("expected hash=0xtxhash1, got %s", got[0].Hash)
	}

	if got[0].ChainID != txRepoTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", txRepoTestChainID, got[0].ChainID)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_ReturnsTxsWhenToAddressMatches(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx1 := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)

	tx2 := newTestTransaction(txRepoTestChainID, "0xtxhash2", 101, 0)
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

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, tx1.ToAddressLower, 10, 0)
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

func findTransactionEntityByHash(txs []models.Transaction, hash string) *models.Transaction {
	for i := range txs {
		if txs[i].Hash == hash {
			return &txs[i]
		}
	}
	return nil
}

func TestTransactionRepository_ListTransactionsByAddress_PreservesReceiptStatusNilZeroOne(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	address := txRepoFromAddressLower

	statusZero := uint64(0)
	statusOne := uint64(1)
	gasUsed := uint64(21000)

	pendingTx := newTestTransaction(txRepoTestChainID, "0xpending", 100, 0)

	failedTx := newTestTransaction(txRepoTestChainID, "0xfailed", 101, 0)
	failedTx.ReceiptStatus = &statusZero
	failedTx.ReceiptGasUsed = &gasUsed

	successTx := newTestTransaction(txRepoTestChainID, "0xsuccess", 102, 0)
	successTx.ReceiptStatus = &statusOne
	successTx.ReceiptGasUsed = &gasUsed

	for _, tx := range []*models.Transaction{pendingTx, failedTx, successTx} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, address, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(got))
	}

	gotPending := findTransactionEntityByHash(got, "0xpending")
	if gotPending == nil {
		t.Fatalf("expected pending transaction to be returned")
	}
	if gotPending.ReceiptStatus != nil {
		t.Fatalf("expected pending receipt status nil, got %d", *gotPending.ReceiptStatus)
	}
	if gotPending.ReceiptGasUsed != nil {
		t.Fatalf("expected pending receipt gas used nil, got %d", *gotPending.ReceiptGasUsed)
	}

	gotFailed := findTransactionEntityByHash(got, "0xfailed")
	if gotFailed == nil {
		t.Fatalf("expected failed transaction to be returned")
	}
	if gotFailed.ReceiptStatus == nil {
		t.Fatalf("expected failed receipt status, got nil")
	}
	if *gotFailed.ReceiptStatus != uint64(0) {
		t.Fatalf("expected failed receipt status=0, got %d", *gotFailed.ReceiptStatus)
	}
	if gotFailed.ReceiptGasUsed == nil {
		t.Fatalf("expected failed receipt gas used, got nil")
	}
	if *gotFailed.ReceiptGasUsed != gasUsed {
		t.Fatalf("expected failed receipt gas used=%d, got %d", gasUsed, *gotFailed.ReceiptGasUsed)
	}

	gotSuccess := findTransactionEntityByHash(got, "0xsuccess")
	if gotSuccess == nil {
		t.Fatalf("expected success transaction to be returned")
	}
	if gotSuccess.ReceiptStatus == nil {
		t.Fatalf("expected success receipt status, got nil")
	}
	if *gotSuccess.ReceiptStatus != uint64(1) {
		t.Fatalf("expected success receipt status=1, got %d", *gotSuccess.ReceiptStatus)
	}
	if gotSuccess.ReceiptGasUsed == nil {
		t.Fatalf("expected success receipt gas used, got nil")
	}
	if *gotSuccess.ReceiptGasUsed != gasUsed {
		t.Fatalf("expected success receipt gas used=%d, got %d", gasUsed, *gotSuccess.ReceiptGasUsed)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_MatchesLowercaseAddressKey(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	checksumAddress := "0x39fA8c5f2793459D6622857E7D9FbB4BD91766d3"
	lowerAddress := strings.ToLower(checksumAddress)

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	tx.FromAddress = checksumAddress
	tx.FromAddressLower = lowerAddress

	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, lowerAddress, 10, 0)
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

	address := txRepoFromAddressLower

	txs := []*models.Transaction{
		newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 1),
		newTestTransaction(txRepoTestChainID, "0xtxhash2", 101, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash3", 101, 2),
		newTestTransaction(txRepoTestChainID, "0xtxhash4", 99, 9),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, address, 10, 0)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 transactions, got %d", len(got))
	}

	expectedHashes := []string{
		"0xtxhash3",
		"0xtxhash2",
		"0xtxhash1",
		"0xtxhash4",
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

	address := txRepoFromAddressLower

	txs := []*models.Transaction{
		newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash2", 101, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash3", 102, 0),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, address, 1, 1)
	if err != nil {
		t.Fatalf("list transactions by address: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(got))
	}

	if got[0].Hash != "0xtxhash2" {
		t.Fatalf("expected hash=0xtxhash2, got %s", got[0].Hash)
	}
}

func TestTransactionRepository_ListTransactionsByAddress_ReturnsEmptySliceWhenNoMatch(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)

	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	got, err := r.ListTransactionsByAddress(
		ctx,
		txRepoTestChainID,
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

func TestTransactionRepository_ListTransactionsByAddress_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	got, err := r.ListTransactionsByAddress(ctx, txRepoTestChainID, txRepoFromAddressLower, 10, 0)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if got != nil {
		t.Fatalf("expected nil result on error, got %+v", got)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_SuccessStatusOne(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	if err := r.UpdateTransactionReceiptByHash(ctx, txRepoTestChainID, "0xtxhash1", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var got models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xtxhash1").First(&got).Error; err != nil {
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

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(0)
	gasUsed := uint64(21000)

	if err := r.UpdateTransactionReceiptByHash(ctx, txRepoTestChainID, "0xtxhash1", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var got models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xtxhash1").First(&got).Error; err != nil {
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

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	err := r.UpdateTransactionReceiptByHash(ctx, txRepoTestChainID, "0xmissing", &status, &gasUsed)
	if !errors.Is(err, types.ErrTxNotFound) {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}

	var got models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xtxhash1").First(&got).Error; err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if got.ReceiptStatus != nil {
		t.Fatalf("expected receipt status nil, got %d", *got.ReceiptStatus)
	}
	if got.ReceiptGasUsed != nil {
		t.Fatalf("expected receipt gas used nil, got %d", *got.ReceiptGasUsed)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_NotFoundForDifferentChain(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction(txRepoTestChainID, "0xsamehash", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	err := r.UpdateTransactionReceiptByHash(ctx, txRepoOtherTestChainID, "0xsamehash", &status, &gasUsed)
	if !errors.Is(err, types.ErrTxNotFound) {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_DoesNotModifyOtherChain(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	targetTx := newTestTransaction(txRepoTestChainID, "0xsamehash", 100, 0)
	otherTx := newTestTransaction(txRepoOtherTestChainID, "0xsamehash", 200, 0)

	if err := db.Create(targetTx).Error; err != nil {
		t.Fatalf("seed target transaction: %v", err)
	}
	if err := db.Create(otherTx).Error; err != nil {
		t.Fatalf("seed other transaction: %v", err)
	}

	status := uint64(1)
	gasUsed := uint64(21000)

	if err := r.UpdateTransactionReceiptByHash(ctx, txRepoTestChainID, "0xsamehash", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var targetGot models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xsamehash").First(&targetGot).Error; err != nil {
		t.Fatalf("get target transaction: %v", err)
	}

	if targetGot.ReceiptStatus == nil || *targetGot.ReceiptStatus != 1 {
		t.Fatalf("expected target receipt_status=1, got %+v", targetGot.ReceiptStatus)
	}

	var otherGot models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoOtherTestChainID, "0xsamehash").First(&otherGot).Error; err != nil {
		t.Fatalf("get other transaction: %v", err)
	}

	if otherGot.ReceiptStatus != nil {
		t.Fatalf("expected other chain receipt_status=nil, got %d", *otherGot.ReceiptStatus)
	}
	if otherGot.ReceiptGasUsed != nil {
		t.Fatalf("expected other chain receipt_gas_used=nil, got %d", *otherGot.ReceiptGasUsed)
	}
}

func TestTransactionRepository_UpdateTransactionReceiptByHash_DoesNotModifyTransactionFields(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	originalChainID := tx.ChainID
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

	if err := r.UpdateTransactionReceiptByHash(ctx, txRepoTestChainID, "0xtxhash1", &status, &gasUsed); err != nil {
		t.Fatalf("update transaction receipt: %v", err)
	}

	var got models.Transaction
	if err := db.Where("chain_id = ? AND hash = ?", txRepoTestChainID, "0xtxhash1").First(&got).Error; err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if got.ChainID != originalChainID {
		t.Fatalf("expected chain_id=%d, got %d", originalChainID, got.ChainID)
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

	err = r.UpdateTransactionReceiptByHash(ctx, txRepoTestChainID, "0xtxhash1", &status, &gasUsed)
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

	missingBoth := newTestTransaction(txRepoTestChainID, "0xmissingboth", 100, 0)

	alreadySuccess := newTestTransaction(txRepoTestChainID, "0xalreadysuccess", 100, 1)
	alreadySuccess.ReceiptStatus = &statusOne
	alreadySuccess.ReceiptGasUsed = &gasUsed

	alreadyFailed := newTestTransaction(txRepoTestChainID, "0xalreadyfailed", 100, 2)
	alreadyFailed.ReceiptStatus = &statusZero
	alreadyFailed.ReceiptGasUsed = &gasUsed

	missingGasUsed := newTestTransaction(txRepoTestChainID, "0xmissinggasused", 100, 3)
	missingGasUsed.ReceiptStatus = &statusOne

	missingStatus := newTestTransaction(txRepoTestChainID, "0xmissingstatus", 100, 4)
	missingStatus.ReceiptGasUsed = &gasUsed

	otherBlockMissing := newTestTransaction(txRepoTestChainID, "0xotherblockmissing", 101, 0)

	otherChainMissing := newTestTransaction(txRepoOtherTestChainID, "0xotherchainmissing", 100, 0)

	for _, tx := range []*models.Transaction{
		missingBoth,
		alreadySuccess,
		alreadyFailed,
		missingGasUsed,
		missingStatus,
		otherBlockMissing,
		otherChainMissing,
	} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, txRepoTestChainID, 100)
	if err != nil {
		t.Fatalf("list transactions missing receipt: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 missing receipt transactions, got %d", len(got))
	}

	if findTransactionByHash(got, "0xmissingboth") == nil {
		t.Fatalf("expected missing-both transaction to be returned")
	}
	if findTransactionByHash(got, "0xmissinggasused") == nil {
		t.Fatalf("expected missing-gas-used transaction to be returned")
	}
	if findTransactionByHash(got, "0xmissingstatus") == nil {
		t.Fatalf("expected missing-status transaction to be returned")
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
	if findTransactionByHash(got, "0xotherchainmissing") != nil {
		t.Fatalf("expected other-chain missing transaction not to be returned")
	}
}

func TestTransactionRepository_ListTransactionsMissingReceiptByBlockNumber_OrdersByTxIndexAsc(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	txs := []*models.Transaction{
		newTestTransaction(txRepoTestChainID, "0xtxhash2", 100, 2),
		newTestTransaction(txRepoTestChainID, "0xtxhash0", 100, 0),
		newTestTransaction(txRepoTestChainID, "0xtxhash1", 100, 1),
	}

	for _, tx := range txs {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, txRepoTestChainID, 100)
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

	successTx := newTestTransaction(txRepoTestChainID, "0xsuccess", 100, 0)
	successTx.ReceiptStatus = &statusOne
	successTx.ReceiptGasUsed = &gasUsed

	failedTx := newTestTransaction(txRepoTestChainID, "0xfailed", 100, 1)
	failedTx.ReceiptStatus = &statusZero
	failedTx.ReceiptGasUsed = &gasUsed

	for _, tx := range []*models.Transaction{successTx, failedTx} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, txRepoTestChainID, 100)
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

	got, err := r.ListTransactionsMissingReceiptByBlockNumber(ctx, txRepoTestChainID, 100)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if got != nil {
		t.Fatalf("expected nil transactions, got %+v", got)
	}
}

func TestTransactionRepository_ListWalletCompletedTransactionRows_EmptyBlockNumbersReturnsEmptySlice(t *testing.T) {
	r, _ := setupTransactionRepo(t)
	ctx := context.Background()

	txs, err := r.ListWalletCompletedTransactionRows(ctx, txRepoTestChainID, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if txs == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txs))
	}

	txs, err = r.ListWalletCompletedTransactionRows(ctx, txRepoTestChainID, []uint64{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if txs == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txs))
	}
}

func TestTransactionRepository_ListWalletCompletedTransactionRows_ReturnsTransactionsByBlockNumbersAndChain(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	statusZero := uint64(0)
	statusOne := uint64(1)

	tx101Index2 := newTestTransaction(txRepoTestChainID, "0xwallet-tx-101-2", 101, 2)
	tx101Index2.ReceiptStatus = &statusOne

	tx100Index2 := newTestTransaction(txRepoTestChainID, "0xwallet-tx-100-2", 100, 2)
	tx100Index2.ReceiptStatus = &statusZero

	tx100Index1 := newTestTransaction(txRepoTestChainID, "0xwallet-tx-100-1", 100, 1)
	tx100Index1.ReceiptStatus = &statusOne

	tx102Index1 := newTestTransaction(txRepoTestChainID, "0xwallet-tx-102-1", 102, 1)
	tx102Index1.ReceiptStatus = &statusOne

	otherChainTx := newTestTransaction(txRepoOtherTestChainID, "0xwallet-other-chain-100-1", 100, 1)
	otherChainTx.ReceiptStatus = &statusOne

	for _, tx := range []*models.Transaction{
		tx101Index2,
		tx100Index2,
		tx100Index1,
		tx102Index1,
		otherChainTx,
	} {
		if err := db.Create(tx).Error; err != nil {
			t.Fatalf("seed transaction %s: %v", tx.Hash, err)
		}
	}

	txs, err := r.ListWalletCompletedTransactionRows(ctx, txRepoTestChainID, []uint64{100, 101})
	if err != nil {
		t.Fatalf("list wallet completed transaction rows: %v", err)
	}

	if len(txs) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(txs))
	}

	expectedHashes := []string{
		"0xwallet-tx-100-1",
		"0xwallet-tx-100-2",
		"0xwallet-tx-101-2",
	}

	for i, expectedHash := range expectedHashes {
		if txs[i].Hash != expectedHash {
			t.Fatalf("expected txs[%d].Hash=%s, got %s", i, expectedHash, txs[i].Hash)
		}
		if txs[i].ChainID != txRepoTestChainID {
			t.Fatalf("expected txs[%d].ChainID=%d, got %d", i, txRepoTestChainID, txs[i].ChainID)
		}
	}

	if txs[1].ReceiptStatus == nil {
		t.Fatal("expected receipt_status=0 transaction to be returned, got nil receipt status")
	}

	if *txs[1].ReceiptStatus != 0 {
		t.Fatalf("expected receipt_status=0 to be returned, got %d", *txs[1].ReceiptStatus)
	}

	for _, tx := range txs {
		if tx.BlockNumber == 102 {
			t.Fatalf("expected block 102 transaction not to be returned, got %s", tx.Hash)
		}
		if tx.ChainID == txRepoOtherTestChainID {
			t.Fatalf("expected other chain transaction not to be returned, got %s", tx.Hash)
		}
	}
}

func TestTransactionRepository_ListWalletCompletedTransactionRows_ReturnsEmptyWhenNoBlockMatches(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	tx := newTestTransaction(txRepoTestChainID, "0xwallet-tx-100", 100, 0)
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	txs, err := r.ListWalletCompletedTransactionRows(ctx, txRepoTestChainID, []uint64{999})
	if err != nil {
		t.Fatalf("list wallet completed transaction rows: %v", err)
	}

	if len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txs))
	}
}

func TestTransactionRepository_ListWalletCompletedTransactionRows_DBError(t *testing.T) {
	r, db := setupTransactionRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	txs, err := r.ListWalletCompletedTransactionRows(ctx, txRepoTestChainID, []uint64{100})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if txs != nil {
		t.Fatalf("expected nil transactions, got %+v", txs)
	}
}

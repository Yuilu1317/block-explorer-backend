package repo

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestBlockRepository_GetLatestFullySyncedBlockNumber_EmptyTable(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	number, found, err := r.GetLatestFullySyncedBlockNumber(context.Background())
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

func TestBlockRepository_GetLatestFullySyncedBlockNumber_WithBlocks(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	err := db.Create(&models.Block{
		Number:             25,
		Hash:               "0xabc",
		TransactionsSynced: true,
		ReceiptsSynced:     true,
		SyncStatus:         models.BlockSyncStatusCompleted,
	}).Error
	if err != nil {
		t.Fatalf("seed block: %v", err)
	}

	number, found, err := r.GetLatestFullySyncedBlockNumber(context.Background())
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

func TestBlockRepository_GetLatestFullySyncedBlockNumber_IgnoresHigherPartiallySyncedBlock(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	err := db.Create(&[]models.Block{
		{
			Number:             40,
			Hash:               "0xzxc",
			TransactionsSynced: true,
			ReceiptsSynced:     true,
			SyncStatus:         models.BlockSyncStatusCompleted,
		},
		{
			Number:             50,
			Hash:               "0xghi",
			TransactionsSynced: true,
			ReceiptsSynced:     false,
			SyncStatus:         models.BlockSyncStatusReceiptsFailed,
		},
		{
			Number:             30,
			Hash:               "0xdef",
			TransactionsSynced: true,
			ReceiptsSynced:     true,
			SyncStatus:         models.BlockSyncStatusCompleted,
		},
	}).Error
	if err != nil {
		t.Fatalf("seed block: %v", err)
	}

	number, found, err := r.GetLatestFullySyncedBlockNumber(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !found {
		t.Fatalf("expected found=true, got false")
	}
	if number != 40 {
		t.Fatalf("expected number=40, got %d", number)
	}
}

func TestBlockRepository_GetLatestFullySyncedBlockNumber_ExcludesPartiallySyncedBlock(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)

	err := db.Create(&models.Block{
		Number:             25,
		Hash:               "0xabc",
		TransactionsSynced: true,
		ReceiptsSynced:     false,
		SyncStatus:         models.BlockSyncStatusReceiptsFailed,
	}).Error
	if err != nil {
		t.Fatalf("seed block: %v", err)
	}

	number, found, err := r.GetLatestFullySyncedBlockNumber(context.Background())
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

		TransactionsSynced: true,
		ReceiptsSynced:     false,
		SyncStatus:         models.BlockSyncStatusTransactionsSynced,
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

	var savedBlock models.Block
	if err := db.Where("number = ?", block.Number).First(&savedBlock).Error; err != nil {
		t.Fatalf("find saved block: %v", err)
	}

	if !savedBlock.TransactionsSynced {
		t.Fatalf("expected transactions_synced=true")
	}
	if savedBlock.ReceiptsSynced {
		t.Fatalf("expected receipts_synced=false before receipt sync")
	}
	if savedBlock.SyncStatus != models.BlockSyncStatusTransactionsSynced {
		t.Fatalf("expected sync_status=%s, got %s",
			models.BlockSyncStatusTransactionsSynced,
			savedBlock.SyncStatus,
		)
	}
	if savedBlock.LastSyncError != nil {
		t.Fatalf("expected last_sync_error=nil, got %v", *savedBlock.LastSyncError)
	}

	for _, tx := range txs {
		var savedTx models.Transaction
		if err := db.Where("hash = ?", tx.Hash).First(&savedTx).Error; err != nil {
			t.Fatalf("find saved tx %s: %v", tx.Hash, err)
		}

		if savedTx.ReceiptStatus != nil {
			t.Fatalf("expected receipt_status=nil before receipt sync, got %v", *savedTx.ReceiptStatus)
		}
		if savedTx.ReceiptGasUsed != nil {
			t.Fatalf("expected receipt_gas_used=nil before receipt sync, got %v", *savedTx.ReceiptGasUsed)
		}
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

func stringPtr(s string) *string {
	return &s
}

func TestBlockRepository_MarkBlockReceiptsSynced_Success(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	block := newBlockWithTxTestBlock(100, "0xblockhash100")
	block.ReceiptsSynced = false
	block.SyncStatus = models.BlockSyncStatusReceiptsFailed
	block.LastSyncError = stringPtr("previous receipt sync error")

	if err := db.Create(block).Error; err != nil {
		t.Fatalf("seed block: %v", err)
	}

	if err := r.MarkBlockReceiptsSynced(ctx, 100); err != nil {
		t.Fatalf("mark block receipts synced: %v", err)
	}

	var savedBlock models.Block
	if err := db.Where("number = ?", 100).First(&savedBlock).Error; err != nil {
		t.Fatalf("find saved block: %v", err)
	}

	if !savedBlock.TransactionsSynced {
		t.Fatalf("expected transactions_synced=true")
	}
	if !savedBlock.ReceiptsSynced {
		t.Fatalf("expected receipts_synced=true")
	}
	if savedBlock.SyncStatus != models.BlockSyncStatusCompleted {
		t.Fatalf("expected sync_status=%s, got %s",
			models.BlockSyncStatusCompleted,
			savedBlock.SyncStatus,
		)
	}
	if savedBlock.LastSyncError != nil {
		t.Fatalf("expected last_sync_error=nil, got %v", *savedBlock.LastSyncError)
	}
}

func TestBlockRepository_MarkBlockReceiptsSynced_NotFound(t *testing.T) {
	r, _ := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	err := r.MarkBlockReceiptsSynced(ctx, 999)
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}
}

func TestBlockRepository_MarkBlockReceiptsSynced_DBError(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err = r.MarkBlockReceiptsSynced(ctx, 100)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBlockRepository_MarkBlockReceiptsSyncFailed_Success(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	block := newBlockWithTxTestBlock(100, "0xblockhash100")
	block.ReceiptsSynced = false
	block.SyncStatus = models.BlockSyncStatusTransactionsSynced
	block.LastSyncError = stringPtr("old receipt error")

	if err := db.Create(block).Error; err != nil {
		t.Fatalf("seed block: %v", err)
	}

	reason := "get receipt: context deadline exceeded"

	if err := r.MarkBlockReceiptsSyncFailed(ctx, 100, reason); err != nil {
		t.Fatalf("mark block receipts sync failed: %v", err)
	}

	var savedBlock models.Block
	if err := db.Where("number = ?", 100).First(&savedBlock).Error; err != nil {
		t.Fatalf("find saved block: %v", err)
	}

	if !savedBlock.TransactionsSynced {
		t.Fatalf("expected transactions_synced=true")
	}
	if savedBlock.ReceiptsSynced {
		t.Fatalf("expected receipts_synced=false")
	}
	if savedBlock.SyncStatus != models.BlockSyncStatusReceiptsFailed {
		t.Fatalf("expected sync_status=%s, got %s",
			models.BlockSyncStatusReceiptsFailed,
			savedBlock.SyncStatus,
		)
	}
	if savedBlock.LastSyncError == nil {
		t.Fatalf("expected last_sync_error to be set")
	}
	if *savedBlock.LastSyncError != reason {
		t.Fatalf("expected last_sync_error=%q, got %q", reason, *savedBlock.LastSyncError)
	}
}

func TestBlockRepository_MarkBlockReceiptsSyncFailed_NotFound(t *testing.T) {
	r, _ := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	err := r.MarkBlockReceiptsSyncFailed(ctx, 999, "receipt sync failed")
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}
}

func TestBlockRepository_MarkBlockReceiptsSyncFailed_DBError(t *testing.T) {
	r, db := setupBlockWithTransactionsRepo(t)
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err = r.MarkBlockReceiptsSyncFailed(ctx, 100, "receipt sync failed")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBlockRepository_ListWalletCompletedBlockRows_ReturnsOnlyCompletedBlocks(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)
	ctx := context.Background()

	if err := db.Create(newWalletCompletedRowsTestBlock(99, true, true, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
		t.Fatalf("seed block 99: %v", err)
	}

	if err := db.Create(newWalletCompletedRowsTestBlock(100, true, true, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
		t.Fatalf("seed block 100: %v", err)
	}

	if err := db.Create(newWalletCompletedRowsTestBlock(101, false, true, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
		t.Fatalf("seed block 101: %v", err)
	}

	if err := db.Create(newWalletCompletedRowsTestBlock(102, true, false, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
		t.Fatalf("seed block 102: %v", err)
	}

	if err := db.Create(newWalletCompletedRowsTestBlock(103, true, true, models.BlockSyncStatusPending, nil)).Error; err != nil {
		t.Fatalf("seed block 103: %v", err)
	}

	syncErr := "receipt sync failed"
	if err := db.Create(newWalletCompletedRowsTestBlock(104, true, true, models.BlockSyncStatusCompleted, &syncErr)).Error; err != nil {
		t.Fatalf("seed block 104: %v", err)
	}

	if err := db.Create(newWalletCompletedRowsTestBlock(105, true, true, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
		t.Fatalf("seed block 105: %v", err)
	}

	blocks, err := r.ListWalletCompletedBlockRows(ctx, 100, 10)
	if err != nil {
		t.Fatalf("list wallet completed block rows: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 completed blocks, got %d", len(blocks))
	}

	if blocks[0].Number != 100 {
		t.Fatalf("expected first block 100, got %d", blocks[0].Number)
	}

	if blocks[1].Number != 105 {
		t.Fatalf("expected second block 105, got %d", blocks[1].Number)
	}
}

func TestBlockRepository_ListWalletCompletedBlockRows_AppliesLimitToCompletedBlocks(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)
	ctx := context.Background()

	for _, number := range []uint64{100, 101, 102} {
		if err := db.Create(newWalletCompletedRowsTestBlock(number, true, true, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
			t.Fatalf("seed block %d: %v", number, err)
		}
	}

	blocks, err := r.ListWalletCompletedBlockRows(ctx, 100, 2)
	if err != nil {
		t.Fatalf("list wallet completed block rows: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].Number != 100 {
		t.Fatalf("expected first block 100, got %d", blocks[0].Number)
	}

	if blocks[1].Number != 101 {
		t.Fatalf("expected second block 101, got %d", blocks[1].Number)
	}
}

func TestBlockRepository_ListWalletCompletedBlockRows_ReturnsEmptyWhenNoMatch(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)
	ctx := context.Background()

	if err := db.Create(newWalletCompletedRowsTestBlock(100, true, false, models.BlockSyncStatusCompleted, nil)).Error; err != nil {
		t.Fatalf("seed block: %v", err)
	}

	blocks, err := r.ListWalletCompletedBlockRows(ctx, 100, 10)
	if err != nil {
		t.Fatalf("list wallet completed block rows: %v", err)
	}

	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestBlockRepository_ListWalletCompletedBlockRows_InvalidArgsReturnError(t *testing.T) {
	db := SetupTestDB(t)
	r := NewBlockRepository(db)
	ctx := context.Background()

	_, err := r.ListWalletCompletedBlockRows(ctx, -1, 10)
	if err == nil {
		t.Fatal("expected error for negative fromBlock, got nil")
	}

	if !strings.Contains(err.Error(), "from_block must be non-negative") {
		t.Fatalf("expected from_block error, got %q", err.Error())
	}

	_, err = r.ListWalletCompletedBlockRows(ctx, 100, 0)
	if err == nil {
		t.Fatal("expected error for non-positive limit, got nil")
	}

	if !strings.Contains(err.Error(), "limit must be positive") {
		t.Fatalf("expected limit error, got %q", err.Error())
	}
}

func newWalletCompletedRowsTestBlock(
	number uint64,
	transactionsSynced bool,
	receiptsSynced bool,
	syncStatus string,
	lastSyncError *string,
) *models.Block {
	return &models.Block{
		Number:             number,
		Hash:               fmt.Sprintf("0xwalletblockhash%d", number),
		ParentHash:         fmt.Sprintf("0xwalletparenthash%d", number),
		Timestamp:          1700000000 + number,
		Miner:              "0x1111111111111111111111111111111111111111",
		TxCount:            1,
		GasUsed:            21000,
		GasLimit:           30000000,
		TransactionsSynced: transactionsSynced,
		ReceiptsSynced:     receiptsSynced,
		SyncStatus:         syncStatus,
		LastSyncError:      lastSyncError,
	}
}

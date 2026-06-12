package service

import (
	"block-explorer-backend/internal/db/models"
	servicemodel "block-explorer-backend/internal/service/model"
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	blockServiceTestChainID int64  = 11155111
	blockServiceStartBlock  uint64 = 100
)

type fakeChainBlockReader struct {
	blocks map[uint64]*ethtypes.Block
	err    error

	called    bool
	gotNumber uint64
}

func (f *fakeChainBlockReader) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	f.called = true
	f.gotNumber = number

	if f.err != nil {
		return nil, f.err
	}

	if f.blocks != nil {
		if block, ok := f.blocks[number]; ok {
			return block, nil
		}
	}

	return nil, nil
}

type fakeGetBlockCall struct {
	chainID int64
	number  uint64
}

type fakeBlockSyncStore struct {
	blocks map[uint64]*models.Block
	getErr error

	getCalls []fakeGetBlockCall

	insertCalled  bool
	insertedBlock *models.Block
	insertedTxs   []*models.Transaction
	insertErr     error

	markReceiptsSyncedCalled      bool
	markReceiptsSyncedChainID     int64
	markReceiptsSyncedBlockNumber uint64
	markReceiptsSyncedErr         error

	markReceiptsSyncFailedCalled      bool
	markReceiptsSyncFailedChainID     int64
	markReceiptsSyncFailedBlockNumber uint64
	markReceiptsSyncFailedReason      string
	markReceiptsSyncFailedErr         error
}

func (f *fakeBlockSyncStore) GetBlockByNumber(
	ctx context.Context,
	chainID int64,
	number uint64,
) (*models.Block, bool, error) {
	f.getCalls = append(f.getCalls, fakeGetBlockCall{
		chainID: chainID,
		number:  number,
	})

	if f.getErr != nil {
		return nil, false, f.getErr
	}

	if f.blocks != nil {
		if block, ok := f.blocks[number]; ok {
			return block, true, nil
		}
	}

	return nil, false, nil
}

func (f *fakeBlockSyncStore) InsertBlockWithTransactions(
	ctx context.Context,
	block *models.Block,
	txs []*models.Transaction,
) error {
	f.insertCalled = true
	f.insertedBlock = block
	f.insertedTxs = txs

	if f.insertErr != nil {
		return f.insertErr
	}

	return nil
}

func (f *fakeBlockSyncStore) MarkBlockReceiptsSynced(
	ctx context.Context,
	chainID int64,
	blockNumber uint64,
) error {
	f.markReceiptsSyncedCalled = true
	f.markReceiptsSyncedChainID = chainID
	f.markReceiptsSyncedBlockNumber = blockNumber

	if f.markReceiptsSyncedErr != nil {
		return f.markReceiptsSyncedErr
	}

	return nil
}

func (f *fakeBlockSyncStore) MarkBlockReceiptsSyncFailed(
	ctx context.Context,
	chainID int64,
	blockNumber uint64,
	reason string,
) error {
	f.markReceiptsSyncFailedCalled = true
	f.markReceiptsSyncFailedChainID = chainID
	f.markReceiptsSyncFailedBlockNumber = blockNumber
	f.markReceiptsSyncFailedReason = reason

	if f.markReceiptsSyncFailedErr != nil {
		return f.markReceiptsSyncFailedErr
	}

	return nil
}

type fakeBlockReceiptSyncer struct {
	called         bool
	gotBlockNumber uint64
	err            error
}

func (f *fakeBlockReceiptSyncer) SyncBlockTransactionReceipts(
	ctx context.Context,
	blockNumber uint64,
) error {
	f.called = true
	f.gotBlockNumber = blockNumber

	if f.err != nil {
		return f.err
	}

	return nil
}

type fakeTransactionConflictReader struct {
	existingTxs map[string]*models.Transaction
	err         error

	called     bool
	gotChainID int64
	gotHashes  []string
}

func (f *fakeTransactionConflictReader) GetTransactionsByHashes(
	ctx context.Context,
	chainID int64,
	hashes []string,
) (map[string]*models.Transaction, error) {
	f.called = true
	f.gotChainID = chainID
	f.gotHashes = append([]string(nil), hashes...)

	if f.err != nil {
		return nil, f.err
	}

	if f.existingTxs != nil {
		return f.existingTxs, nil
	}

	return map[string]*models.Transaction{}, nil
}

func setupBlockServiceChainIDTest() (
	*BlockService,
	*fakeChainBlockReader,
	*fakeBlockSyncStore,
	*fakeBlockReceiptSyncer,
	*fakeTransactionConflictReader,
) {
	chainReader := &fakeChainBlockReader{
		blocks: map[uint64]*ethtypes.Block{},
	}

	blockStore := &fakeBlockSyncStore{
		blocks: map[uint64]*models.Block{},
	}

	receiptSyncer := &fakeBlockReceiptSyncer{}

	txConflictReader := &fakeTransactionConflictReader{
		existingTxs: map[string]*models.Transaction{},
	}

	s := NewBlockService(
		blockServiceTestChainID,
		chainReader,
		blockStore,
		receiptSyncer,
		txConflictReader,
		blockServiceStartBlock,
	)

	return s, chainReader, blockStore, receiptSyncer, txConflictReader
}

func newBlockServiceTestRPCBlock(number uint64, parentHash common.Hash) *ethtypes.Block {
	header := &ethtypes.Header{
		Number:     big.NewInt(int64(number)),
		ParentHash: parentHash,
		Coinbase:   common.HexToAddress("0x000000000000000000000000000000000000beef"),
		Time:       1710000000,
		GasLimit:   30000000,
		GasUsed:    21000,
	}

	return ethtypes.NewBlockWithHeader(header)
}

func newBlockServiceTestDBBlock(
	chainID int64,
	number uint64,
	hash string,
	parentHash string,
) *models.Block {
	return &models.Block{
		ChainID:    chainID,
		Number:     number,
		Hash:       hash,
		ParentHash: parentHash,
		Timestamp:  1710000000,
		Miner:      "0x000000000000000000000000000000000000beef",
		GasLimit:   30000000,
		GasUsed:    21000,
		TxCount:    0,
	}
}

func TestBlockService_GetBlockByNumber_FromDBPassesChainID(t *testing.T) {
	s, chainReader, blockStore, _, _ := setupBlockServiceChainIDTest()

	number := uint64(100)
	dbBlock := newBlockServiceTestDBBlock(
		blockServiceTestChainID,
		number,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)

	blockStore.blocks[number] = dbBlock

	got, err := s.GetBlockByNumber(context.Background(), number)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(blockStore.getCalls) != 1 {
		t.Fatalf("expected 1 get block call, got %d", len(blockStore.getCalls))
	}

	if blockStore.getCalls[0].chainID != blockServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", blockServiceTestChainID, blockStore.getCalls[0].chainID)
	}

	if blockStore.getCalls[0].number != number {
		t.Fatalf("expected block number=%d, got %d", number, blockStore.getCalls[0].number)
	}

	if chainReader.called {
		t.Fatalf("expected rpc reader not called when block exists in db")
	}

	if got.Source != servicemodel.DataSourceDB {
		t.Fatalf("expected source=%s, got %s", servicemodel.DataSourceDB, got.Source)
	}

	if got.Block.ChainID != blockServiceTestChainID {
		t.Fatalf("expected result chain_id=%d, got %d", blockServiceTestChainID, got.Block.ChainID)
	}

	if got.Block.Number != number {
		t.Fatalf("expected block number=%d, got %d", number, got.Block.Number)
	}

	if got.Block.Hash != dbBlock.Hash {
		t.Fatalf("expected hash=%s, got %s", dbBlock.Hash, got.Block.Hash)
	}
}

func TestBlockService_GetBlockByNumber_FromRPCUsesServiceChainID(t *testing.T) {
	s, chainReader, blockStore, _, _ := setupBlockServiceChainIDTest()

	number := uint64(100)
	parentHash := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	rpcBlock := newBlockServiceTestRPCBlock(number, parentHash)

	chainReader.blocks[number] = rpcBlock

	got, err := s.GetBlockByNumber(context.Background(), number)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(blockStore.getCalls) != 1 {
		t.Fatalf("expected 1 get block call, got %d", len(blockStore.getCalls))
	}

	if blockStore.getCalls[0].chainID != blockServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", blockServiceTestChainID, blockStore.getCalls[0].chainID)
	}

	if !chainReader.called {
		t.Fatalf("expected rpc reader called")
	}

	if chainReader.gotNumber != number {
		t.Fatalf("expected rpc block number=%d, got %d", number, chainReader.gotNumber)
	}

	if got.Source != servicemodel.DataSourceRPC {
		t.Fatalf("expected source=%s, got %s", servicemodel.DataSourceRPC, got.Source)
	}

	if got.Block.ChainID != blockServiceTestChainID {
		t.Fatalf("expected result chain_id=%d, got %d", blockServiceTestChainID, got.Block.ChainID)
	}

	if got.Block.Number != number {
		t.Fatalf("expected block number=%d, got %d", number, got.Block.Number)
	}

	if got.Block.Hash != rpcBlock.Hash().Hex() {
		t.Fatalf("expected hash=%s, got %s", rpcBlock.Hash().Hex(), got.Block.Hash)
	}
}

func TestBlockService_SyncBlockToDB_EmptyBlockInsertsBlockWithChainID(t *testing.T) {
	s, chainReader, blockStore, receiptSyncer, _ := setupBlockServiceChainIDTest()

	number := blockServiceStartBlock
	parentHash := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	rpcBlock := newBlockServiceTestRPCBlock(number, parentHash)

	chainReader.blocks[number] = rpcBlock

	err := s.SyncBlockToDB(context.Background(), number)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !blockStore.insertCalled {
		t.Fatalf("expected block inserted")
	}

	if blockStore.insertedBlock == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if blockStore.insertedBlock.ChainID != blockServiceTestChainID {
		t.Fatalf("expected inserted block chain_id=%d, got %d", blockServiceTestChainID, blockStore.insertedBlock.ChainID)
	}

	if blockStore.insertedBlock.Number != number {
		t.Fatalf("expected inserted block number=%d, got %d", number, blockStore.insertedBlock.Number)
	}

	if blockStore.insertedBlock.Hash != rpcBlock.Hash().Hex() {
		t.Fatalf("expected inserted block hash=%s, got %s", rpcBlock.Hash().Hex(), blockStore.insertedBlock.Hash)
	}

	if !blockStore.insertedBlock.TransactionsSynced {
		t.Fatalf("expected TransactionsSynced=true")
	}

	if !blockStore.insertedBlock.ReceiptsSynced {
		t.Fatalf("expected ReceiptsSynced=true for empty block")
	}

	if blockStore.insertedBlock.SyncStatus != models.BlockSyncStatusCompleted {
		t.Fatalf("expected sync status=%s, got %s", models.BlockSyncStatusCompleted, blockStore.insertedBlock.SyncStatus)
	}

	if len(blockStore.insertedTxs) != 0 {
		t.Fatalf("expected no inserted txs, got %d", len(blockStore.insertedTxs))
	}

	if receiptSyncer.called {
		t.Fatalf("expected receipt syncer not called for empty block")
	}

	if blockStore.markReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced not called for empty block")
	}

	for _, call := range blockStore.getCalls {
		if call.chainID != blockServiceTestChainID {
			t.Fatalf("expected all get block calls use chain_id=%d, got %d for block %d", blockServiceTestChainID, call.chainID, call.number)
		}
	}
}

func TestBlockService_ValidateBlockForSync_ParentContinuityUsesChainID(t *testing.T) {
	s, _, blockStore, _, _ := setupBlockServiceChainIDTest()

	parentNumber := uint64(100)
	childNumber := uint64(101)

	parentHash := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	childHash := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	blockStore.blocks[parentNumber] = newBlockServiceTestDBBlock(
		blockServiceTestChainID,
		parentNumber,
		parentHash,
		"0xparentparent",
	)

	childBlock := &models.Block{
		ChainID:    blockServiceTestChainID,
		Number:     childNumber,
		Hash:       childHash,
		ParentHash: parentHash,
	}

	err := s.validateBlockForSync(context.Background(), childBlock)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(blockStore.getCalls) != 2 {
		t.Fatalf("expected 2 get block calls, got %d", len(blockStore.getCalls))
	}

	if blockStore.getCalls[0].chainID != blockServiceTestChainID {
		t.Fatalf("expected first get chain_id=%d, got %d", blockServiceTestChainID, blockStore.getCalls[0].chainID)
	}

	if blockStore.getCalls[0].number != childNumber {
		t.Fatalf("expected first get block number=%d, got %d", childNumber, blockStore.getCalls[0].number)
	}

	if blockStore.getCalls[1].chainID != blockServiceTestChainID {
		t.Fatalf("expected second get chain_id=%d, got %d", blockServiceTestChainID, blockStore.getCalls[1].chainID)
	}

	if blockStore.getCalls[1].number != parentNumber {
		t.Fatalf("expected second get block number=%d, got %d", parentNumber, blockStore.getCalls[1].number)
	}
}

func TestBlockService_ValidateTransactionsForSync_PassesChainIDAndHashes(t *testing.T) {
	s, _, _, _, txConflictReader := setupBlockServiceChainIDTest()

	txModels := []*models.Transaction{
		{
			ChainID:     blockServiceTestChainID,
			Hash:        "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			BlockNumber: 100,
			BlockHash:   "0xblockhash1",
			TxIndex:     0,
		},
		{
			ChainID:     blockServiceTestChainID,
			Hash:        "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			BlockNumber: 100,
			BlockHash:   "0xblockhash1",
			TxIndex:     1,
		},
	}

	err := s.validateTransactionsForSync(context.Background(), txModels)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !txConflictReader.called {
		t.Fatalf("expected transaction conflict reader called")
	}

	if txConflictReader.gotChainID != blockServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", blockServiceTestChainID, txConflictReader.gotChainID)
	}

	if len(txConflictReader.gotHashes) != len(txModels) {
		t.Fatalf("expected %d hashes, got %d", len(txModels), len(txConflictReader.gotHashes))
	}

	for i, txModel := range txModels {
		if txConflictReader.gotHashes[i] != txModel.Hash {
			t.Fatalf("expected hash[%d]=%s, got %s", i, txModel.Hash, txConflictReader.gotHashes[i])
		}
	}
}

func TestBlockService_ValidateTransactionsForSync_DetectsConflictWithinSameChain(t *testing.T) {
	s, _, _, _, txConflictReader := setupBlockServiceChainIDTest()

	hash := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	txModels := []*models.Transaction{
		{
			ChainID:     blockServiceTestChainID,
			Hash:        hash,
			BlockNumber: 101,
			BlockHash:   "0xnewblockhash",
			TxIndex:     0,
		},
	}

	txConflictReader.existingTxs = map[string]*models.Transaction{
		hash: {
			ChainID:     blockServiceTestChainID,
			Hash:        hash,
			BlockNumber: 100,
			BlockHash:   "0xoldblockhash",
			TxIndex:     0,
		},
	}

	err := s.validateTransactionsForSync(context.Background(), txModels)
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}

	if !strings.Contains(err.Error(), "transaction conflict") {
		t.Fatalf("expected transaction conflict error, got %v", err)
	}

	if txConflictReader.gotChainID != blockServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", blockServiceTestChainID, txConflictReader.gotChainID)
	}
}

package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type fakeChainBlockReader struct {
	block *ethtypes.Block
	err   error

	blocks map[uint64]*ethtypes.Block
	errs   map[uint64]error

	onGetBlock func(number uint64)

	chainID          *big.Int
	chainIDErr       error
	getChainIDCalled bool
}

func (f *fakeChainBlockReader) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	if f.onGetBlock != nil {
		f.onGetBlock(number)
	}

	if f.errs != nil {
		if err, ok := f.errs[number]; ok {
			return nil, err
		}
	}
	if f.blocks != nil {
		if block, ok := f.blocks[number]; ok {
			return block, nil
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.block, nil
}

func (f *fakeChainBlockReader) GetChainID(ctx context.Context) (*big.Int, error) {
	f.getChainIDCalled = true
	if f.chainIDErr != nil {
		return nil, f.chainIDErr
	}
	if f.chainID != nil {
		return f.chainID, nil
	}
	return big.NewInt(1), nil
}

type fakeBlockSyncStore struct {
	block *models.Block
	found bool
	err   error

	blocks map[uint64]*models.Block

	insertBlockWithTransactionsCalled bool
	insertBlockArg                    *models.Block
	insertTxsArg                      []*models.Transaction
	insertBlockWithTransactionsErr    error
	insertedBlocks                    map[uint64]*models.Block

	markBlockReceiptsSyncedCalled      bool
	markBlockReceiptsSyncedBlockNumber uint64
	markBlockReceiptsSyncedErr         error

	markBlockReceiptsSyncFailedCalled      bool
	markBlockReceiptsSyncFailedBlockNumber uint64
	markBlockReceiptsSyncFailedReason      string
	markBlockReceiptsSyncFailedErr         error
}

func (f *fakeBlockSyncStore) InsertBlockWithTransactions(
	ctx context.Context,
	block *models.Block,
	txs []*models.Transaction,
) error {
	f.insertBlockWithTransactionsCalled = true
	f.insertBlockArg = block
	f.insertTxsArg = txs

	if f.insertBlockWithTransactionsErr != nil {
		return f.insertBlockWithTransactionsErr
	}

	if f.insertedBlocks == nil {
		f.insertedBlocks = make(map[uint64]*models.Block)
	}

	f.insertedBlocks[block.Number] = block
	return nil
}

func (f *fakeBlockSyncStore) GetBlockByNumber(ctx context.Context, number uint64) (*models.Block, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}

	if f.blocks != nil {
		if block, found := f.blocks[number]; found {
			return block, true, nil
		}
	}

	if f.insertedBlocks != nil {
		if block, found := f.insertedBlocks[number]; found {
			return block, true, nil
		}
	}

	if f.found && f.block != nil && f.block.Number == number {
		return f.block, true, nil
	}
	return nil, false, nil
}

func (f *fakeBlockSyncStore) MarkBlockReceiptsSynced(ctx context.Context, blockNumber uint64) error {
	f.markBlockReceiptsSyncedCalled = true
	f.markBlockReceiptsSyncedBlockNumber = blockNumber

	if f.markBlockReceiptsSyncedErr != nil {
		return f.markBlockReceiptsSyncedErr
	}

	return nil
}

func (f *fakeBlockSyncStore) MarkBlockReceiptsSyncFailed(ctx context.Context, blockNumber uint64, reason string) error {
	f.markBlockReceiptsSyncFailedCalled = true
	f.markBlockReceiptsSyncFailedBlockNumber = blockNumber
	f.markBlockReceiptsSyncFailedReason = reason

	if f.markBlockReceiptsSyncFailedErr != nil {
		return f.markBlockReceiptsSyncFailedErr
	}

	return nil
}

type fakeBlockReceiptSyncer struct {
	called         bool
	calls          int
	gotBlockNumber uint64
	err            error
}

func (f *fakeBlockReceiptSyncer) SyncBlockTransactionReceipts(ctx context.Context, blockNumber uint64) error {
	f.called = true
	f.calls++
	f.gotBlockNumber = blockNumber

	if f.err != nil {
		return f.err
	}

	return nil
}

type fakeTransactionConflictReader struct {
	existingTxs map[string]*models.Transaction
	err         error

	called bool
	hashes []string
}

func (f *fakeTransactionConflictReader) GetTransactionsByHashes(
	ctx context.Context,
	hashes []string,
) (map[string]*models.Transaction, error) {
	f.called = true
	f.hashes = hashes

	if f.err != nil {
		return nil, f.err
	}
	if f.existingTxs != nil {
		return f.existingTxs, nil
	}
	return map[string]*models.Transaction{}, nil
}

func assertInsertedBlockSyncState(
	t *testing.T,
	block *models.Block,
	wantTransactionsSynced bool,
	wantReceiptsSynced bool,
	wantSyncStatus string,
) {
	t.Helper()

	if block == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if block.TransactionsSynced != wantTransactionsSynced {
		t.Fatalf("expected transactions_synced=%v, got %v",
			wantTransactionsSynced,
			block.TransactionsSynced,
		)
	}

	if block.ReceiptsSynced != wantReceiptsSynced {
		t.Fatalf("expected receipts_synced=%v, got %v",
			wantReceiptsSynced,
			block.ReceiptsSynced,
		)
	}

	if block.SyncStatus != wantSyncStatus {
		t.Fatalf("expected sync_status=%s, got %s",
			wantSyncStatus,
			block.SyncStatus,
		)
	}
}

type blockServiceTestEnv struct {
	svc           *BlockService
	blockRPC      *fakeChainBlockReader
	blockRepo     *fakeBlockSyncStore
	receiptSyncer *fakeBlockReceiptSyncer
	txRepo        *fakeTransactionConflictReader
}

func setupBlockServiceTestEnv(t *testing.T, startBlock uint64) *blockServiceTestEnv {
	t.Helper()

	rpc := &fakeChainBlockReader{}
	blockRepo := &fakeBlockSyncStore{}
	receiptSyncer := &fakeBlockReceiptSyncer{}
	txRepo := &fakeTransactionConflictReader{}

	svc := NewBlockService(
		rpc,
		blockRepo,
		receiptSyncer,
		txRepo,
		startBlock,
	)

	return &blockServiceTestEnv{
		svc:           svc,
		blockRepo:     blockRepo,
		blockRPC:      rpc,
		txRepo:        txRepo,
		receiptSyncer: receiptSyncer,
	}
}

func setupTestService(t *testing.T) (*BlockService, *fakeBlockSyncStore, *fakeChainBlockReader) {
	t.Helper()

	env := setupBlockServiceTestEnv(t, 20)

	return env.svc, env.blockRepo, env.blockRPC
}

func setupTestServiceWithStartBlock(
	t *testing.T,
	startBlock uint64,
) (*BlockService, *fakeBlockSyncStore, *fakeChainBlockReader) {
	t.Helper()

	env := setupBlockServiceTestEnv(t, startBlock)

	return env.svc, env.blockRepo, env.blockRPC
}

func setupTestServiceWithReceiptSyncer(
	t *testing.T,
) (*BlockService, *fakeBlockSyncStore, *fakeChainBlockReader, *fakeBlockReceiptSyncer) {
	t.Helper()

	env := setupBlockServiceTestEnv(t, 20)

	return env.svc, env.blockRepo, env.blockRPC, env.receiptSyncer
}

func setupTestServiceWithReceiptSyncerAndStartBlock(
	t *testing.T,
	startBlock uint64,
) (*BlockService, *fakeChainBlockReader, *fakeBlockSyncStore, *fakeBlockReceiptSyncer) {
	t.Helper()

	env := setupBlockServiceTestEnv(t, startBlock)

	return env.svc, env.blockRPC, env.blockRepo, env.receiptSyncer
}

func TestBlockService_GetBlockByNumber_DBHit(t *testing.T) {
	svc, repo, _ := setupTestService(t)

	repo.block = &models.Block{
		Number: 20,
		Hash:   "0x123",
	}
	repo.found = true
	repo.err = nil

	result, err := svc.GetBlockByNumber(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Block.Number != 20 {
		t.Fatalf("expected block number 20, got %d", result.Block.Number)
	}
	if result.Block.Hash != "0x123" {
		t.Fatalf("expected block hash \"0x123\", got %s", result.Block.Hash)
	}
}

func TestBlockService_GetBlockByNumber_DBError(t *testing.T) {
	svc, repo, _ := setupTestService(t)

	repo.block = nil
	repo.found = false
	repo.err = types.ErrDBTimeout

	_, err := svc.GetBlockByNumber(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrDBTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestBlockService_GetBlockByNumber_RPCSuccess(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	repo.block = nil
	repo.found = false
	repo.err = nil

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	result, err := svc.GetBlockByNumber(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Block.Number != 20 {
		t.Fatalf("expected block number 20, got %d", result.Block.Number)
	}
}

func TestBlockService_GetBlockByNumber_RPCNotFound(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	repo.block = nil
	repo.found = false
	repo.err = nil

	rpc.err = types.ErrBlockNotFound
	_, err := svc.GetBlockByNumber(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected RPCNotFound, got %v", err)
	}
}

func TestBlockService_SyncBlockToDB_EmptyBlock_Success(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertBlockArg.Number)
	}
	if len(repo.insertTxsArg) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(repo.insertTxsArg))
	}
	if rpc.getChainIDCalled {
		t.Fatalf("expected GetChainID not to be called for empty block")
	}
}

func newSignedTestTransaction(t *testing.T, chainID *big.Int, nonce uint64, to common.Address) (*ethtypes.Transaction, common.Address) {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    big.NewInt(1000000000000000000),
		Gas:      21000,
		GasPrice: big.NewInt(1000000000),
		Data:     []byte{0x01, 0x02},
	})

	signer := ethtypes.LatestSignerForChainID(chainID)

	signedTx, err := ethtypes.SignTx(tx, signer, privateKey)
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}

	return signedTx, from
}

func newTestBlockWithTransactions(number uint64, txs []*ethtypes.Transaction) *ethtypes.Block {
	header := &ethtypes.Header{
		Number: big.NewInt(int64(number)),
	}

	return ethtypes.NewBlockWithHeader(header).WithBody(ethtypes.Body{
		Transactions: txs,
	})
}

func TestBlockService_SyncBlockToDB_InsertsBlockWithTransactions(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	signedTx, expectedFrom := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertBlockArg.Number)
	}

	if len(repo.insertTxsArg) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.insertTxsArg))
	}

	gotTx := repo.insertTxsArg[0]
	if gotTx.Hash != signedTx.Hash().Hex() {
		t.Fatalf("expected tx hash=%s, got %s", signedTx.Hash().Hex(), gotTx.Hash)
	}

	if gotTx.BlockNumber != 20 {
		t.Fatalf("expected tx block number=20, got %d", gotTx.BlockNumber)
	}

	if gotTx.BlockHash != rpc.block.Hash().Hex() {
		t.Fatalf("expected tx block hash=%s, got %s", rpc.block.Hash().Hex(), gotTx.BlockHash)
	}

	if gotTx.TxIndex != 0 {
		t.Fatalf("expected tx index=0, got %d", gotTx.TxIndex)
	}

	if !strings.EqualFold(gotTx.FromAddress, expectedFrom.Hex()) {
		t.Fatalf("expected from address=%s, got %s", expectedFrom.Hex(), gotTx.FromAddress)
	}

	if !strings.EqualFold(gotTx.ToAddress, to.Hex()) {
		t.Fatalf("expected to address=%s, got %s", to.Hex(), gotTx.ToAddress)
	}

	if gotTx.ValueWei != "1000000000000000000" {
		t.Fatalf("expected value wei=1000000000000000000, got %s", gotTx.ValueWei)
	}

	if gotTx.GasLimit != 21000 {
		t.Fatalf("expected gas limit=21000, got %d", gotTx.GasLimit)
	}

	if gotTx.GasPriceWei != "1000000000" {
		t.Fatalf("expected gas price wei=1000000000, got %s", gotTx.GasPriceWei)
	}

	if gotTx.InputData != "0x0102" {
		t.Fatalf("expected input data=0x0102, got %s", gotTx.InputData)
	}

}

func TestBlockService_SyncBlockToDB_BackfillsTransactionsWhenBlockAlreadyExistsWithSameHash(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	signedTx, expectedFrom := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	repo.block = &models.Block{
		Number: 20,
	}
	repo.found = true
	repo.block.Hash = rpc.block.Hash().Hex()

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertBlockArg.Number)
	}

	if len(repo.insertTxsArg) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.insertTxsArg))
	}

	gotTx := repo.insertTxsArg[0]

	if gotTx.Hash != signedTx.Hash().Hex() {
		t.Fatalf("expected tx hash=%s, got %s", signedTx.Hash().Hex(), gotTx.Hash)
	}

	if gotTx.BlockNumber != 20 {
		t.Fatalf("expected tx block number=20, got %d", gotTx.BlockNumber)
	}

	if !strings.EqualFold(gotTx.FromAddress, expectedFrom.Hex()) {
		t.Fatalf("expected from address=%s, got %s", expectedFrom.Hex(), gotTx.FromAddress)
	}

	if !strings.EqualFold(gotTx.ToAddress, to.Hex()) {
		t.Fatalf("expected to address=%s, got %s", to.Hex(), gotTx.ToAddress)
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrReorgDetectedWhenSameHeightHasDifferentHash(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	rpcHash := rpc.block.Hash().Hex()
	dbHash := common.HexToHash("0xbbbb").Hex()

	if strings.EqualFold(rpcHash, dbHash) {
		t.Fatalf("test setup invalid: rpc hash equals db hash")
	}

	repo.block = &models.Block{
		Number: 20,
		Hash:   dbHash,
	}
	repo.found = true

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, types.ErrReorgDetected) {
		t.Fatalf("expected ErrReorgDetected, got %v", err)
	}

	if rpc.getChainIDCalled {
		t.Fatalf("expected GetChainID not to be called when reorg is detected")
	}

	if repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}
}

func TestBlockService_SyncBlockToDB_RPCNotFound(t *testing.T) {
	svc, repo, rpc := setupTestService(t)
	rpc.block = nil
	rpc.err = types.ErrBlockNotFound
	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}
	if repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}
}

func TestBlockService_SyncBlockToDB_RPCError(t *testing.T) {
	svc, repo, rpc := setupTestService(t)
	rpc.block = nil
	rpc.err = types.ErrRPCTimeout
	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrRPCTimeout) {
		t.Fatalf("expected ErrRPCTimeout, got %v", err)
	}
	if repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}
}

func TestBlockService_SyncBlockToDB_InsertBlockWithTransactionsError(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	repo.insertBlockWithTransactionsErr = errors.New("some error")

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected insertBlockArg not nil")
	}

	if repo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertBlockArg.Number)
	}
}

func TestBlockService_SyncBlockToDB_InsertsBlockWhenParentHashMatches(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	parentHash := common.HexToHash("0xaaa")

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number:     big.NewInt(100),
		ParentHash: parentHash,
	})

	repo.blocks = map[uint64]*models.Block{
		99: {
			Number: 99,
			Hash:   parentHash.Hex(),
		},
	}

	err := svc.SyncBlockToDB(context.Background(), 100)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected insertBlockWithTransactions to be called")
	}
	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 100 {
		t.Fatalf("expected inserted block number=100, got %d", repo.insertBlockArg.Number)
	}
}

func TestBlockService_SyncBlockToDB_AllowsSyncFromLocalStartBlockWhenParentIsMissing(t *testing.T) {
	svc, repo, rpc := setupTestServiceWithStartBlock(t, 100)
	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number:     big.NewInt(100),
		ParentHash: common.HexToHash("0xaaa"),
	})
	repo.blocks = map[uint64]*models.Block{}
	err := svc.SyncBlockToDB(context.Background(), 100)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 100 {
		t.Fatalf("expected inserted block number=100, got %d", repo.insertBlockArg.Number)
	}

	if len(repo.insertTxsArg) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(repo.insertTxsArg))
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrChainDiscontinuityWhenParentMissingAfterStartBlock(t *testing.T) {
	svc, repo, rpc := setupTestServiceWithStartBlock(t, 100)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number:     big.NewInt(101),
		ParentHash: common.HexToHash("0xaaa"),
	})
	repo.blocks = map[uint64]*models.Block{}

	err := svc.SyncBlockToDB(context.Background(), 101)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, types.ErrChainDiscontinuity) {
		t.Fatalf("expected ErrChainDiscontinuity, got %v", err)
	}

	if repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrChainDiscontinuityWhenParentHashMismatch(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpcParentHash := common.HexToHash("0xaaa")
	dbParentHash := common.HexToHash("0xbbb")
	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number:     big.NewInt(100),
		ParentHash: rpcParentHash,
	})
	repo.blocks = map[uint64]*models.Block{
		99: {
			Number: 99,
			Hash:   dbParentHash.Hex(),
		},
	}

	err := svc.SyncBlockToDB(context.Background(), 100)
	if !errors.Is(err, types.ErrChainDiscontinuity) {
		t.Fatalf("expected ErrChainDiscontinuity, got %v", err)
	}
	if repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}
}

func newLinkedTestBlocks(numbers ...uint64) map[uint64]*ethtypes.Block {
	blocks := make(map[uint64]*ethtypes.Block, len(numbers))

	var parentHash common.Hash
	for i, number := range numbers {
		header := &ethtypes.Header{
			Number: big.NewInt(int64(number)),
		}

		if i > 0 {
			header.ParentHash = parentHash
		}

		block := ethtypes.NewBlockWithHeader(header)
		blocks[number] = block
		parentHash = block.Hash()
	}

	return blocks
}

func TestBlockService_SyncBlockRangeToDB_AllSuccess(t *testing.T) {
	svc, repo, rpc := setupTestServiceWithStartBlock(t, 10)

	rpc.blocks = newLinkedTestBlocks(10, 11, 12)

	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 12)
	switch {
	case err != nil:
		t.Fatalf("expected nil error, got %v", err)
	case result.Requested != 3:
		t.Fatalf("expected requested=3, got %d", result.Requested)
	case result.Succeeded != 3:
		t.Fatalf("expected succeeded=3, got %d", result.Succeeded)
	case result.Failed != 0:
		t.Fatalf("expected failed=0, got %d", result.Failed)
	case len(repo.insertedBlocks) != 3:
		t.Fatalf("expected 3 inserted blocks, got %d", len(repo.insertedBlocks))
	}
}

func TestBlockService_SyncBlockRangeToDB_ErrInvalidBlockRange(t *testing.T) {
	svc, _, _ := setupTestService(t)

	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 9)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrInvalidBlockRange) {
		t.Fatalf("expected ErrInvalidBlockRange, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil block, got %v", result)
	}
}

func TestBlockService_SyncBlockRangeToDB_ErrBlockRangeTooLarge(t *testing.T) {
	svc, _, _ := setupTestService(t)

	result, err := svc.SyncBlockRangeToDB(context.Background(), 1, 200)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrBlockRangeTooLarge) {
		t.Fatalf("expected ErrBlockRangeTooLarge, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil block, got %v", result)
	}
}

func TestBlockService_SyncBlockRangeToDB_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, repo, rpc := setupTestServiceWithStartBlock(t, 10)

	rpc.blocks = newLinkedTestBlocks(10, 11, 12)

	rpc.onGetBlock = func(number uint64) {
		if number == 11 {
			cancel()
		}
	}

	result, err := svc.SyncBlockRangeToDB(ctx, 10, 12)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrRequestCanceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	if result.Succeeded != 2 {
		t.Fatalf("expected succeeded=2, got %d", result.Succeeded)
	}
	if len(repo.insertedBlocks) != 2 {
		t.Fatalf("expected 2 inserted blocks, got %d", len(repo.insertedBlocks))
	}
}

func TestBlockService_SyncBlockRangeToDB_PartialFailureDoesNotCreateDiscontinuousChain(t *testing.T) {
	svc, repo, rpc := setupTestServiceWithStartBlock(t, 10)

	blocks := newLinkedTestBlocks(10, 11, 12)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: blocks[10],
		12: blocks[12],
	}
	rpc.errs = map[uint64]error{
		11: errors.New("some error"),
	}

	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 12)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrChainDiscontinuity) {
		t.Fatalf("expected ErrChainDiscontinuity, got %v", err)
	}

	if result == nil {
		t.Fatalf("expected result, got nil")
	}

	if result.Succeeded != 1 {
		t.Fatalf("expected succeeded=1, got %d", result.Succeeded)
	}
	if result.Failed != 2 {
		t.Fatalf("expected failed=2, got %d", result.Failed)
	}
	if len(result.FailedBlocks) != 2 || result.FailedBlocks[0] != 11 || result.FailedBlocks[1] != 12 {
		t.Fatalf("expected failed blocks [11 12], got %+v", result.FailedBlocks)
	}
	if len(repo.insertedBlocks) != 1 {
		t.Fatalf("expected 1 inserted block, got %d", len(repo.insertedBlocks))
	}
}

func TestBlockService_SyncBlockRangeToDB_ReturnsErrWhenReorgDetected(t *testing.T) {
	svc, repo, rpc := setupTestService(t)
	rpc.blocks = map[uint64]*ethtypes.Block{
		100: ethtypes.NewBlockWithHeader(&ethtypes.Header{
			Number: big.NewInt(100),
		}),
		101: ethtypes.NewBlockWithHeader(&ethtypes.Header{
			Number: big.NewInt(101),
		}),
	}
	repo.blocks = map[uint64]*models.Block{
		100: {
			Number: 100,
			Hash:   common.HexToHash("0xbbbb").Hex(),
		},
	}
	result, err := svc.SyncBlockRangeToDB(context.Background(), 100, 101)
	if !errors.Is(err, types.ErrReorgDetected) {
		t.Fatalf("expected ErrReorgDetected, got %v", err)
	}

	if result == nil {
		t.Fatalf("expected result, got nil")
	}

	if result.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", result.Failed)
	}

	if len(result.FailedBlocks) != 1 || result.FailedBlocks[0] != 100 {
		t.Fatalf("expected failed block 100, got %+v", result.FailedBlocks)
	}
}

func TestBlockService_SyncBlockRangeToDB_ReturnsErrWhenChainDiscontinuityDetected(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpcParentHash := common.HexToHash("0xaaa")
	dbParentHash := common.HexToHash("0xbbb")

	rpc.blocks = map[uint64]*ethtypes.Block{
		100: ethtypes.NewBlockWithHeader(&ethtypes.Header{
			Number:     big.NewInt(100),
			ParentHash: rpcParentHash,
		}),
		101: ethtypes.NewBlockWithHeader(&ethtypes.Header{
			Number: big.NewInt(101),
		}),
	}

	repo.blocks = map[uint64]*models.Block{
		99: {
			Number: 99,
			Hash:   dbParentHash.Hex(),
		},
	}
	result, err := svc.SyncBlockRangeToDB(context.Background(), 100, 101)
	if !errors.Is(err, types.ErrChainDiscontinuity) {
		t.Fatalf("expected ErrChainDiscontinuity, got %v", err)
	}

	if result == nil {
		t.Fatalf("expected result, got nil")
	}

	if result.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", result.Failed)
	}

	if len(result.FailedBlocks) != 1 || result.FailedBlocks[0] != 100 {
		t.Fatalf("expected failed block 100, got %+v", result.FailedBlocks)
	}
}

func TestBlockService_SyncBlockToDB_DoesNotSyncReceiptsWhenInsertBlockWithTransactionsFails(t *testing.T) {
	svc, repo, rpc, receiptSyncer := setupTestServiceWithReceiptSyncer(t)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	insertErr := errors.New("insert block with transactions error")
	repo.insertBlockWithTransactionsErr = insertErr

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, insertErr) {
		t.Fatalf("expected insert error, got %v", err)
	}

	if !strings.Contains(err.Error(), "insert block 20 into db") {
		t.Fatalf("expected wrapped insert error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts not to be called")
	}

	if receiptSyncer.calls != 0 {
		t.Fatalf("expected 0 receipt sync calls, got %d", receiptSyncer.calls)
	}

	if repo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced not to be called")
	}

	if repo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called")
	}
}

func TestBlockService_SyncBlockToDB_EmptyBlock_CompletesWithoutReceiptSync(t *testing.T) {
	svc, repo, rpc, receiptSyncer := setupTestServiceWithReceiptSyncer(t)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertBlockArg.Number)
	}

	if len(repo.insertTxsArg) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(repo.insertTxsArg))
	}

	assertInsertedBlockSyncState(
		t,
		repo.insertBlockArg,
		true,
		true,
		models.BlockSyncStatusCompleted,
	)

	if rpc.getChainIDCalled {
		t.Fatalf("expected GetChainID not to be called for empty block")
	}

	if receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts not to be called for empty block")
	}

	if receiptSyncer.calls != 0 {
		t.Fatalf("expected 0 receipt sync calls, got %d", receiptSyncer.calls)
	}

	if repo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced not to be called for empty block")
	}

	if repo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called for empty block")
	}
}

func TestBlockService_SyncBlockToDB_WithTransactions_SyncsReceiptsAndMarksCompleted(t *testing.T) {
	svc, repo, rpc, receiptSyncer := setupTestServiceWithReceiptSyncer(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if repo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertBlockArg.Number)
	}

	if len(repo.insertTxsArg) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.insertTxsArg))
	}

	assertInsertedBlockSyncState(
		t,
		repo.insertBlockArg,
		true,
		false,
		models.BlockSyncStatusTransactionsSynced,
	)

	if !receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts to be called")
	}

	if receiptSyncer.calls != 1 {
		t.Fatalf("expected 1 receipt sync call, got %d", receiptSyncer.calls)
	}

	if receiptSyncer.gotBlockNumber != 20 {
		t.Fatalf("expected receipt sync block number=20, got %d", receiptSyncer.gotBlockNumber)
	}

	if !repo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced to be called")
	}

	if repo.markBlockReceiptsSyncedBlockNumber != 20 {
		t.Fatalf("expected MarkBlockReceiptsSynced block number=20, got %d",
			repo.markBlockReceiptsSyncedBlockNumber,
		)
	}

	if repo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called")
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrorAndMarksFailedWhenReceiptSyncFails(t *testing.T) {
	svc, repo, rpc, receiptSyncer := setupTestServiceWithReceiptSyncer(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	receiptErr := errors.New("receipt sync error")
	receiptSyncer.err = receiptErr

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, receiptErr) {
		t.Fatalf("expected receipt sync error, got %v", err)
	}

	if !strings.Contains(err.Error(), "receipt") {
		t.Fatalf("expected wrapped receipt error, got %v", err)
	}

	if !strings.Contains(err.Error(), "block 20") {
		t.Fatalf("expected error to include block number, got %v", err)
	}

	if !repo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	assertInsertedBlockSyncState(
		t,
		repo.insertBlockArg,
		true,
		false,
		models.BlockSyncStatusTransactionsSynced,
	)

	if !receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts to be called")
	}

	if receiptSyncer.calls != 1 {
		t.Fatalf("expected 1 receipt sync call, got %d", receiptSyncer.calls)
	}

	if receiptSyncer.gotBlockNumber != 20 {
		t.Fatalf("expected receipt sync block number=20, got %d", receiptSyncer.gotBlockNumber)
	}

	if !repo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed to be called")
	}

	if repo.markBlockReceiptsSyncFailedBlockNumber != 20 {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed block number=20, got %d",
			repo.markBlockReceiptsSyncFailedBlockNumber,
		)
	}

	if repo.markBlockReceiptsSyncFailedReason != receiptErr.Error() {
		t.Fatalf("expected failure reason=%q, got %q",
			receiptErr.Error(),
			repo.markBlockReceiptsSyncFailedReason,
		)
	}

	if repo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced not to be called")
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrorWhenMarkBlockReceiptsSyncedFails(t *testing.T) {
	svc, repo, rpc, receiptSyncer := setupTestServiceWithReceiptSyncer(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	markErr := errors.New("mark receipts synced error")
	repo.markBlockReceiptsSyncedErr = markErr

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, markErr) {
		t.Fatalf("expected mark receipts synced error, got %v", err)
	}

	if !receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts to be called")
	}

	if receiptSyncer.calls != 1 {
		t.Fatalf("expected 1 receipt sync call, got %d", receiptSyncer.calls)
	}

	if !repo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced to be called")
	}

	if repo.markBlockReceiptsSyncedBlockNumber != 20 {
		t.Fatalf("expected MarkBlockReceiptsSynced block number=20, got %d",
			repo.markBlockReceiptsSyncedBlockNumber,
		)
	}

	if repo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called")
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrorWhenMarkBlockReceiptsSyncFailedFails(t *testing.T) {
	svc, repo, rpc, receiptSyncer := setupTestServiceWithReceiptSyncer(t)

	chainID := big.NewInt(1)
	rpc.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	rpc.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	rpc.err = nil

	receiptErr := errors.New("receipt sync error")
	markErr := errors.New("mark receipts sync failed error")

	receiptSyncer.err = receiptErr
	repo.markBlockReceiptsSyncFailedErr = markErr

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, receiptErr) {
		t.Fatalf("expected original receipt sync error, got %v", err)
	}

	if !strings.Contains(err.Error(), markErr.Error()) {
		t.Fatalf("expected error to include mark failure %q, got %v", markErr.Error(), err)
	}

	if !repo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed to be called")
	}

	if repo.markBlockReceiptsSyncFailedBlockNumber != 20 {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed block number=20, got %d",
			repo.markBlockReceiptsSyncFailedBlockNumber,
		)
	}

	if repo.markBlockReceiptsSyncFailedReason != receiptErr.Error() {
		t.Fatalf("expected failure reason=%q, got %q",
			receiptErr.Error(),
			repo.markBlockReceiptsSyncFailedReason,
		)
	}

	if repo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced not to be called")
	}
}

func TestBlockService_SyncBlockToDB_AllowsInsertWhenTransactionDoesNotExist(t *testing.T) {
	env := setupBlockServiceTestEnv(t, 20)

	chainID := big.NewInt(1)
	env.blockRPC.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	env.blockRPC.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	env.blockRPC.err = nil

	env.txRepo.existingTxs = map[string]*models.Transaction{}

	err := env.svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !env.txRepo.called {
		t.Fatalf("expected GetTransactionsByHashes to be called")
	}

	if len(env.txRepo.hashes) != 1 {
		t.Fatalf("expected 1 hash lookup, got %d", len(env.txRepo.hashes))
	}

	if env.txRepo.hashes[0] != signedTx.Hash().Hex() {
		t.Fatalf("expected hash=%s, got %s", signedTx.Hash().Hex(), env.txRepo.hashes[0])
	}

	if !env.blockRepo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if !env.receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts to be called")
	}

	if !env.blockRepo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced to be called")
	}
}

func TestBlockService_SyncBlockToDB_AllowsIdempotentTransactionWhenExistingLocationMatches(t *testing.T) {
	env := setupBlockServiceTestEnv(t, 20)

	chainID := big.NewInt(1)
	env.blockRPC.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	block := newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	env.blockRPC.block = block
	env.blockRPC.err = nil

	txHash := signedTx.Hash().Hex()

	env.txRepo.existingTxs = map[string]*models.Transaction{
		txHash: {
			Hash:        txHash,
			BlockNumber: 20,
			BlockHash:   block.Hash().Hex(),
			TxIndex:     0,
		},
	}

	err := env.svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !env.txRepo.called {
		t.Fatalf("expected GetTransactionsByHashes to be called")
	}

	if !env.blockRepo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if len(env.blockRepo.insertTxsArg) != 1 {
		t.Fatalf("expected 1 tx passed to InsertBlockWithTransactions, got %d", len(env.blockRepo.insertTxsArg))
	}

	if !env.receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts to be called")
	}

	if !env.blockRepo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced to be called")
	}

	if env.blockRepo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called")
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrChainDataConflictWhenExistingTransactionLocationDiffers(t *testing.T) {
	env := setupBlockServiceTestEnv(t, 20)

	chainID := big.NewInt(1)
	env.blockRPC.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	block := newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	env.blockRPC.block = block
	env.blockRPC.err = nil

	txHash := signedTx.Hash().Hex()

	env.txRepo.existingTxs = map[string]*models.Transaction{
		txHash: {
			Hash:        txHash,
			BlockNumber: 19,
			BlockHash:   common.HexToHash("0xoldblock").Hex(),
			TxIndex:     0,
		},
	}

	err := env.svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, types.ErrChainDataConflict) {
		t.Fatalf("expected ErrChainDataConflict, got %v", err)
	}

	if !strings.Contains(err.Error(), txHash) {
		t.Fatalf("expected error to include tx hash %s, got %v", txHash, err)
	}

	if env.blockRepo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}

	if env.receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts not to be called")
	}

	if env.blockRepo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced not to be called")
	}

	if env.blockRepo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called")
	}
}

func TestBlockService_SyncBlockToDB_ReturnsErrorWhenQueryExistingTransactionsFails(t *testing.T) {
	env := setupBlockServiceTestEnv(t, 20)

	chainID := big.NewInt(1)
	env.blockRPC.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	env.blockRPC.block = newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	env.blockRPC.err = nil

	txRepoErr := errors.New("query existing txs error")
	env.txRepo.err = txRepoErr

	err := env.svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, txRepoErr) {
		t.Fatalf("expected tx repo error, got %v", err)
	}

	if !strings.Contains(err.Error(), "query existing transactions by hashes") {
		t.Fatalf("expected wrapped tx lookup error, got %v", err)
	}

	if env.blockRepo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions not to be called")
	}

	if env.receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts not to be called")
	}
}

func TestBlockService_SyncBlockToDB_RetriesReceiptsFailedBlockAndMarksCompleted(t *testing.T) {
	env := setupBlockServiceTestEnv(t, 20)

	chainID := big.NewInt(1)
	env.blockRPC.chainID = chainID

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signedTx, _ := newSignedTestTransaction(t, chainID, 1, to)

	block := newTestBlockWithTransactions(20, []*ethtypes.Transaction{
		signedTx,
	})
	env.blockRPC.block = block
	env.blockRPC.err = nil

	previousErr := "previous receipt sync error"

	env.blockRepo.block = &models.Block{
		Number:             20,
		Hash:               block.Hash().Hex(),
		TransactionsSynced: true,
		ReceiptsSynced:     false,
		SyncStatus:         models.BlockSyncStatusReceiptsFailed,
		LastSyncError:      &previousErr,
	}
	env.blockRepo.found = true

	txHash := signedTx.Hash().Hex()
	env.txRepo.existingTxs = map[string]*models.Transaction{
		txHash: {
			Hash:        txHash,
			BlockNumber: 20,
			BlockHash:   block.Hash().Hex(),
			TxIndex:     0,
		},
	}

	err := env.svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !env.txRepo.called {
		t.Fatalf("expected GetTransactionsByHashes to be called")
	}

	if len(env.txRepo.hashes) != 1 {
		t.Fatalf("expected 1 hash lookup, got %d", len(env.txRepo.hashes))
	}

	if env.txRepo.hashes[0] != txHash {
		t.Fatalf("expected hash=%s, got %s", txHash, env.txRepo.hashes[0])
	}

	if !env.blockRepo.insertBlockWithTransactionsCalled {
		t.Fatalf("expected InsertBlockWithTransactions to be called")
	}

	if env.blockRepo.insertBlockArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if env.blockRepo.insertBlockArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", env.blockRepo.insertBlockArg.Number)
	}

	assertInsertedBlockSyncState(
		t,
		env.blockRepo.insertBlockArg,
		true,
		false,
		models.BlockSyncStatusTransactionsSynced,
	)

	if len(env.blockRepo.insertTxsArg) != 1 {
		t.Fatalf("expected 1 tx passed to InsertBlockWithTransactions, got %d", len(env.blockRepo.insertTxsArg))
	}

	if !env.receiptSyncer.called {
		t.Fatalf("expected SyncBlockTransactionReceipts to be called")
	}

	if env.receiptSyncer.calls != 1 {
		t.Fatalf("expected 1 receipt sync call, got %d", env.receiptSyncer.calls)
	}

	if env.receiptSyncer.gotBlockNumber != 20 {
		t.Fatalf("expected receipt sync block number=20, got %d", env.receiptSyncer.gotBlockNumber)
	}

	if !env.blockRepo.markBlockReceiptsSyncedCalled {
		t.Fatalf("expected MarkBlockReceiptsSynced to be called")
	}

	if env.blockRepo.markBlockReceiptsSyncedBlockNumber != 20 {
		t.Fatalf("expected MarkBlockReceiptsSynced block number=20, got %d",
			env.blockRepo.markBlockReceiptsSyncedBlockNumber,
		)
	}

	if env.blockRepo.markBlockReceiptsSyncFailedCalled {
		t.Fatalf("expected MarkBlockReceiptsSyncFailed not to be called")
	}
}

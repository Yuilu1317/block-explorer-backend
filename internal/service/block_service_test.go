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

type fakeBlockRPC struct {
	block *ethtypes.Block
	err   error

	blocks map[uint64]*ethtypes.Block
	errs   map[uint64]error

	onGetBlock func(number uint64)

	chainID          *big.Int
	chainIDErr       error
	getChainIDCalled bool
}

func (f *fakeBlockRPC) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
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

func (f *fakeBlockRPC) GetChainID(ctx context.Context) (*big.Int, error) {
	f.getChainIDCalled = true
	if f.chainIDErr != nil {
		return nil, f.chainIDErr
	}
	if f.chainID != nil {
		return f.chainID, nil
	}
	return big.NewInt(1), nil
}

type fakeBlockRepo struct {
	block *models.Block
	found bool
	err   error

	blocks map[uint64]*models.Block

	insertBlockWithTransactionsCalled bool
	insertBlockArg                    *models.Block
	insertTxsArg                      []*models.Transaction
	insertBlockWithTransactionsErr    error
	insertedBlocks                    map[uint64]*models.Block
}

func (f *fakeBlockRepo) InsertBlockWithTransactions(
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

func (f *fakeBlockRepo) GetBlockByNumber(ctx context.Context, number uint64) (*models.Block, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}

	if f.blocks != nil {
		block, found := f.blocks[number]
		if found {
			return block, true, nil
		}
		return block, found, nil
	}

	if f.found && f.block != nil && f.block.Number == number {
		return f.block, true, nil
	}
	return nil, false, nil
}

func setupTestService(t *testing.T) (*BlockService, *fakeBlockRepo, *fakeBlockRPC) {
	t.Helper()

	rpc := &fakeBlockRPC{}
	r := &fakeBlockRepo{}

	svc := NewBlockService(rpc, r)

	return svc, r, rpc
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
	svc, repo, rpc := setupTestService(t)
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

func TestBlockService_SyncBlockRangeToDB_AllSuccess(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(10)}),
		11: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(11)}),
		12: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(12)}),
	}

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

	svc, repo, rpc := setupTestService(t)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(10)}),
		11: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(11)}),
		12: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(12)}),
	}

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

func TestBlockService_SyncBlockRangeToDB_PartialFailure(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(10)}),
		12: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(12)}),
	}
	rpc.errs = map[uint64]error{
		11: errors.New("some error"),
	}
	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 12)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Succeeded != 2 {
		t.Fatalf("expected succeeded=2, got %d", result.Succeeded)
	}
	if result.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", result.Failed)
	}
	if len(result.FailedBlocks) != 1 || result.FailedBlocks[0] != 11 {
		t.Fatalf("expected failed block 11, got %+v", result.FailedBlocks)
	}
	if len(repo.insertedBlocks) != 2 {
		t.Fatalf("expected 2 inserted blocks, got %d", len(repo.insertedBlocks))
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

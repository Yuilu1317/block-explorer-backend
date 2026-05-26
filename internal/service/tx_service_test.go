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
)

type fakeTxServiceRPC struct {
	txRaw *types.TxRaw
	err   error

	called  bool
	gotHash string

	receipts     map[string]*ethtypes.Receipt
	receiptErrs  map[string]error
	receiptCalls int
}

func (f *fakeTxServiceRPC) GetTransactionByHash(ctx context.Context, hash string) (*types.TxRaw, error) {
	f.called = true
	f.gotHash = hash

	if f.err != nil {
		return nil, f.err
	}

	return f.txRaw, nil
}

func (f *fakeTxServiceRPC) GetTransactionReceipt(ctx context.Context, hash string) (*ethtypes.Receipt, error) {
	f.called = true
	f.gotHash = hash
	f.receiptCalls++

	if f.receiptErrs != nil {
		if err, ok := f.receiptErrs[hash]; ok {
			return nil, err
		}
	}
	if f.receipts != nil {
		return f.receipts[hash], nil
	}
	return nil, nil
}

type updateReceiptCall struct {
	hash    string
	status  *uint64
	gasUsed *uint64
}
type fakeTxServiceRepo struct {
	tx    *models.Transaction
	found bool
	err   error

	called  bool
	gotHash string

	txs            []*models.Transaction
	listErr        error
	listCalled     bool
	gotBlockNumber uint64

	updateErr   error
	updateCalls []updateReceiptCall
}

func (f *fakeTxServiceRepo) GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, bool, error) {
	f.called = true
	f.gotHash = hash

	if f.err != nil {
		return nil, false, f.err
	}
	return f.tx, f.found, nil
}

func (f *fakeTxServiceRepo) ListTransactionsMissingReceiptByBlockNumber(
	ctx context.Context,
	blockNumber uint64,
) ([]*models.Transaction, error) {
	f.listCalled = true
	f.gotBlockNumber = blockNumber
	if f.listErr != nil {
		return nil, f.listErr
	}

	return f.txs, nil
}

func (f *fakeTxServiceRepo) UpdateTransactionReceiptByHash(
	ctx context.Context,
	hash string,
	status *uint64,
	gasUsed *uint64,
) error {
	f.updateCalls = append(f.updateCalls, updateReceiptCall{
		hash:    hash,
		status:  status,
		gasUsed: gasUsed,
	})

	if f.updateErr != nil {
		return f.updateErr
	}

	return nil
}

func setupTxTestService(t *testing.T) (*TxService, *fakeTxServiceRepo, *fakeTxServiceRPC) {
	t.Helper()

	rpc := &fakeTxServiceRPC{}
	repo := &fakeTxServiceRepo{}

	svc := NewTxService(rpc, repo)

	return svc, repo, rpc
}

func validTxServiceTxHash() string {
	return "0x" + strings.Repeat("a", 64)
}

func validTxServiceBlockHash() string {
	return "0x" + strings.Repeat("b", 64)
}

func testTransactionModel(hash string) *models.Transaction {
	return &models.Transaction{
		Hash:        hash,
		BlockNumber: 100,
		BlockHash:   validTxServiceBlockHash(),
		TxIndex:     2,

		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",

		Nonce:       7,
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0xabcdef",
	}
}

func testTxRaw() *types.TxRaw {
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	tx := ethtypes.NewTransaction(
		7,
		to,
		big.NewInt(1000000000000000000),
		21000,
		big.NewInt(1000000000),
		[]byte{0xab, 0xcd},
	)

	return &types.TxRaw{
		Tx:        tx,
		From:      "0x1111111111111111111111111111111111111111",
		IsPending: false,
		Receipt:   nil,
	}
}

func testReceiptForTransaction(tx *models.Transaction, status uint64, gasUsed uint64) *ethtypes.Receipt {
	return &ethtypes.Receipt{
		TxHash:      common.HexToHash(tx.Hash),
		BlockHash:   common.HexToHash(tx.BlockHash),
		BlockNumber: new(big.Int).SetUint64(tx.BlockNumber),
		Status:      status,
		GasUsed:     gasUsed,
	}
}

func TestTxService_GetIndexedTransactionByHash_Success(t *testing.T) {
	svc, repo, _ := setupTxTestService(t)
	hash := validTxServiceTxHash()

	repo.tx = testTransactionModel(hash)
	repo.found = true

	result, err := svc.GetIndexedTransactionByHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !repo.called {
		t.Fatal("expected repo to be called")
	}

	if repo.gotHash != hash {
		t.Fatalf("expected repo hash %q, got %q", hash, repo.gotHash)
	}

	if result.Hash != repo.tx.Hash {
		t.Fatalf("expected hash %q, got %q", repo.tx.Hash, result.Hash)
	}

	if result.BlockNumber != repo.tx.BlockNumber {
		t.Fatalf("expected block number %d, got %d", repo.tx.BlockNumber, result.BlockNumber)
	}

	if result.BlockHash != repo.tx.BlockHash {
		t.Fatalf("expected block hash %q, got %q", repo.tx.BlockHash, result.BlockHash)
	}

	if result.TxIndex != repo.tx.TxIndex {
		t.Fatalf("expected tx index %d, got %d", repo.tx.TxIndex, result.TxIndex)
	}

	if result.FromAddress != repo.tx.FromAddress {
		t.Fatalf("expected from address %q, got %q", repo.tx.FromAddress, result.FromAddress)
	}

	if result.ToAddress != repo.tx.ToAddress {
		t.Fatalf("expected to address %q, got %q", repo.tx.ToAddress, result.ToAddress)
	}

	if result.Nonce != repo.tx.Nonce {
		t.Fatalf("expected nonce %d, got %d", repo.tx.Nonce, result.Nonce)
	}

	if result.ValueWei != repo.tx.ValueWei {
		t.Fatalf("expected value wei %q, got %q", repo.tx.ValueWei, result.ValueWei)
	}

	if result.GasLimit != repo.tx.GasLimit {
		t.Fatalf("expected gas limit %d, got %d", repo.tx.GasLimit, result.GasLimit)
	}

	if result.GasPriceWei != repo.tx.GasPriceWei {
		t.Fatalf("expected gas price wei %q, got %q", repo.tx.GasPriceWei, result.GasPriceWei)
	}

	if result.InputData != repo.tx.InputData {
		t.Fatalf("expected input data %q, got %q", repo.tx.InputData, result.InputData)
	}
}

func TestTxService_GetIndexedTransactionByHash_PreservesReceiptStatusNilZeroOne(t *testing.T) {
	statusZero := uint64(0)
	statusOne := uint64(1)
	gasUsed := uint64(21000)

	tests := []struct {
		name           string
		setupTx        func(hash string) *models.Transaction
		wantStatusNil  bool
		wantStatus     uint64
		wantGasUsedNil bool
		wantGasUsed    uint64
	}{
		{
			name: "receipt not synced keeps nil status and nil gas used",
			setupTx: func(hash string) *models.Transaction {
				tx := testTransactionModel(hash)
				tx.ReceiptStatus = nil
				tx.ReceiptGasUsed = nil
				return tx
			},
			wantStatusNil:  true,
			wantGasUsedNil: true,
		},
		{
			name: "failed receipt keeps status zero",
			setupTx: func(hash string) *models.Transaction {
				tx := testTransactionModel(hash)
				tx.ReceiptStatus = &statusZero
				tx.ReceiptGasUsed = &gasUsed
				return tx
			},
			wantStatusNil:  false,
			wantStatus:     0,
			wantGasUsedNil: false,
			wantGasUsed:    gasUsed,
		},
		{
			name: "successful receipt keeps status one",
			setupTx: func(hash string) *models.Transaction {
				tx := testTransactionModel(hash)
				tx.ReceiptStatus = &statusOne
				tx.ReceiptGasUsed = &gasUsed
				return tx
			},
			wantStatusNil:  false,
			wantStatus:     1,
			wantGasUsedNil: false,
			wantGasUsed:    gasUsed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, rpc := setupTxTestService(t)

			hash := validTxServiceTxHash()
			repo.tx = tt.setupTx(hash)
			repo.found = true

			got, err := svc.GetIndexedTransactionByHash(context.Background(), hash)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if got == nil {
				t.Fatalf("expected result, got nil")
			}

			if !repo.called {
				t.Fatalf("expected repo to be called")
			}

			if repo.gotHash != hash {
				t.Fatalf("expected repo hash=%s, got %s", hash, repo.gotHash)
			}

			if rpc.called {
				t.Fatalf("expected rpc not to be called")
			}

			if got.Hash != hash {
				t.Fatalf("expected hash=%s, got %s", hash, got.Hash)
			}

			if tt.wantStatusNil {
				if got.Status != nil {
					t.Fatalf("expected nil status, got %d", *got.Status)
				}
			} else {
				if got.Status == nil {
					t.Fatalf("expected status=%d, got nil", tt.wantStatus)
				}
				if *got.Status != tt.wantStatus {
					t.Fatalf("expected status=%d, got %d", tt.wantStatus, *got.Status)
				}
			}

			if tt.wantGasUsedNil {
				if got.GasUsed != nil {
					t.Fatalf("expected nil gas used, got %d", *got.GasUsed)
				}
			} else {
				if got.GasUsed == nil {
					t.Fatalf("expected gas_used=%d, got nil", tt.wantGasUsed)
				}
				if *got.GasUsed != tt.wantGasUsed {
					t.Fatalf("expected gas_used=%d, got %d", tt.wantGasUsed, *got.GasUsed)
				}
			}
		})
	}
}

func TestTxService_GetIndexedTransactionByHash_ReturnsErrInvalidTxHash(t *testing.T) {
	svc, repo, _ := setupTxTestService(t)

	result, err := svc.GetIndexedTransactionByHash(context.Background(), "0x123")

	if !errors.Is(err, types.ErrInvalidTxHash) {
		t.Fatalf("expected ErrInvalidTxHash, got %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if repo.called {
		t.Fatal("expected repo not to be called")
	}
}

func TestTxService_GetIndexedTransactionByHash_ReturnsErrTxNotFoundWhenNotFound(t *testing.T) {
	svc, repo, _ := setupTxTestService(t)

	hash := validTxServiceTxHash()
	repo.tx = nil
	repo.found = false

	result, err := svc.GetIndexedTransactionByHash(context.Background(), hash)

	if !errors.Is(err, types.ErrTxNotFound) {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if !repo.called {
		t.Fatal("expected repo to be called")
	}
}

func TestTxService_GetIndexedTransactionByHash_ReturnsRepoError(t *testing.T) {
	svc, repo, _ := setupTxTestService(t)

	hash := validTxServiceTxHash()
	repoErr := errors.New("db error")
	repo.err = repoErr

	result, err := svc.GetIndexedTransactionByHash(context.Background(), hash)

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if !repo.called {
		t.Fatal("expected repo to be called")
	}
}

func TestTxService_GetIndexedTransactionByHash_NormalizesHashBeforeQuery(t *testing.T) {
	svc, repo, _ := setupTxTestService(t)

	upperHash := "0x" + strings.Repeat("A", 64)
	expectedHash := strings.ToLower(upperHash)

	repo.tx = testTransactionModel(expectedHash)
	repo.found = true

	result, err := svc.GetIndexedTransactionByHash(context.Background(), "  "+upperHash+"\n")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !repo.called {
		t.Fatal("expected repo to be called")
	}

	if repo.gotHash != expectedHash {
		t.Fatalf("expected repo hash %q, got %q", expectedHash, repo.gotHash)
	}
}

func TestTxService_GetTxDetailByHashFromRPC_Success(t *testing.T) {
	svc, _, rpc := setupTxTestService(t)

	hash := validTxServiceTxHash()
	raw := testTxRaw()
	rpc.txRaw = raw

	result, err := svc.GetTxDetailByHashFromRPC(context.Background(), hash)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !rpc.called {
		t.Fatal("expected rpc to be called")
	}

	if rpc.gotHash != hash {
		t.Fatalf("expected rpc hash %q, got %q", hash, rpc.gotHash)
	}

	if result.Hash != raw.Tx.Hash().Hex() {
		t.Fatalf("expected hash %q, got %q", raw.Tx.Hash().Hex(), result.Hash)
	}

	if result.FromAddress != raw.From {
		t.Fatalf("expected from address %q, got %q", raw.From, result.FromAddress)
	}

	if result.ToAddress != raw.Tx.To().Hex() {
		t.Fatalf("expected to address %q, got %q", raw.Tx.To().Hex(), result.ToAddress)
	}

	if result.ValueWei != raw.Tx.Value().String() {
		t.Fatalf("expected value wei %q, got %q", raw.Tx.Value().String(), result.ValueWei)
	}

	if result.Nonce != raw.Tx.Nonce() {
		t.Fatalf("expected nonce %d, got %d", raw.Tx.Nonce(), result.Nonce)
	}

	if result.GasLimit != raw.Tx.Gas() {
		t.Fatalf("expected gas limit %d, got %d", raw.Tx.Gas(), result.GasLimit)
	}

	if result.GasPriceWei != raw.Tx.GasPrice().String() {
		t.Fatalf("expected gas price wei %q, got %q", raw.Tx.GasPrice().String(), result.GasPriceWei)
	}

	if result.Data != "0xabcd" {
		t.Fatalf("expected data %q, got %q", "0xabcd", result.Data)
	}

	if result.IsPending != raw.IsPending {
		t.Fatalf("expected is_pending %v, got %v", raw.IsPending, result.IsPending)
	}
}
func TestTxService_GetTxDetailByHashFromRPC_ReturnsErrInvalidTxHash(t *testing.T) {
	svc, _, rpc := setupTxTestService(t)

	result, err := svc.GetTxDetailByHashFromRPC(context.Background(), "abc")

	if !errors.Is(err, types.ErrInvalidTxHash) {
		t.Fatalf("expected ErrInvalidTxHash, got %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if rpc.called {
		t.Fatal("expected rpc not to be called")
	}
}

func TestTxService_GetTxDetailByHashFromRPC_ReturnsRPCError(t *testing.T) {
	svc, _, rpc := setupTxTestService(t)

	hash := validTxServiceTxHash()
	rpcErr := errors.New("rpc error")
	rpc.err = rpcErr

	result, err := svc.GetTxDetailByHashFromRPC(context.Background(), hash)

	if !errors.Is(err, rpcErr) {
		t.Fatalf("expected rpc error, got %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if !rpc.called {
		t.Fatal("expected rpc to be called")
	}
}

func TestTxService_GetTxDetailByHashFromRPC_NormalizesHashBeforeQuery(t *testing.T) {
	svc, _, rpc := setupTxTestService(t)

	upperHash := "0x" + strings.Repeat("A", 64)
	expectedHash := strings.ToLower(upperHash)
	rpc.txRaw = testTxRaw()

	result, err := svc.GetTxDetailByHashFromRPC(context.Background(), "  "+upperHash+"\n")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !rpc.called {
		t.Fatal("expected rpc to be called")
	}

	if rpc.gotHash != expectedHash {
		t.Fatalf("expected rpc hash %q, got %q", expectedHash, rpc.gotHash)
	}
}

func TestTxService_validateReceiptMatchesTransaction_Success(t *testing.T) {
	svc, _, _ := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	receipt := testReceiptForTransaction(tx, 1, 21000)

	if err := svc.validateReceiptMatchesTransaction(tx, receipt); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestTxService_validateReceiptMatchesTransaction_ReturnsErrorWhenReceiptNil(t *testing.T) {
	svc, _, _ := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())

	err := svc.validateReceiptMatchesTransaction(tx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "receipt is nil") {
		t.Fatalf("expected receipt nil error, got %v", err)
	}
}

func TestTxService_validateReceiptMatchesTransaction_ReturnsErrorWhenTxHashMismatch(t *testing.T) {
	svc, _, _ := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	receipt := testReceiptForTransaction(tx, 1, 21000)
	receipt.TxHash = common.HexToHash("0x" + strings.Repeat("c", 64))

	err := svc.validateReceiptMatchesTransaction(tx, receipt)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "receipt tx hash mismatch") {
		t.Fatalf("expected tx hash mismatch error, got %v", err)
	}
}

func TestTxService_validateReceiptMatchesTransaction_ReturnsErrorWhenBlockNumberNil(t *testing.T) {
	svc, _, _ := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	receipt := testReceiptForTransaction(tx, 1, 21000)
	receipt.BlockNumber = nil

	err := svc.validateReceiptMatchesTransaction(tx, receipt)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "receipt block number is nil") {
		t.Fatalf("expected block number nil error, got %v", err)
	}
}

func TestTxService_validateReceiptMatchesTransaction_ReturnsErrorWhenBlockNumberMismatch(t *testing.T) {
	svc, _, _ := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	receipt := testReceiptForTransaction(tx, 1, 21000)
	receipt.BlockNumber = new(big.Int).SetUint64(tx.BlockNumber + 1)

	err := svc.validateReceiptMatchesTransaction(tx, receipt)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "receipt block number mismatch") {
		t.Fatalf("expected block number mismatch error, got %v", err)
	}
}

func TestTxService_validateReceiptMatchesTransaction_ReturnsErrorWhenBlockHashMismatch(t *testing.T) {
	svc, _, _ := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	receipt := testReceiptForTransaction(tx, 1, 21000)
	receipt.BlockHash = common.HexToHash("0x" + strings.Repeat("d", 64))

	err := svc.validateReceiptMatchesTransaction(tx, receipt)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "receipt block hash mismatch") {
		t.Fatalf("expected block hash mismatch error, got %v", err)
	}
}

func TestTxService_SyncBlockTransactionReceipts_ReturnsListError(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	listErr := errors.New("db list error")
	repo.listErr = listErr

	err := svc.SyncBlockTransactionReceipts(context.Background(), 100)
	if !errors.Is(err, listErr) {
		t.Fatalf("expected list error, got %v", err)
	}
	if !strings.Contains(err.Error(), "service: list missing receipt transactions for block 100") {
		t.Fatalf("expected wrapped list error, got %v", err)
	}

	if !repo.listCalled {
		t.Fatal("expected list to be called")
	}
	if rpc.receiptCalls != 0 {
		t.Fatalf("expected no receipt rpc calls, got %d", rpc.receiptCalls)
	}
	if len(repo.updateCalls) != 0 {
		t.Fatalf("expected no update calls, got %d", len(repo.updateCalls))
	}
}

func TestTxService_SyncBlockTransactionReceipts_UpdatesStatusOne(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	repo.txs = []*models.Transaction{tx}

	rpc.receipts = map[string]*ethtypes.Receipt{
		tx.Hash: testReceiptForTransaction(tx, 1, 21000),
	}

	err := svc.SyncBlockTransactionReceipts(context.Background(), tx.BlockNumber)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.listCalled {
		t.Fatal("expected list to be called")
	}
	if repo.gotBlockNumber != tx.BlockNumber {
		t.Fatalf("expected block number %d, got %d", tx.BlockNumber, repo.gotBlockNumber)
	}

	if rpc.receiptCalls != 1 {
		t.Fatalf("expected 1 receipt rpc call, got %d", rpc.receiptCalls)
	}

	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}

	call := repo.updateCalls[0]
	if call.hash != tx.Hash {
		t.Fatalf("expected update hash %s, got %s", tx.Hash, call.hash)
	}
	if call.status == nil {
		t.Fatal("expected status, got nil")
	}
	if *call.status != uint64(1) {
		t.Fatalf("expected status=1, got %d", *call.status)
	}
	if call.gasUsed == nil {
		t.Fatal("expected gas used, got nil")
	}
	if *call.gasUsed != uint64(21000) {
		t.Fatalf("expected gas used=21000, got %d", *call.gasUsed)
	}
}

func TestTxService_SyncBlockTransactionReceipts_UpdatesStatusZero(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	repo.txs = []*models.Transaction{tx}

	rpc.receipts = map[string]*ethtypes.Receipt{
		tx.Hash: testReceiptForTransaction(tx, 0, 21000),
	}

	err := svc.SyncBlockTransactionReceipts(context.Background(), tx.BlockNumber)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}

	call := repo.updateCalls[0]
	if call.status == nil {
		t.Fatal("expected status, got nil")
	}
	if *call.status != uint64(0) {
		t.Fatalf("expected status=0, got %d", *call.status)
	}
	if call.gasUsed == nil {
		t.Fatal("expected gas used, got nil")
	}
	if *call.gasUsed != uint64(21000) {
		t.Fatalf("expected gas used=21000, got %d", *call.gasUsed)
	}
}

func TestTxService_SyncBlockTransactionReceipts_SkipsReceiptNotFound(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	repo.txs = []*models.Transaction{tx}

	rpc.receiptErrs = map[string]error{
		tx.Hash: types.ErrTxReceiptNotFound,
	}

	err := svc.SyncBlockTransactionReceipts(context.Background(), tx.BlockNumber)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if rpc.receiptCalls != 1 {
		t.Fatalf("expected 1 receipt rpc call, got %d", rpc.receiptCalls)
	}

	if len(repo.updateCalls) != 0 {
		t.Fatalf("expected no update calls, got %d", len(repo.updateCalls))
	}
}

func TestTxService_SyncBlockTransactionReceipts_ReturnsRPCError(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	repo.txs = []*models.Transaction{tx}

	rpcErr := errors.New("rpc down")
	rpc.receiptErrs = map[string]error{
		tx.Hash: rpcErr,
	}

	err := svc.SyncBlockTransactionReceipts(context.Background(), tx.BlockNumber)
	if !errors.Is(err, rpcErr) {
		t.Fatalf("expected rpc error, got %v", err)
	}
	if !strings.Contains(err.Error(), "service: get transaction receipt for tx") {
		t.Fatalf("expected wrapped rpc error, got %v", err)
	}

	if len(repo.updateCalls) != 0 {
		t.Fatalf("expected no update calls, got %d", len(repo.updateCalls))
	}
}

func TestTxService_SyncBlockTransactionReceipts_ReturnsValidationErrorWithoutUpdate(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	repo.txs = []*models.Transaction{tx}

	receipt := testReceiptForTransaction(tx, 1, 21000)
	receipt.TxHash = common.HexToHash("0x" + strings.Repeat("c", 64))

	rpc.receipts = map[string]*ethtypes.Receipt{
		tx.Hash: receipt,
	}

	err := svc.SyncBlockTransactionReceipts(context.Background(), tx.BlockNumber)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "service: validate transaction receipt for tx") {
		t.Fatalf("expected wrapped validation error, got %v", err)
	}

	if len(repo.updateCalls) != 0 {
		t.Fatalf("expected no update calls, got %d", len(repo.updateCalls))
	}
}

func TestTxService_SyncBlockTransactionReceipts_ReturnsUpdateError(t *testing.T) {
	svc, repo, rpc := setupTxTestService(t)

	tx := testTransactionModel(validTxServiceTxHash())
	repo.txs = []*models.Transaction{tx}

	rpc.receipts = map[string]*ethtypes.Receipt{
		tx.Hash: testReceiptForTransaction(tx, 1, 21000),
	}

	updateErr := errors.New("db update error")
	repo.updateErr = updateErr

	err := svc.SyncBlockTransactionReceipts(context.Background(), tx.BlockNumber)
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected update error, got %v", err)
	}
	if !strings.Contains(err.Error(), "service: update transaction receipt for tx") {
		t.Fatalf("expected wrapped update error, got %v", err)
	}

	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
}

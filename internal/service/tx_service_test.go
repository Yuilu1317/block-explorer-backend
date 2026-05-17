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
}

func (f *fakeTxServiceRPC) GetTransactionByHash(ctx context.Context, hash string) (*types.TxRaw, error) {
	f.called = true
	f.gotHash = hash

	if f.err != nil {
		return nil, f.err
	}

	return f.txRaw, nil
}

type fakeTxServiceRepo struct {
	tx    *models.Transaction
	found bool
	err   error

	called  bool
	gotHash string
}

func (f *fakeTxServiceRepo) GetTransactionByHash(ctx context.Context, hash string) (*models.Transaction, bool, error) {
	f.called = true
	f.gotHash = hash

	if f.err != nil {
		return nil, false, f.err
	}
	return f.tx, f.found, nil
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

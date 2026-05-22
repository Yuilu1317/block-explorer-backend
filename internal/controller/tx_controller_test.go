package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeTxService struct {
	txDetail  *types.TxDetailDTO
	indexedTx *types.IndexedTransactionDTO

	rpcErr     error
	indexedErr error

	calledRPC      bool
	calledIndexed  bool
	gotRPCHash     string
	gotIndexedHash string
}

func (f *fakeTxService) GetTxDetailByHashFromRPC(ctx context.Context, hash string) (*types.TxDetailDTO, error) {
	f.calledRPC = true
	f.gotRPCHash = hash

	if f.rpcErr != nil {
		return nil, f.rpcErr
	}

	return f.txDetail, nil
}

func (f *fakeTxService) GetIndexedTransactionByHash(ctx context.Context, hash string) (*types.IndexedTransactionDTO, error) {
	f.calledIndexed = true
	f.gotIndexedHash = hash

	if f.indexedErr != nil {
		return nil, f.indexedErr
	}

	return f.indexedTx, nil
}

func setupTestTxController(t *testing.T) (*TxController, *fakeTxService) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	svc := &fakeTxService{}
	ctl := NewTxController(svc)

	return ctl, svc
}

func setupTxRouter(ctl *TxController) *gin.Engine {
	r := gin.New()

	r.GET("/tx/:hash", ctl.GetTxDetailByHashFromRPC)
	r.GET("/indexed/tx/:hash", ctl.GetIndexedTransactionByHash)

	return r
}

func testTxControllerHash() string {
	return "0x" + strings.Repeat("a", 64)
}

func TestTxController_GetIndexedTransactionByHash_Success(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	svc.indexedTx = &types.IndexedTransactionDTO{
		Hash:        hash,
		BlockNumber: 100,
		BlockHash:   "0x" + strings.Repeat("b", 64),
		TxIndex:     2,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		Nonce:       7,
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0xabcdef",
	}

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/indexed/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledIndexed {
		t.Fatal("expected indexed service to be called")
	}

	if svc.gotIndexedHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotIndexedHash)
	}

	if svc.calledRPC {
		t.Fatal("expected rpc service not to be called")
	}

	if !strings.Contains(w.Body.String(), hash) {
		t.Fatalf("expected response body to contain hash %q, got %s", hash, w.Body.String())
	}
}

func TestTxController_GetIndexedTransactionByHash_ReturnsBadRequestWhenHashInvalid(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := "invalid-hash"
	svc.indexedErr = types.ErrInvalidTxHash

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/indexed/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledIndexed {
		t.Fatal("expected indexed service to be called")
	}

	if svc.gotIndexedHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotIndexedHash)
	}

	if svc.calledRPC {
		t.Fatal("expected rpc service not to be called")
	}
}

func TestTxController_GetIndexedTransactionByHash_ReturnsNotFound(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	svc.indexedErr = types.ErrTxNotFound

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/indexed/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledIndexed {
		t.Fatal("expected indexed service to be called")
	}

	if svc.gotIndexedHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotIndexedHash)
	}
}

func TestTxController_GetIndexedTransactionByHash_ReturnsInternalServerError(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	svc.indexedErr = errors.New("service error")

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/indexed/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledIndexed {
		t.Fatal("expected indexed service to be called")
	}

	if svc.gotIndexedHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotIndexedHash)
	}
}

func TestTxController_GetTxDetailByHashFromRPC_Success(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	svc.txDetail = &types.TxDetailDTO{
		Hash:        hash,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		ValueWei:    "1000000000000000000",
		Nonce:       7,
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		Data:        "0xabcdef",
		IsPending:   false,
	}

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledRPC {
		t.Fatal("expected rpc service to be called")
	}

	if svc.gotRPCHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotRPCHash)
	}

	if svc.calledIndexed {
		t.Fatal("expected indexed service not to be called")
	}

	if !strings.Contains(w.Body.String(), hash) {
		t.Fatalf("expected response body to contain hash %q, got %s", hash, w.Body.String())
	}
}

func TestTxController_GetTxDetailByHashFromRPC_ReturnsReceiptStatusZero(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	status := uint64(0)
	gasUsed := uint64(21000)
	blockNumber := uint64(100)

	svc.txDetail = &types.TxDetailDTO{
		Hash:        hash,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		ValueWei:    "1000000000000000000",
		Nonce:       7,
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		Data:        "0xabcdef",
		IsPending:   false,
		BlockNumber: &blockNumber,
		Status:      &status,
		GasUsed:     &gasUsed,
	}

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledRPC {
		t.Fatal("expected rpc service to be called")
	}

	if svc.gotRPCHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotRPCHash)
	}

	body := w.Body.String()

	if !strings.Contains(body, `"status":0`) {
		t.Fatalf("expected response body to contain status=0, got %s", body)
	}

	if !strings.Contains(body, `"gas_used":21000`) {
		t.Fatalf("expected response body to contain gas_used=21000, got %s", body)
	}
}

func TestTxController_GetTxDetailByHashFromRPC_ReturnsNilReceiptFields(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()

	svc.txDetail = &types.TxDetailDTO{
		Hash:        hash,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		ValueWei:    "1000000000000000000",
		Nonce:       7,
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		Data:        "0xabcdef",
		IsPending:   true,
		BlockNumber: nil,
		Status:      nil,
		GasUsed:     nil,
	}

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledRPC {
		t.Fatal("expected rpc service to be called")
	}

	body := w.Body.String()

	if !strings.Contains(body, `"status":null`) {
		t.Fatalf("expected response body to contain status=null, got %s", body)
	}

	if !strings.Contains(body, `"gas_used":null`) {
		t.Fatalf("expected response body to contain gas_used=null, got %s", body)
	}
}

func TestTxController_GetIndexedTransactionByHash_ReturnsReceiptStatusZero(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	status := uint64(0)
	gasUsed := uint64(21000)

	svc.indexedTx = &types.IndexedTransactionDTO{
		Hash:        hash,
		BlockNumber: 100,
		BlockHash:   "0x" + strings.Repeat("b", 64),
		TxIndex:     2,

		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",

		Status:  &status,
		GasUsed: &gasUsed,

		Nonce:       7,
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0xabcdef",
	}

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/indexed/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledIndexed {
		t.Fatal("expected indexed service to be called")
	}

	if svc.gotIndexedHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotIndexedHash)
	}

	if svc.calledRPC {
		t.Fatal("expected rpc service not to be called")
	}

	body := w.Body.String()

	if !strings.Contains(body, `"status":0`) {
		t.Fatalf("expected response body to contain status=0, got %s", body)
	}

	if !strings.Contains(body, `"gas_used":21000`) {
		t.Fatalf("expected response body to contain gas_used=21000, got %s", body)
	}
}

func TestTxController_GetIndexedTransactionByHash_ReturnsNilReceiptFields(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()

	svc.indexedTx = &types.IndexedTransactionDTO{
		Hash:        hash,
		BlockNumber: 100,
		BlockHash:   "0x" + strings.Repeat("b", 64),
		TxIndex:     2,

		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",

		Status:  nil,
		GasUsed: nil,

		Nonce:       7,
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0xabcdef",
	}

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/indexed/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledIndexed {
		t.Fatal("expected indexed service to be called")
	}

	body := w.Body.String()

	if !strings.Contains(body, `"status":null`) {
		t.Fatalf("expected response body to contain status=null, got %s", body)
	}

	if !strings.Contains(body, `"gas_used":null`) {
		t.Fatalf("expected response body to contain gas_used=null, got %s", body)
	}
}

func TestTxController_GetTxDetailByHashFromRPC_ReturnsBadRequestWhenHashInvalid(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := "invalid-hash"
	svc.rpcErr = types.ErrInvalidTxHash

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledRPC {
		t.Fatal("expected rpc service to be called")
	}

	if svc.gotRPCHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotRPCHash)
	}

	if svc.calledIndexed {
		t.Fatal("expected indexed service not to be called")
	}
}

func TestTxController_GetTxDetailByHashFromRPC_ReturnsInternalServerError(t *testing.T) {
	ctl, svc := setupTestTxController(t)

	hash := testTxControllerHash()
	svc.rpcErr = errors.New("rpc error")

	r := setupTxRouter(ctl)

	req := httptest.NewRequest(http.MethodGet, "/tx/"+hash, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledRPC {
		t.Fatal("expected rpc service to be called")
	}

	if svc.gotRPCHash != hash {
		t.Fatalf("expected hash %q, got %q", hash, svc.gotRPCHash)
	}
}

package rpc

import (
	"block-explorer-backend/internal/types"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type jsonRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      any               `json:"id"`
}

func newTestBlockRPC(t *testing.T, url string) *BlockRPC {
	t.Helper()

	ethClient, rpcClient, err := NewEthClient(url)
	if err != nil {
		t.Fatalf("new eth client: %v", err)
	}

	t.Cleanup(rpcClient.Close)

	TimeoutSeconds := 5

	baseRPC := NewBaseRPC(
		ethClient,
		rpcClient,
		time.Duration(TimeoutSeconds)*time.Second,
	)

	return NewBlockRPC(baseRPC)
}

func decodeJSONRPCRequest(t *testing.T, r *http.Request, expectedMethod string) jsonRPCRequest {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", r.Method)
	}

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode request: %v", err)
	}

	if req.Method != expectedMethod {
		t.Fatalf("expected method %s, got %s", expectedMethod, req.Method)
	}

	return req
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, resp any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestBlockRPC_GetLatestBlockNumber_ReturnsErrorWhenHTTPStatusNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)

	number, err := blockRPC.GetLatestBlockNumber(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if number != 0 {
		t.Fatalf("expected block number 0, got %d", number)
	}
}

func TestBlockRPC_GetLatestBlockNumber_ReturnsErrorWhenResponseBodyInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)

	number, err := blockRPC.GetLatestBlockNumber(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if number != 0 {
		t.Fatalf("expected block number 0, got %d", number)
	}
}

func TestBlockRPC_GetLatestBlockNumber_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_blockNumber")
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "0x10",
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)
	number, err := blockRPC.GetLatestBlockNumber(context.Background())
	if err != nil {
		t.Fatalf("GetLatestBlockNumber error: %v", err)
	}

	if number != 16 {
		t.Fatalf("expected block number 16, got %d", number)
	}
}

func TestBlockRPC_GetLatestBlockNumber_ReturnsErrorWhenRPCReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_blockNumber")
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32000,
				"message": "execution error",
			},
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)
	number, err := blockRPC.GetLatestBlockNumber(context.Background())
	if err == nil {
		t.Fatalf("GetLatestBlockNumber should return error")
	}
	if number != 0 {
		t.Fatalf("expected block number 0, got %d", number)
	}
	if !strings.Contains(err.Error(), "execution error") {
		t.Fatalf("expected error to contain execution error, got %v", err)
	}
}

func TestBlockRPC_GetLatestBlockNumber_ReturnsErrorWhenResultIsInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_blockNumber")
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "not-a-hex",
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)
	number, err := blockRPC.GetLatestBlockNumber(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if number != 0 {
		t.Fatalf("expected block number 0, got %d", number)
	}
}

func TestBlockRPC_GetBlockByNumber_ReturnsErrorWhenHTTPStatusNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	blockRPC := newTestBlockRPC(t, server.URL)

	block, err := blockRPC.GetBlockByNumber(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if block != nil {
		t.Fatalf("expected nil block, got %v", block)
	}
	if errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected transport error, got ErrBlockNotFound: %v", err)
	}
}

func TestBlockRPC_GetBlockByNumber_ReturnsErrorWhenResponseBodyInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(server.Close)
	blockRPC := newTestBlockRPC(t, server.URL)

	block, err := blockRPC.GetBlockByNumber(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if block != nil {
		t.Fatalf("expected nil block, got %v", block)
	}

	if errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected decode error, got ErrBlockNotFound: %v", err)
	}
}

func fakeBlockResult() map[string]any {
	zeroHash := "0x" + strings.Repeat("00", 32)
	oneHash := "0x" + strings.Repeat("11", 32)
	zeroBloom := "0x" + strings.Repeat("00", 256)

	return map[string]any{
		"number":           "0x1",
		"hash":             oneHash,
		"parentHash":       zeroHash,
		"nonce":            "0x0000000000000000",
		"sha3Uncles":       "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
		"logsBloom":        zeroBloom,
		"transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
		"stateRoot":        zeroHash,
		"receiptsRoot":     zeroHash,
		"miner":            "0x0000000000000000000000000000000000000001",
		"difficulty":       "0x0",
		"totalDifficulty":  "0x0",
		"extraData":        "0x",
		"size":             "0x1",
		"gasLimit":         "0x1c9c380",
		"gasUsed":          "0x5208",
		"timestamp":        "0x65",
		"transactions":     []any{},
		"uncles":           []any{},
		"baseFeePerGas":    "0x1",
	}
}

func TestBlockRPC_GetBlockByNumber_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_getBlockByNumber")
		if len(req.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(req.Params))
		}
		var blockNumber string
		if err := json.Unmarshal(req.Params[0], &blockNumber); err != nil {
			t.Fatalf("decode block number param: %v", err)
		}
		if blockNumber != "0x1" {
			t.Fatalf("expected block number param 0x1, got %s", blockNumber)
		}
		var fullTx bool
		if err := json.Unmarshal(req.Params[1], &fullTx); err != nil {
			t.Fatalf("decode fullTx param: %v", err)
		}

		if !fullTx {
			t.Fatalf("expected fullTx true")
		}

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  fakeBlockResult(),
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)

	block, err := blockRPC.GetBlockByNumber(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetBlockByNumber error: %v", err)
	}

	if block == nil {
		t.Fatal("expected block, got nil")
	}

	if block.NumberU64() != 1 {
		t.Fatalf("expected block number 1, got %d", block.NumberU64())
	}

	if len(block.Transactions()) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(block.Transactions()))
	}
}

func TestBlockRPC_GetBlockByNumber_ReturnsErrBlockNotFoundWhenResultIsNull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_getBlockByNumber")
		if len(req.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(req.Params))
		}
		var blockNumber string
		if err := json.Unmarshal(req.Params[0], &blockNumber); err != nil {
			t.Fatalf("decode block number param: %v", err)
		}

		if blockNumber != "0x3e7" {
			t.Fatalf("expected block number param 0x3e7, got %s", blockNumber)
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  nil,
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)
	blockRPC := newTestBlockRPC(t, server.URL)

	block, err := blockRPC.GetBlockByNumber(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if block != nil {
		t.Fatalf("expected nil block, got %v", block)
	}
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}
}

func TestBlockRPC_GetBlockByNumber_ReturnsErrorWhenRPCReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_getBlockByNumber")
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32602,
				"message": "invalid argument",
			},
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)
	blockRPC := newTestBlockRPC(t, server.URL)

	block, err := blockRPC.GetBlockByNumber(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if block != nil {
		t.Fatalf("expected nil block, got %v", block)
	}

	if !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("expected error to contain invalid argument, got %v", err)
	}
}

func TestBlockRPC_GetBlockNumberByTag_SafeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_getBlockByNumber")
		if len(req.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(req.Params))
		}

		var tag string
		if err := json.Unmarshal(req.Params[0], &tag); err != nil {
			t.Fatalf("decode tag param: %v", err)
		}
		if tag != "safe" {
			t.Fatalf("expected tag safe, got %s", tag)
		}

		var fullTx bool
		if err := json.Unmarshal(req.Params[1], &fullTx); err != nil {
			t.Fatalf("decode fullTx param: %v", err)
		}
		if fullTx {
			t.Fatalf("expected fullTx false")
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"number": "0x520",
			},
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)
	blockRPC := newTestBlockRPC(t, server.URL)

	number, err := blockRPC.GetBlockNumberByTag(context.Background(), "safe")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if number != 1312 {
		t.Fatalf("expected number 1312, got %d", number)
	}
}

func TestBlockRPC_GetBlockNumberByTag_InvalidTag(t *testing.T) {
	var called int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
	}))
	t.Cleanup(server.Close)
	blockRPC := newTestBlockRPC(t, server.URL)
	number, err := blockRPC.GetBlockNumberByTag(context.Background(), "finalize")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid block tag") {
		t.Fatalf("expected invalid block tag error, got %v", err)
	}

	if number != 0 {
		t.Fatalf("expected number 0, got %d", number)
	}

	if atomic.LoadInt32(&called) != 0 {
		t.Fatalf("expected RPC server not to be called")
	}
}

func TestBlockRPC_GetBlockNumberByTag_ReturnsErrBlockNotFoundWhenResultIsNull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_getBlockByNumber")

		if len(req.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(req.Params))
		}

		var tag string
		if err := json.Unmarshal(req.Params[0], &tag); err != nil {
			t.Fatalf("decode tag param: %v", err)
		}
		if tag != "safe" {
			t.Fatalf("expected tag safe, got %s", tag)
		}

		var fullTx bool
		if err := json.Unmarshal(req.Params[1], &fullTx); err != nil {
			t.Fatalf("decode fullTx param: %v", err)
		}
		if fullTx {
			t.Fatalf("expected fullTx false")
		}

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  nil,
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)

	number, err := blockRPC.GetBlockNumberByTag(context.Background(), "safe")
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}

	if number != 0 {
		t.Fatalf("expected number 0, got %d", number)
	}
}

func TestBlockRPC_GetBlockNumberByTag_ReturnsErrorWhenRPCReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONRPCRequest(t, r, "eth_getBlockByNumber")

		if len(req.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(req.Params))
		}

		var tag string
		if err := json.Unmarshal(req.Params[0], &tag); err != nil {
			t.Fatalf("decode tag param: %v", err)
		}
		if tag != "safe" {
			t.Fatalf("expected tag safe, got %s", tag)
		}

		var fullTx bool
		if err := json.Unmarshal(req.Params[1], &fullTx); err != nil {
			t.Fatalf("decode fullTx param: %v", err)
		}
		if fullTx {
			t.Fatalf("expected fullTx false")
		}

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32602,
				"message": "invalid argument",
			},
		}
		writeJSONResponse(t, w, resp)
	}))
	t.Cleanup(server.Close)

	blockRPC := newTestBlockRPC(t, server.URL)

	number, err := blockRPC.GetBlockNumberByTag(context.Background(), "safe")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if number != 0 {
		t.Fatalf("expected number 0, got %d", number)
	}
	if !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("expected invalid argument error, got %v", err)
	}
}

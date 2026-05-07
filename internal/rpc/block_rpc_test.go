package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	return NewBlockRPC(ethClient, rpcClient, 5)
}

func TestBlockRPC_GetLatestBlockNumber_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method != "eth_blockNumber" {
			t.Fatalf("expected method eth_blockNumber, got %s", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": 1,
			"result": "0x10"
		}`))
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
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method != "eth_blockNumber" {
			t.Fatalf("expected method eth_blockNumber, got %s", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"error": {
		"code": -32000,
		"message": "execution error"
		}
	}`))
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
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method != "eth_blockNumber" {
			t.Fatalf("expected method eth_blockNumber, got %s", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"result": "not-a-hex"
		}`))
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

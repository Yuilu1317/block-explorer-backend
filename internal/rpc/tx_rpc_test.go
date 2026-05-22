package rpc

import (
	"block-explorer-backend/internal/types"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type jsonTxRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      any               `json:"id"`
}

func newTestTxRPC(t *testing.T, url string) *TxRPC {
	t.Helper()

	ethClient, rpcClient, err := NewEthClient(url)
	if err != nil {
		t.Fatalf("new eth client: %v", err)
	}
	t.Cleanup(rpcClient.Close)

	return NewTxRPC(ethClient, rpcClient, 5)
}

func decodeJSONTxRPCRequest(t *testing.T, r *http.Request, expectedMethod string) jsonTxRPCRequest {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", r.Method)
	}

	var req jsonTxRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode request: %v", err)
	}

	if req.Method != expectedMethod {
		t.Fatalf("expected method %s, got %s", expectedMethod, req.Method)
	}

	return req
}

func writeJSONTxResponse(t *testing.T, w http.ResponseWriter, resp any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func testReceiptResult(txHash common.Hash, status string, gasUsed string) map[string]any {
	return map[string]any{
		"type":              "0x2",
		"transactionHash":   txHash.Hex(),
		"blockHash":         "0x2222222222222222222222222222222222222222222222222222222222222222",
		"blockNumber":       "0x1",
		"transactionIndex":  "0x0",
		"status":            status,
		"gasUsed":           gasUsed,
		"cumulativeGasUsed": gasUsed,
		"effectiveGasPrice": "0x3b9aca00",
		"contractAddress":   nil,
		"logsBloom":         "0x" + strings.Repeat("00", 256),
		"logs":              []any{},
	}
}

func decodeTxHashParam(t *testing.T, req jsonTxRPCRequest) string {
	t.Helper()

	if len(req.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(req.Params))
	}

	var gotHash string
	if err := json.Unmarshal(req.Params[0], &gotHash); err != nil {
		t.Fatalf("unmarshal tx hash param: %v", err)
	}

	return gotHash
}

func TestTxRPC_GetTransactionReceipt_Success(t *testing.T) {
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	txHash := common.HexToHash(hash)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONTxRPCRequest(t, r, "eth_getTransactionReceipt")

		gotHash := decodeTxHashParam(t, req)
		if gotHash != txHash.Hex() {
			t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
		}

		writeJSONTxResponse(t, w, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  testReceiptResult(txHash, "0x1", "0x5208"),
		})
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	receipt, err := txRPC.GetTransactionReceipt(context.Background(), hash)
	if err != nil {
		t.Fatalf("get tx receipt: %v", err)
	}
	if receipt == nil {
		t.Fatalf("receipt is nil")
	}
	if receipt.TxHash != txHash {
		t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), receipt.TxHash)
	}
	if receipt.Status != uint64(1) {
		t.Fatalf("expected receipt status %d, got %d", uint64(1), receipt.Status)
	}
	if receipt.GasUsed != uint64(21000) {
		t.Fatalf("expected receipt gas used %d, got %d", uint64(21000), receipt.GasUsed)
	}
}

func TestTxRPC_GetTransactionReceipt_StatusZero(t *testing.T) {
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	txHash := common.HexToHash(hash)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONTxRPCRequest(t, r, "eth_getTransactionReceipt")

		gotHash := decodeTxHashParam(t, req)
		if gotHash != txHash.Hex() {
			t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
		}

		writeJSONTxResponse(t, w, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  testReceiptResult(txHash, "0x0", "0x5208"),
		})
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	receipt, err := txRPC.GetTransactionReceipt(context.Background(), hash)
	if err != nil {
		t.Fatalf("get tx receipt: %v", err)
	}
	if receipt == nil {
		t.Fatalf("receipt is nil")
	}
	if receipt.TxHash != txHash {
		t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), receipt.TxHash)
	}
	if receipt.Status != uint64(0) {
		t.Fatalf("expected receipt status %d, got %d", uint64(0), receipt.Status)
	}
	if receipt.GasUsed != uint64(21000) {
		t.Fatalf("expected receipt gas used %d, got %d", uint64(21000), receipt.GasUsed)
	}
}

func TestTxRPC_GetTransactionReceipt_NotFound(t *testing.T) {
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	txHash := common.HexToHash(hash)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONTxRPCRequest(t, r, "eth_getTransactionReceipt")

		gotHash := decodeTxHashParam(t, req)
		if gotHash != txHash.Hex() {
			t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
		}

		writeJSONTxResponse(t, w, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  nil,
		})
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	receipt, err := txRPC.GetTransactionReceipt(context.Background(), hash)
	if receipt != nil {
		t.Fatalf("expected nil receipt, got %+v", receipt)
	}
	if !errors.Is(err, types.ErrTxReceiptNotFound) {
		t.Fatalf("expected ErrTxReceiptNotFound, got %v", err)
	}
}

func TestTxRPC_GetTransactionReceipt_MapsCanceledError(t *testing.T) {
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when context is canceled")
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	receipt, err := txRPC.GetTransactionReceipt(ctx, hash)

	if receipt != nil {
		t.Fatalf("expected nil receipt, got %+v", receipt)
	}
	if !errors.Is(err, types.ErrRequestCanceled) {
		t.Fatalf("expected ErrRequestCanceled, got %v", err)
	}
}

func TestTxRPC_GetTransactionReceipt_ReturnsWrappedErrorForInvalidJSON(t *testing.T) {
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	txHash := common.HexToHash(hash)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONTxRPCRequest(t, r, "eth_getTransactionReceipt")

		gotHash := decodeTxHashParam(t, req)
		if gotHash != txHash.Hex() {
			t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid-json`))
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	receipt, err := txRPC.GetTransactionReceipt(context.Background(), hash)

	if receipt != nil {
		t.Fatalf("expected nil receipt, got %+v", receipt)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rpc: get transaction receipt") {
		t.Fatalf("expected wrapped error to contain rpc context, got %v", err)
	}
	if !strings.Contains(err.Error(), hash) {
		t.Fatalf("expected wrapped error to contain tx hash %s, got %v", hash, err)
	}
}

func newTestSignedTx(t *testing.T, chainID *big.Int) (*gethtypes.Transaction, common.Address) {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	to := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     1,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1_000_000_000),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(12345),
		Data:      nil,
	})

	signer := gethtypes.LatestSignerForChainID(chainID)

	signedTx, err := gethtypes.SignTx(tx, signer, privateKey)
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}

	from, err := gethtypes.Sender(signer, signedTx)
	if err != nil {
		t.Fatalf("derive sender from signed tx: %v", err)
	}

	return signedTx, from
}

func testTransactionResult(t *testing.T, tx *gethtypes.Transaction, pending bool) map[string]any {
	t.Helper()

	raw, err := tx.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal tx json: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal tx json: %v", err)
	}

	if pending {
		result["blockHash"] = nil
		result["blockNumber"] = nil
		result["transactionIndex"] = nil
	} else {
		result["blockHash"] = "0x2222222222222222222222222222222222222222222222222222222222222222"
		result["blockNumber"] = "0x1"
		result["transactionIndex"] = "0x0"
	}

	return result
}

func decodeAnyJSONTxRPCRequest(t *testing.T, r *http.Request) jsonTxRPCRequest {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", r.Method)
	}

	var req jsonTxRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode request: %v", err)
	}

	return req
}

func TestTxRPC_GetTransactionByHash_NotFound(t *testing.T) {
	hash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	txHash := common.HexToHash(hash)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeJSONTxRPCRequest(t, r, "eth_getTransactionByHash")

		gotHash := decodeTxHashParam(t, req)
		if gotHash != txHash.Hex() {
			t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
		}

		writeJSONTxResponse(t, w, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  nil,
		})
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	raw, err := txRPC.GetTransactionByHash(context.Background(), hash)

	if raw != nil {
		t.Fatalf("expected nil tx raw, got %+v", raw)
	}
	if !errors.Is(err, types.ErrTxNotFound) {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}
}

func TestTxRPC_GetTransactionByHash_PendingTransactionDoesNotFetchReceipt(t *testing.T) {
	chainID := big.NewInt(1)
	signedTx, expectedFrom := newTestSignedTx(t, chainID)

	hash := signedTx.Hash().Hex()
	txHash := signedTx.Hash()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeAnyJSONTxRPCRequest(t, r)

		switch req.Method {
		case "eth_getTransactionByHash":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  testTransactionResult(t, signedTx, true),
			})

		case "eth_chainId":
			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  "0x1",
			})

		case "eth_getTransactionReceipt":
			t.Fatal("should not fetch receipt for pending transaction")

		default:
			t.Fatalf("unexpected rpc method %s", req.Method)
		}
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	raw, err := txRPC.GetTransactionByHash(context.Background(), hash)

	if err != nil {
		t.Fatalf("get transaction by hash: %v", err)
	}
	if raw == nil {
		t.Fatal("expected tx raw, got nil")
	}
	if raw.Tx == nil {
		t.Fatal("expected tx, got nil")
	}
	if raw.Tx.Hash() != txHash {
		t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), raw.Tx.Hash().Hex())
	}
	if raw.From != expectedFrom.Hex() {
		t.Fatalf("expected from %s, got %s", expectedFrom.Hex(), raw.From)
	}
	if !raw.IsPending {
		t.Fatal("expected pending transaction")
	}
	if raw.Receipt != nil {
		t.Fatalf("expected nil receipt for pending tx, got %+v", raw.Receipt)
	}
}

func TestTxRPC_GetTransactionByHash_ConfirmedTransactionWithReceipt(t *testing.T) {
	chainID := big.NewInt(1)
	signedTx, expectedFrom := newTestSignedTx(t, chainID)

	hash := signedTx.Hash().Hex()
	txHash := signedTx.Hash()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeAnyJSONTxRPCRequest(t, r)

		switch req.Method {
		case "eth_getTransactionByHash":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  testTransactionResult(t, signedTx, false),
			})

		case "eth_chainId":
			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  "0x1",
			})

		case "eth_getTransactionReceipt":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected receipt tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  testReceiptResult(txHash, "0x1", "0x5208"),
			})

		default:
			t.Fatalf("unexpected rpc method %s", req.Method)
		}
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	raw, err := txRPC.GetTransactionByHash(context.Background(), hash)

	if err != nil {
		t.Fatalf("get transaction by hash: %v", err)
	}
	if raw == nil {
		t.Fatal("expected tx raw, got nil")
	}
	if raw.Tx == nil {
		t.Fatal("expected tx, got nil")
	}
	if raw.Tx.Hash() != txHash {
		t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), raw.Tx.Hash().Hex())
	}
	if raw.From != expectedFrom.Hex() {
		t.Fatalf("expected from %s, got %s", expectedFrom.Hex(), raw.From)
	}
	if raw.IsPending {
		t.Fatal("expected confirmed transaction")
	}
	if raw.Receipt == nil {
		t.Fatal("expected receipt, got nil")
	}
	if raw.Receipt.TxHash != txHash {
		t.Fatalf("expected receipt tx hash %s, got %s", txHash.Hex(), raw.Receipt.TxHash.Hex())
	}
	if raw.Receipt.Status != uint64(1) {
		t.Fatalf("expected receipt status 1, got %d", raw.Receipt.Status)
	}
	if raw.Receipt.GasUsed != uint64(21000) {
		t.Fatalf("expected receipt gas used 21000, got %d", raw.Receipt.GasUsed)
	}
}

func TestTxRPC_GetTransactionByHash_ConfirmedTransactionReceiptNotFound(t *testing.T) {
	chainID := big.NewInt(1)
	signedTx, expectedFrom := newTestSignedTx(t, chainID)

	hash := signedTx.Hash().Hex()
	txHash := signedTx.Hash()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeAnyJSONTxRPCRequest(t, r)

		switch req.Method {
		case "eth_getTransactionByHash":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  testTransactionResult(t, signedTx, false),
			})

		case "eth_chainId":
			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  "0x1",
			})

		case "eth_getTransactionReceipt":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected receipt tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  nil,
			})

		default:
			t.Fatalf("unexpected rpc method %s", req.Method)
		}
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	raw, err := txRPC.GetTransactionByHash(context.Background(), hash)

	if err != nil {
		t.Fatalf("get transaction by hash: %v", err)
	}
	if raw == nil {
		t.Fatal("expected tx raw, got nil")
	}
	if raw.Tx == nil {
		t.Fatal("expected tx, got nil")
	}
	if raw.From != expectedFrom.Hex() {
		t.Fatalf("expected from %s, got %s", expectedFrom.Hex(), raw.From)
	}
	if raw.IsPending {
		t.Fatal("expected confirmed transaction")
	}
	if raw.Receipt != nil {
		t.Fatalf("expected nil receipt, got %+v", raw.Receipt)
	}
}

func TestTxRPC_GetTransactionByHash_ReturnsErrorWhenChainIDFails(t *testing.T) {
	chainID := big.NewInt(1)
	signedTx, _ := newTestSignedTx(t, chainID)

	hash := signedTx.Hash().Hex()
	txHash := signedTx.Hash()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeAnyJSONTxRPCRequest(t, r)

		switch req.Method {
		case "eth_getTransactionByHash":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  testTransactionResult(t, signedTx, false),
			})

		case "eth_chainId":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{invalid-json`))

		default:
			t.Fatalf("unexpected rpc method %s", req.Method)
		}
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	raw, err := txRPC.GetTransactionByHash(context.Background(), hash)

	if raw != nil {
		t.Fatalf("expected nil raw, got %+v", raw)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rpc: get chain id") {
		t.Fatalf("expected error to contain chain id context, got %v", err)
	}
}

func TestTxRPC_GetTransactionByHash_ReturnsErrorWhenRecoverSenderFails(t *testing.T) {
	signedTxChainID := big.NewInt(2)
	rpcChainID := "0x1"

	signedTx, _ := newTestSignedTx(t, signedTxChainID)

	hash := signedTx.Hash().Hex()
	txHash := signedTx.Hash()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeAnyJSONTxRPCRequest(t, r)

		switch req.Method {
		case "eth_getTransactionByHash":
			gotHash := decodeTxHashParam(t, req)
			if gotHash != txHash.Hex() {
				t.Fatalf("expected tx hash %s, got %s", txHash.Hex(), gotHash)
			}

			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  testTransactionResult(t, signedTx, false),
			})

		case "eth_chainId":
			writeJSONTxResponse(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  rpcChainID,
			})

		default:
			t.Fatalf("unexpected rpc method %s", req.Method)
		}
	}))
	defer server.Close()

	txRPC := newTestTxRPC(t, server.URL)

	raw, err := txRPC.GetTransactionByHash(context.Background(), hash)

	if raw != nil {
		t.Fatalf("expected nil raw, got %+v", raw)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rpc: derive sender from tx") {
		t.Fatalf("expected error to contain derive sender context, got %v", err)
	}
}

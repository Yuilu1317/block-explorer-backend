package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"block-explorer-backend/internal/types"

	"github.com/gin-gonic/gin"
)

func TestWalletController_ListCompletedBlocks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakeWalletService{
		completedBlocksResp: &types.WalletCompletedBlocksResponse{
			ChainID: 11155111,
			Blocks: []types.WalletCompletedBlock{
				{
					Number:     100,
					Hash:       "0xblock100",
					ParentHash: "0xblock99",
					Transactions: []types.WalletCompletedTransaction{
						{
							TxHash:        "0xtx100",
							FromAddress:   "0xfrom",
							ToAddress:     "0xto",
							AmountWei:     "100",
							ReceiptStatus: 1,
						},
					},
				},
			},
		},
	}

	router := newWalletControllerTestRouter(NewWalletController(service))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/internal/wallet/completed-blocks?chain_id=11155111&from_block=100&limit=10",
		nil,
	)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !service.listCompletedBlocksCalled {
		t.Fatal("expected service to be called")
	}

	if service.gotListChainID != 11155111 {
		t.Fatalf("expected chain id 11155111, got %d", service.gotListChainID)
	}

	if service.gotFromBlock != 100 {
		t.Fatalf("expected from block 100, got %d", service.gotFromBlock)
	}

	if service.gotLimit != 10 {
		t.Fatalf("expected limit 10, got %d", service.gotLimit)
	}

	var resp types.WalletCompletedBlocksResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.ChainID != 11155111 {
		t.Fatalf("expected response chain id 11155111, got %d", resp.ChainID)
	}

	if len(resp.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(resp.Blocks))
	}

	if resp.Blocks[0].Number != 100 {
		t.Fatalf("expected block number 100, got %d", resp.Blocks[0].Number)
	}

	if len(resp.Blocks[0].Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Blocks[0].Transactions))
	}

	if resp.Blocks[0].Transactions[0].TxHash != "0xtx100" {
		t.Fatalf("expected tx hash 0xtx100, got %q", resp.Blocks[0].Transactions[0].TxHash)
	}
}

func TestWalletController_ListCompletedBlocks_BadRequest(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "missing chain id",
			path:    "/internal/wallet/completed-blocks?from_block=100&limit=10",
			wantErr: "chain_id is required",
		},
		{
			name:    "invalid chain id",
			path:    "/internal/wallet/completed-blocks?chain_id=abc&from_block=100&limit=10",
			wantErr: "chain_id must be a valid int64",
		},
		{
			name:    "zero chain id",
			path:    "/internal/wallet/completed-blocks?chain_id=0&from_block=100&limit=10",
			wantErr: "chain_id must be positive",
		},
		{
			name:    "missing from block",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&limit=10",
			wantErr: "from_block is required",
		},
		{
			name:    "invalid from block",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&from_block=abc&limit=10",
			wantErr: "from_block must be a valid int64",
		},
		{
			name:    "negative from block",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&from_block=-1&limit=10",
			wantErr: "from_block must be non-negative",
		},
		{
			name:    "missing limit",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&from_block=100",
			wantErr: "limit is required",
		},
		{
			name:    "invalid limit",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&from_block=100&limit=abc",
			wantErr: "limit must be a valid integer",
		},
		{
			name:    "zero limit",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&from_block=100&limit=0",
			wantErr: "limit must be positive",
		},
		{
			name:    "limit too large",
			path:    "/internal/wallet/completed-blocks?chain_id=11155111&from_block=100&limit=101",
			wantErr: "limit must be less than or equal to 100",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			service := &fakeWalletService{
				completedBlocksResp: &types.WalletCompletedBlocksResponse{
					ChainID: 11155111,
					Blocks:  []types.WalletCompletedBlock{},
				},
			}

			router := newWalletControllerTestRouter(NewWalletController(service))

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
			}

			if service.listCompletedBlocksCalled {
				t.Fatal("expected service not to be called")
			}

			if !strings.Contains(w.Body.String(), tt.wantErr) {
				t.Fatalf("expected body to contain %q, got %q", tt.wantErr, w.Body.String())
			}
		})
	}
}

func TestWalletController_ListCompletedBlocks_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	serviceErr := errors.New("service failed")
	service := &fakeWalletService{
		completedBlocksErr: serviceErr,
	}

	router := newWalletControllerTestRouter(NewWalletController(service))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/internal/wallet/completed-blocks?chain_id=11155111&from_block=100&limit=10",
		nil,
	)

	router.ServeHTTP(w, req)

	if !service.listCompletedBlocksCalled {
		t.Fatal("expected service to be called")
	}

	if w.Code < 400 {
		t.Fatalf("expected error status, got %d, body=%s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Fatalf("expected body to contain service error, got %q", w.Body.String())
	}
}

func newWalletControllerTestRouter(controller *WalletController) *gin.Engine {
	router := gin.New()

	internalGroup := router.Group("/internal")
	{
		walletGroup := internalGroup.Group("/wallet")
		{
			walletGroup.GET("/completed-blocks", controller.ListCompletedBlocks)
		}
	}

	return router
}

type fakeWalletService struct {
	completedBlocksResp *types.WalletCompletedBlocksResponse
	completedBlocksErr  error

	syncStatusResp *types.GetSyncStatusResponse
	syncStatusErr  error

	listCompletedBlocksCalled bool
	gotListChainID            int64
	gotFromBlock              int64
	gotLimit                  int

	getSyncStatusCalled  bool
	gotSyncStatusChainID int64
}

func (f *fakeWalletService) ListCompletedBlocks(
	ctx context.Context,
	chainID int64,
	fromBlock int64,
	limit int,
) (*types.WalletCompletedBlocksResponse, error) {
	f.listCompletedBlocksCalled = true
	f.gotListChainID = chainID
	f.gotFromBlock = fromBlock
	f.gotLimit = limit

	if f.completedBlocksErr != nil {
		return nil, f.completedBlocksErr
	}

	return f.completedBlocksResp, nil
}

func (f *fakeWalletService) GetSyncStatus(
	ctx context.Context,
	chainID int64,
) (*types.GetSyncStatusResponse, error) {
	f.getSyncStatusCalled = true
	f.gotSyncStatusChainID = chainID

	if f.syncStatusErr != nil {
		return nil, f.syncStatusErr
	}

	return f.syncStatusResp, nil
}

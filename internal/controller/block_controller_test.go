package controller

import (
	"block-explorer-backend/internal/service/model"
	"block-explorer-backend/internal/types"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeBlockService struct {
	block model.BlockQueryResult
	err   error

	blockRange *types.BlockRangeSyncResult

	syncRangeCalled bool
	gotStart        uint64
	gotEnd          uint64

	getBlockCalled bool
	gotBlockNumber uint64

	syncBlockCalled bool
	gotSyncNumber   uint64
}

func (s *fakeBlockService) GetBlockByNumber(ctx context.Context, number uint64) (model.BlockQueryResult, error) {
	s.getBlockCalled = true
	s.gotBlockNumber = number
	return s.block, s.err
}

func (s *fakeBlockService) SyncBlockToDB(ctx context.Context, number uint64) error {
	s.syncBlockCalled = true
	s.gotSyncNumber = number
	return s.err
}

func (s *fakeBlockService) SyncBlockRangeToDB(ctx context.Context, start, end uint64) (*types.BlockRangeSyncResult, error) {
	s.syncRangeCalled = true
	s.gotStart = start
	s.gotEnd = end
	return s.blockRange, s.err
}

func setupTestController(t *testing.T) (*BlockController, *fakeBlockService) {
	t.Helper()

	service := &fakeBlockService{}

	controller := NewBlockController(service, service)

	return controller, service
}

func setupSyncBlockRangeRouter(t *testing.T) (*gin.Engine, *fakeBlockService) {
	t.Helper()

	controller, service := setupTestController(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/blocks/sync", controller.SyncBlockRange)

	return router, service
}

func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()

	if w.Code != wantStatus {
		t.Fatalf("expected status %d, got %d, body: %s", wantStatus, w.Code, w.Body.String())
	}

	var resp types.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if resp.Code != wantStatus {
		t.Fatalf("expected response code %d, got %d", wantStatus, resp.Code)
	}

	if resp.Message != wantMessage {
		t.Fatalf("expected message %q, got %q", wantMessage, resp.Message)
	}
}

func TestBlockController_SyncBlockRange_BadRequest(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantMessage string
	}{
		{
			name:        "missing start",
			url:         "/blocks/sync?end=10",
			wantMessage: "start and end are required",
		},
		{
			name:        "missing end",
			url:         "/blocks/sync?start=1",
			wantMessage: "start and end are required",
		},
		{
			name:        "invalid start",
			url:         "/blocks/sync?start=abc&end=10",
			wantMessage: "invalid start",
		},
		{
			name:        "invalid end",
			url:         "/blocks/sync?start=1&end=abc",
			wantMessage: "invalid end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, service := setupSyncBlockRangeRouter(t)

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assertErrorResponse(t, w, http.StatusBadRequest, tt.wantMessage)

			if service.syncRangeCalled {
				t.Fatalf("service should not be called for bad request")
			}
		})
	}
}

func TestBlockController_SyncBlockRange_Success(t *testing.T) {
	router, service := setupSyncBlockRangeRouter(t)

	service.blockRange = &types.BlockRangeSyncResult{
		Start:     10,
		End:       20,
		Requested: 11,
		Succeeded: 11,
		Failed:    0,
	}

	req := httptest.NewRequest(http.MethodPost, "/blocks/sync?start=10&end=20", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !service.syncRangeCalled {
		t.Fatalf("service should be called")
	}
	if service.gotStart != 10 {
		t.Fatalf("expected start %d, got %d", 10, service.gotStart)
	}

	if service.gotEnd != 20 {
		t.Fatalf("expected end %d, got %d", 20, service.gotEnd)
	}

	var resp types.SuccessResponse[*types.BlockRangeSyncResult]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected response code %d, got %d", 0, resp.Code)
	}

	if resp.Message != "success" {
		t.Fatalf("expected message %q, got %q", "success", resp.Message)
	}

	if resp.Data == nil {
		t.Fatalf("expected response data, got nil")
	}

	if resp.Data.Start != 10 {
		t.Fatalf("expected data start %d, got %d", 10, resp.Data.Start)
	}

	if resp.Data.End != 20 {
		t.Fatalf("expected data end %d, got %d", 20, resp.Data.End)
	}

	if resp.Data.Requested != 11 {
		t.Fatalf("expected requested %d, got %d", 11, resp.Data.Requested)
	}
}

func TestBlockController_SyncBlockRange_ServiceError(t *testing.T) {
	router, service := setupSyncBlockRangeRouter(t)

	service.err = types.ErrInvalidBlockRange

	req := httptest.NewRequest(http.MethodPost, "/blocks/sync?start=10&end=20", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !service.syncRangeCalled {
		t.Fatalf("service should be called")
	}
	if service.gotStart != 10 {
		t.Fatalf("expected start %d, got %d", 10, service.gotStart)
	}
	if service.gotEnd != 20 {
		t.Fatalf("expected end %d, got %d", 20, service.gotEnd)
	}

	var resp types.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	assertErrorResponse(t, w, http.StatusBadRequest, "invalid block range")
}

func setupSyncBlockRouter(t *testing.T) (*gin.Engine, *fakeBlockService) {
	t.Helper()

	controller, service := setupTestController(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/block/sync/:number", controller.SyncBlock)

	return router, service
}

func TestBlockController_SyncBlock_InvalidNumber(t *testing.T) {
	router, service := setupSyncBlockRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/block/sync/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusBadRequest, "invalid block number")
	if service.syncBlockCalled {
		t.Fatalf("service should not be called when number is invalid")
	}
}

func TestBlockController_SyncBlock_Success(t *testing.T) {
	router, service := setupSyncBlockRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/block/sync/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !service.syncBlockCalled {
		t.Fatalf("service should be called")
	}
	if service.gotSyncNumber != 123 {
		t.Fatalf("expected number %d, got %d", 123, service.gotSyncNumber)
	}

	var resp types.SuccessResponse[uint64]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected response code %d, got %d", 0, resp.Code)
	}

	if resp.Message != "success" {
		t.Fatalf("expected message %q, got %q", "success", resp.Message)
	}

	if resp.Data != 123 {
		t.Fatalf("expected data %d, got %d", 123, resp.Data)
	}
}

func TestBlockController_SyncBlock_ServiceError(t *testing.T) {
	router, service := setupSyncBlockRouter(t)

	service.err = types.ErrRPCTimeout

	req := httptest.NewRequest(http.MethodPost, "/block/sync/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !service.syncBlockCalled {
		t.Fatalf("service should be called")
	}
	if service.gotSyncNumber != 123 {
		t.Fatalf("expected number %d, got %d", 123, service.gotSyncNumber)
	}

	var resp types.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	assertErrorResponse(t, w, http.StatusGatewayTimeout, "upstream timeout")
}

func setupGetBlockRouter(t *testing.T) (*gin.Engine, *fakeBlockService) {
	t.Helper()

	controller, service := setupTestController(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/block/:number", controller.GetBlock)

	return router, service
}

func TestBlockController_GetBlock_InvalidNumber(t *testing.T) {
	router, service := setupGetBlockRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/block/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusBadRequest, "invalid block number")

	if service.getBlockCalled {
		t.Fatalf("service should not be called when number is invalid")
	}
}

func TestBlockController_GetBlock_Success(t *testing.T) {
	router, service := setupGetBlockRouter(t)

	service.block = model.BlockQueryResult{
		Block: model.BlockDetail{
			Number: 123,
		},
		Source: model.DataSourceRPC,
	}

	req := httptest.NewRequest(http.MethodGet, "/block/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !service.getBlockCalled {
		t.Fatalf("service should be called")
	}

	if service.gotBlockNumber != 123 {
		t.Fatalf("expected number %d, got %d", 123, service.gotBlockNumber)
	}
	var resp types.SuccessResponse[types.BlockDetailDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected response code %d, got %d", 0, resp.Code)
	}

	if resp.Message != "success" {
		t.Fatalf("expected message %q, got %q", "success", resp.Message)
	}

	if resp.Data.Number != 123 {
		t.Fatalf("expected data %d, got %d", 123, resp.Data.Number)
	}
}

func TestBlockController_GetBlock_ServiceError(t *testing.T) {
	router, service := setupGetBlockRouter(t)
	service.err = types.ErrRPCTimeout
	req := httptest.NewRequest(http.MethodGet, "/block/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !service.getBlockCalled {
		t.Fatalf("service should be called")
	}
	if service.gotBlockNumber != 123 {
		t.Fatalf("expected number %d, got %d", 123, service.gotBlockNumber)
	}
	var resp types.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	assertErrorResponse(t, w, http.StatusGatewayTimeout, "upstream timeout")
}

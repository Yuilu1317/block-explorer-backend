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

type fakeAddressService struct {
	addressInfo   *types.AddressInfo
	addressTxList *types.AddressTransactionListDTO

	infoErr   error
	txListErr error

	calledInfo   bool
	calledTxList bool
	gotAddress   string
	gotPage      int
	gotPageSize  int
}

func (f *fakeAddressService) GetAddress(ctx context.Context, address string) (*types.AddressInfo, error) {
	f.calledInfo = true
	f.gotAddress = address
	if f.infoErr != nil {
		return nil, f.infoErr
	}
	return f.addressInfo, nil
}

func (f *fakeAddressService) GetIndexedTransactionsByAddress(
	ctx context.Context,
	address string,
	page int,
	pageSize int,
) (*types.AddressTransactionListDTO, error) {
	f.calledTxList = true
	f.gotAddress = address
	f.gotPage = page
	f.gotPageSize = pageSize
	if f.txListErr != nil {
		return nil, f.txListErr
	}
	return f.addressTxList, nil
}

func setupTestAddressController(t *testing.T) (*AddressController, *fakeAddressService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := &fakeAddressService{}
	ctrl := NewAddressController(svc)
	return ctrl, svc
}

func setupAddressRouter(ctrl *AddressController) *gin.Engine {
	r := gin.New()

	r.GET("/address/:address", ctrl.GetAddress)
	r.GET("/indexed/address/:address/transactions", ctrl.GetIndexedTransactionsByAddress)

	return r
}

func TestAddressController_GetAddress_MissingAddress(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/address/", nil)

	ctrl.GetAddress(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}

	if svc.calledInfo {
		t.Fatalf("expected GetAddress not called")
	}
}

func TestAddressController_GetAddress_ServiceError(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)
	r := setupAddressRouter(ctrl)

	svc.infoErr = errors.New("service failed")

	req := httptest.NewRequest(
		http.MethodGet,
		"/address/0x1111111111111111111111111111111111111111",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledInfo {
		t.Fatalf("expected GetAddress called")
	}

	if svc.gotAddress != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("expected address passed to service, got %s", svc.gotAddress)
	}
}

func TestAddressController_GetAddress_Success(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)
	r := setupAddressRouter(ctrl)

	address := "0x1111111111111111111111111111111111111111"

	svc.addressInfo = &types.AddressInfo{
		Address:    address,
		Balance:    "1000000000000000000",
		Nonce:      7,
		IsContract: false,
	}

	req := httptest.NewRequest(http.MethodGet, "/address/"+address, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledInfo {
		t.Fatalf("expected GetAddress called")
	}

	if svc.gotAddress != address {
		t.Fatalf("expected address=%s, got %s", address, svc.gotAddress)
	}

	if !strings.Contains(w.Body.String(), address) {
		t.Fatalf("expected response body to contain address %s, got %s", address, w.Body.String())
	}
}

func TestAddressController_GetIndexedTransactionsByAddress_MissingAddress(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/indexed/address//transactions",
		nil,
	)

	ctrl.GetIndexedTransactionsByAddress(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}

	if svc.calledTxList {
		t.Fatalf("expected GetIndexedTransactionsByAddress not called")
	}
}

func TestAddressController_GetIndexedTransactionsByAddress_UsesDefaultPagination(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)
	r := setupAddressRouter(ctrl)

	address := "0x1111111111111111111111111111111111111111"

	svc.addressTxList = &types.AddressTransactionListDTO{
		Items:    []*types.AddressTransactionDTO{},
		Page:     1,
		PageSize: 20,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/indexed/address/"+address+"/transactions",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledTxList {
		t.Fatalf("expected GetIndexedTransactionsByAddress called")
	}

	if svc.gotAddress != address {
		t.Fatalf("expected address=%s, got %s", address, svc.gotAddress)
	}

	if svc.gotPage != 1 {
		t.Fatalf("expected page=1, got %d", svc.gotPage)
	}

	if svc.gotPageSize != 20 {
		t.Fatalf("expected page_size=20, got %d", svc.gotPageSize)
	}
}

func TestAddressController_GetIndexedTransactionsByAddress_ParsesPagination(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)
	r := setupAddressRouter(ctrl)

	address := "0x1111111111111111111111111111111111111111"

	svc.addressTxList = &types.AddressTransactionListDTO{
		Items:    []*types.AddressTransactionDTO{},
		Page:     2,
		PageSize: 50,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/indexed/address/"+address+"/transactions?page=2&page_size=50",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledTxList {
		t.Fatalf("expected GetIndexedTransactionsByAddress called")
	}

	if svc.gotAddress != address {
		t.Fatalf("expected address=%s, got %s", address, svc.gotAddress)
	}

	if svc.gotPage != 2 {
		t.Fatalf("expected page=2, got %d", svc.gotPage)
	}

	if svc.gotPageSize != 50 {
		t.Fatalf("expected page_size=50, got %d", svc.gotPageSize)
	}
}

func TestAddressController_GetIndexedTransactionsByAddress_InvalidPaginationQuery(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "invalid page",
			path: "/indexed/address/0x1111111111111111111111111111111111111111/transactions?page=abc&page_size=20",
		},
		{
			name: "invalid page size",
			path: "/indexed/address/0x1111111111111111111111111111111111111111/transactions?page=1&page_size=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, svc := setupTestAddressController(t)
			r := setupAddressRouter(ctrl)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
			}

			if svc.calledTxList {
				t.Fatalf("expected GetIndexedTransactionsByAddress not called")
			}
		})
	}
}

func TestAddressController_GetIndexedTransactionsByAddress_ServiceError(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)
	r := setupAddressRouter(ctrl)

	address := "0x1111111111111111111111111111111111111111"
	svc.txListErr = errors.New("service failed")

	req := httptest.NewRequest(
		http.MethodGet,
		"/indexed/address/"+address+"/transactions?page=1&page_size=20",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledTxList {
		t.Fatalf("expected GetIndexedTransactionsByAddress called")
	}

	if svc.gotAddress != address {
		t.Fatalf("expected address=%s, got %s", address, svc.gotAddress)
	}

	if svc.gotPage != 1 {
		t.Fatalf("expected page=1, got %d", svc.gotPage)
	}

	if svc.gotPageSize != 20 {
		t.Fatalf("expected page_size=20, got %d", svc.gotPageSize)
	}
}

func TestAddressController_GetIndexedTransactionsByAddress_Success(t *testing.T) {
	ctrl, svc := setupTestAddressController(t)
	r := setupAddressRouter(ctrl)

	address := "0x1111111111111111111111111111111111111111"

	svc.addressTxList = &types.AddressTransactionListDTO{
		Items: []*types.AddressTransactionDTO{
			{
				Hash:                "0xtxhash1",
				BlockNumber:         100,
				BlockHash:           "0xblockhash",
				TxIndex:             0,
				FromAddress:         address,
				ToAddress:           "0x2222222222222222222222222222222222222222",
				Direction:           "out",
				CounterpartyAddress: "0x2222222222222222222222222222222222222222",
				Nonce:               1,
				ValueWei:            "1000000000000000000",
				GasLimit:            21000,
				GasPriceWei:         "1000000000",
				InputData:           "0x",
			},
		},
		Page:     1,
		PageSize: 20,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/indexed/address/"+address+"/transactions?page=1&page_size=20",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if !svc.calledTxList {
		t.Fatalf("expected GetIndexedTransactionsByAddress called")
	}

	if svc.gotAddress != address {
		t.Fatalf("expected address=%s, got %s", address, svc.gotAddress)
	}

	if svc.gotPage != 1 {
		t.Fatalf("expected page=1, got %d", svc.gotPage)
	}

	if svc.gotPageSize != 20 {
		t.Fatalf("expected page_size=20, got %d", svc.gotPageSize)
	}

	body := w.Body.String()

	if !strings.Contains(body, "0xtxhash1") {
		t.Fatalf("expected response body to contain tx hash, got %s", body)
	}

	if !strings.Contains(body, "out") {
		t.Fatalf("expected response body to contain direction, got %s", body)
	}

	if !strings.Contains(body, "page_size") {
		t.Fatalf("expected response body to contain page_size, got %s", body)
	}
}

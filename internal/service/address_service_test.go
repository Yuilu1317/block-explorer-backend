package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeAddressRPC struct {
	Balance string
	Nonce   uint64
	Code    string

	balanceErr error
	nonceErr   error
	codeErr    error

	balanceCalled bool
	nonceCalled   bool
	codeCalled    bool
	gotAddress    string
}

func (f *fakeAddressRPC) GetBalance(ctx context.Context, address string) (string, error) {
	f.balanceCalled = true
	f.gotAddress = address
	if f.balanceErr != nil {
		return "", f.balanceErr
	}
	return f.Balance, nil
}

func (f *fakeAddressRPC) GetNonce(ctx context.Context, address string) (uint64, error) {
	f.nonceCalled = true
	f.gotAddress = address
	if f.nonceErr != nil {
		return 0, f.nonceErr
	}
	return f.Nonce, nil
}

func (f *fakeAddressRPC) GetCode(ctx context.Context, address string) (string, error) {
	f.codeCalled = true
	f.gotAddress = address
	if f.codeErr != nil {
		return "", f.codeErr
	}
	return f.Code, nil
}

type fakeTxRepoToAddressService struct {
	listTransactions []models.Transaction
	err              error

	called     bool
	gotAddress string
	gotLimit   int
	gotOffset  int
}

func (f *fakeTxRepoToAddressService) ListTransactionsByAddress(
	ctx context.Context,
	address string,
	limit int,
	offset int,
) ([]models.Transaction, error) {
	f.called = true
	f.gotAddress = address
	f.gotLimit = limit
	f.gotOffset = offset

	if f.err != nil {
		return nil, f.err
	}
	return f.listTransactions, nil
}

func setupAddressServiceTest() (*AddressService, *fakeAddressRPC, *fakeTxRepoToAddressService) {
	fakeRPC := &fakeAddressRPC{}
	fakeTxRepo := &fakeTxRepoToAddressService{}

	s := NewAddressService(fakeRPC, fakeTxRepo)

	return s, fakeRPC, fakeTxRepo
}

func newAddressServiceTestTransaction(
	hash string,
	blockNumber uint64,
	txIndex uint,
	from string,
	to string,
) models.Transaction {
	return models.Transaction{
		Hash:        hash,
		BlockNumber: blockNumber,
		BlockHash:   "0xblockhash",
		TxIndex:     txIndex,

		FromAddress:      from,
		FromAddressLower: strings.ToLower(from),
		ToAddress:        to,
		ToAddressLower:   strings.ToLower(to),

		Nonce:       uint64(txIndex),
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0x",
	}
}

func TestAddressService_GetAddress_SuccessEOA(t *testing.T) {
	s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"

	fakeRPC.Balance = "1000000000000000000"
	fakeRPC.Nonce = 7
	fakeRPC.Code = "0x"

	got, err := s.GetAddress(ctx, "  "+address+"  ")
	if err != nil {
		t.Fatalf("get address: %v", err)
	}

	if got.Address != address {
		t.Fatalf("expected address=%s, got %s", address, got.Address)
	}

	if got.Balance != "1000000000000000000" {
		t.Fatalf("expected balance=1000000000000000000, got %s", got.Balance)
	}

	if got.Nonce != 7 {
		t.Fatalf("expected nonce=7, got %d", got.Nonce)
	}

	if got.IsContract {
		t.Fatalf("expected is_contract=false, got true")
	}

	if !fakeRPC.balanceCalled {
		t.Fatalf("expected GetBalance called")
	}

	if !fakeRPC.nonceCalled {
		t.Fatalf("expected GetNonce called")
	}

	if !fakeRPC.codeCalled {
		t.Fatalf("expected GetCode called")
	}

	if fakeRPC.gotAddress != address {
		t.Fatalf("expected rpc address=%s, got %s", address, fakeRPC.gotAddress)
	}

	if fakeTxRepo.called {
		t.Fatalf("expected tx repo not called")
	}
}

func TestAddressService_GetAddress_ReturnsContractAddress(t *testing.T) {
	s, fakeRPC, _ := setupAddressServiceTest()
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"

	fakeRPC.Balance = "0"
	fakeRPC.Nonce = 1
	fakeRPC.Code = "0x608060405234801561001057600080fd5b"

	got, err := s.GetAddress(ctx, address)
	if err != nil {
		t.Fatalf("get address: %v", err)
	}

	if !got.IsContract {
		t.Fatalf("expected is_contract=true, got false")
	}
}

func TestAddressService_GetAddress_InvalidAddress(t *testing.T) {
	s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
	ctx := context.Background()

	got, err := s.GetAddress(ctx, "bad-address")
	if !errors.Is(err, types.ErrInvalidAddress) {
		t.Fatalf("expected ErrInvalidAddress, got %v", err)
	}

	if got != nil {
		t.Fatalf("expected got=nil, got %+v", got)
	}

	if fakeRPC.balanceCalled || fakeRPC.nonceCalled || fakeRPC.codeCalled {
		t.Fatalf("expected rpc not called")
	}

	if fakeTxRepo.called {
		t.Fatalf("expected tx repo not called")
	}
}

func TestAddressService_GetAddress_ReturnsErrorWhenGetBalanceFails(t *testing.T) {
	s, fakeRPC, _ := setupAddressServiceTest()
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"
	errBalance := errors.New("balance rpc failed")

	fakeRPC.balanceErr = errBalance

	got, err := s.GetAddress(ctx, address)
	if !errors.Is(err, errBalance) {
		t.Fatalf("expected balance error, got %v", err)
	}

	if got != nil {
		t.Fatalf("expected got=nil, got %+v", got)
	}

	if !fakeRPC.balanceCalled {
		t.Fatalf("expected GetBalance called")
	}

	if fakeRPC.nonceCalled {
		t.Fatalf("expected GetNonce not called")
	}

	if fakeRPC.codeCalled {
		t.Fatalf("expected GetCode not called")
	}
}

func TestAddressService_GetAddress_ReturnsErrorWhenGetNonceFails(t *testing.T) {
	s, fakeRPC, _ := setupAddressServiceTest()
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"
	errNonce := errors.New("nonce rpc failed")

	fakeRPC.Balance = "100"
	fakeRPC.nonceErr = errNonce

	got, err := s.GetAddress(ctx, address)
	if !errors.Is(err, errNonce) {
		t.Fatalf("expected nonce error, got %v", err)
	}

	if got != nil {
		t.Fatalf("expected got=nil, got %+v", got)
	}

	if !fakeRPC.balanceCalled {
		t.Fatalf("expected GetBalance called")
	}

	if !fakeRPC.nonceCalled {
		t.Fatalf("expected GetNonce called")
	}

	if fakeRPC.codeCalled {
		t.Fatalf("expected GetCode not called")
	}
}

func TestAddressService_GetAddress_ReturnsErrorWhenGetCodeFails(t *testing.T) {
	s, fakeRPC, _ := setupAddressServiceTest()
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"
	errCode := errors.New("code rpc failed")

	fakeRPC.Balance = "100"
	fakeRPC.Nonce = 3
	fakeRPC.codeErr = errCode

	got, err := s.GetAddress(ctx, address)
	if !errors.Is(err, errCode) {
		t.Fatalf("expected code error, got %v", err)
	}

	if got != nil {
		t.Fatalf("expected got=nil, got %+v", got)
	}

	if !fakeRPC.balanceCalled {
		t.Fatalf("expected GetBalance called")
	}

	if !fakeRPC.nonceCalled {
		t.Fatalf("expected GetNonce called")
	}

	if !fakeRPC.codeCalled {
		t.Fatalf("expected GetCode called")
	}
}

func TestAddressService_GetIndexedTransactionsByAddress_Success(t *testing.T) {
	s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
	ctx := context.Background()

	queryAddress := "0x39fA8c5f2793459D6622857E7D9FbB4BD91766d3"
	queryAddressLower := strings.ToLower(queryAddress)

	otherAddress1 := "0x1111111111111111111111111111111111111111"
	otherAddress2 := "0x2222222222222222222222222222222222222222"

	fakeTxRepo.listTransactions = []models.Transaction{
		newAddressServiceTestTransaction(
			"0xtxhash1",
			101,
			0,
			queryAddress,
			otherAddress1,
		),
		newAddressServiceTestTransaction(
			"0xtxhash2",
			100,
			0,
			otherAddress2,
			queryAddress,
		),
	}

	got, err := s.GetIndexedTransactionsByAddress(ctx, "  "+queryAddress+"  ", 2, 20)
	if err != nil {
		t.Fatalf("get indexed transactions by address: %v", err)
	}

	if got == nil {
		t.Fatalf("expected result, got nil")
	}

	if got.Page != 2 {
		t.Fatalf("expected page=2, got %d", got.Page)
	}

	if got.PageSize != 20 {
		t.Fatalf("expected page_size=20, got %d", got.PageSize)
	}

	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}

	if !fakeTxRepo.called {
		t.Fatalf("expected tx repo called")
	}

	if fakeTxRepo.gotAddress != queryAddressLower {
		t.Fatalf("expected repo address=%s, got %s", queryAddressLower, fakeTxRepo.gotAddress)
	}

	if fakeTxRepo.gotLimit != 20 {
		t.Fatalf("expected limit=20, got %d", fakeTxRepo.gotLimit)
	}

	if fakeTxRepo.gotOffset != 20 {
		t.Fatalf("expected offset=20, got %d", fakeTxRepo.gotOffset)
	}

	if fakeRPC.balanceCalled || fakeRPC.nonceCalled || fakeRPC.codeCalled {
		t.Fatalf("expected address rpc not called")
	}

	first := got.Items[0]
	if first.Hash != "0xtxhash1" {
		t.Fatalf("expected first hash=0xtxhash1, got %s", first.Hash)
	}

	if first.Direction != "out" {
		t.Fatalf("expected first direction=out, got %s", first.Direction)
	}

	if first.CounterpartyAddress != otherAddress1 {
		t.Fatalf("expected first counterparty=%s, got %s", otherAddress1, first.CounterpartyAddress)
	}

	second := got.Items[1]
	if second.Hash != "0xtxhash2" {
		t.Fatalf("expected second hash=0xtxhash2, got %s", second.Hash)
	}

	if second.Direction != "in" {
		t.Fatalf("expected second direction=in, got %s", second.Direction)
	}

	if second.CounterpartyAddress != otherAddress2 {
		t.Fatalf("expected second counterparty=%s, got %s", otherAddress2, second.CounterpartyAddress)
	}
}

func TestAddressService_GetIndexedTransactionsByAddress_InvalidAddress(t *testing.T) {
	s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
	ctx := context.Background()

	got, err := s.GetIndexedTransactionsByAddress(ctx, "bad-address", 1, 20)
	if !errors.Is(err, types.ErrInvalidAddress) {
		t.Fatalf("expected ErrInvalidAddress, got %v", err)
	}

	if got != nil {
		t.Fatalf("expected got=nil, got %+v", got)
	}

	if fakeTxRepo.called {
		t.Fatalf("expected tx repo not called")
	}

	if fakeRPC.balanceCalled || fakeRPC.nonceCalled || fakeRPC.codeCalled {
		t.Fatalf("expected address rpc not called")
	}
}

func TestAddressService_GetIndexedTransactionsByAddress_InvalidPagination(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
	}{
		{
			name:     "page zero",
			page:     0,
			pageSize: 20,
		},
		{
			name:     "page negative",
			page:     -1,
			pageSize: 20,
		},
		{
			name:     "page size zero",
			page:     1,
			pageSize: 0,
		},
		{
			name:     "page size negative",
			page:     1,
			pageSize: -1,
		},
		{
			name:     "page size too large",
			page:     1,
			pageSize: 101,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
			ctx := context.Background()

			got, err := s.GetIndexedTransactionsByAddress(
				ctx,
				"0x1111111111111111111111111111111111111111",
				tt.page,
				tt.pageSize,
			)

			if !errors.Is(err, types.ErrInvalidPagination) {
				t.Fatalf("expected ErrInvalidPagination, got %v", err)
			}

			if got != nil {
				t.Fatalf("expected got=nil, got %+v", got)
			}

			if fakeTxRepo.called {
				t.Fatalf("expected tx repo not called")
			}

			if fakeRPC.balanceCalled || fakeRPC.nonceCalled || fakeRPC.codeCalled {
				t.Fatalf("expected address rpc not called")
			}
		})
	}
}

func TestAddressService_GetIndexedTransactionsByAddress_ReturnsEmptyList(t *testing.T) {
	s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
	ctx := context.Background()

	fakeTxRepo.listTransactions = []models.Transaction{}

	got, err := s.GetIndexedTransactionsByAddress(
		ctx,
		"0x1111111111111111111111111111111111111111",
		1,
		20,
	)
	if err != nil {
		t.Fatalf("get indexed transactions by address: %v", err)
	}

	if got == nil {
		t.Fatalf("expected result, got nil")
	}

	if got.Page != 1 {
		t.Fatalf("expected page=1, got %d", got.Page)
	}

	if got.PageSize != 20 {
		t.Fatalf("expected page_size=20, got %d", got.PageSize)
	}

	if len(got.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(got.Items))
	}

	if !fakeTxRepo.called {
		t.Fatalf("expected tx repo called")
	}

	if fakeTxRepo.gotAddress != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("expected repo address normalized, got %s", fakeTxRepo.gotAddress)
	}

	if fakeTxRepo.gotLimit != 20 {
		t.Fatalf("expected limit=20, got %d", fakeTxRepo.gotLimit)
	}

	if fakeTxRepo.gotOffset != 0 {
		t.Fatalf("expected offset=0, got %d", fakeTxRepo.gotOffset)
	}

	if fakeRPC.balanceCalled || fakeRPC.nonceCalled || fakeRPC.codeCalled {
		t.Fatalf("expected address rpc not called")
	}
}

func TestAddressService_GetIndexedTransactionsByAddress_ReturnsRepoError(t *testing.T) {
	s, fakeRPC, fakeTxRepo := setupAddressServiceTest()
	ctx := context.Background()

	errRepo := errors.New("repo failed")
	fakeTxRepo.err = errRepo

	got, err := s.GetIndexedTransactionsByAddress(
		ctx,
		"0x1111111111111111111111111111111111111111",
		1,
		20,
	)

	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got %v", err)
	}

	if got != nil {
		t.Fatalf("expected got=nil, got %+v", got)
	}

	if !fakeTxRepo.called {
		t.Fatalf("expected tx repo called")
	}

	if fakeTxRepo.gotLimit != 20 {
		t.Fatalf("expected limit=20, got %d", fakeTxRepo.gotLimit)
	}

	if fakeTxRepo.gotOffset != 0 {
		t.Fatalf("expected offset=0, got %d", fakeTxRepo.gotOffset)
	}

	if fakeRPC.balanceCalled || fakeRPC.nonceCalled || fakeRPC.codeCalled {
		t.Fatalf("expected address rpc not called")
	}
}

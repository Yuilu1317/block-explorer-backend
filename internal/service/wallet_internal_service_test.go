package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"block-explorer-backend/internal/db/models"
)

func TestWalletInternalService_ListCompletedBlocks_ChainIDMismatchReturnsError(t *testing.T) {
	svc := NewWalletInternalService(
		11155111,
		&fakeWalletCompletedBlockReader{},
		&fakeWalletCompletedTransactionReader{},
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), 1, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "unexpected chain_id") {
		t.Fatalf("expected unexpected chain_id error, got %q", err.Error())
	}
}

func TestWalletInternalService_ListCompletedBlocks_InvalidFromBlockReturnsError(t *testing.T) {
	svc := NewWalletInternalService(
		11155111,
		&fakeWalletCompletedBlockReader{},
		&fakeWalletCompletedTransactionReader{},
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, -1, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "from_block must be non-negative") {
		t.Fatalf("expected from_block error, got %q", err.Error())
	}
}

func TestWalletInternalService_ListCompletedBlocks_InvalidLimitReturnsError(t *testing.T) {
	svc := NewWalletInternalService(
		11155111,
		&fakeWalletCompletedBlockReader{},
		&fakeWalletCompletedTransactionReader{},
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "limit must be positive") {
		t.Fatalf("expected limit error, got %q", err.Error())
	}
}

func TestWalletInternalService_ListCompletedBlocks_EmptyBlocksReturnsEmptySlice(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{},
	}

	txReader := &fakeWalletCompletedTransactionReader{}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.ChainID != 11155111 {
		t.Fatalf("expected chain id 11155111, got %d", resp.ChainID)
	}

	if resp.Blocks == nil {
		t.Fatal("expected empty blocks slice, got nil")
	}

	if len(resp.Blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(resp.Blocks))
	}

	if txReader.called {
		t.Fatal("expected transaction reader not to be called when blocks are empty")
	}
}

func TestWalletInternalService_ListCompletedBlocks_MapsBlocksAndTransactions(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
			{
				Number:     101,
				Hash:       "0xblock101",
				ParentHash: "0xblock100",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				BlockNumber:   100,
				Hash:          "0xtx100a",
				FromAddress:   "0xfrom100a",
				ToAddress:     "0xto100a",
				ValueWei:      "100",
				ReceiptStatus: uint64Ptr(1),
			},
			{
				BlockNumber:   100,
				Hash:          "0xtx100b",
				FromAddress:   "0xfrom100b",
				ToAddress:     "0xto100b",
				ValueWei:      "200",
				ReceiptStatus: uint64Ptr(0),
			},
			{
				BlockNumber:   101,
				Hash:          "0xtx101a",
				FromAddress:   "0xfrom101a",
				ToAddress:     "0xto101a",
				ValueWei:      "300",
				ReceiptStatus: uint64Ptr(1),
			},
		},
	}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp.ChainID != 11155111 {
		t.Fatalf("expected chain id 11155111, got %d", resp.ChainID)
	}

	if len(resp.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(resp.Blocks))
	}

	if !uint64SlicesEqual(txReader.gotBlockNumbers, []uint64{100, 101}) {
		t.Fatalf("expected block numbers [100 101], got %+v", txReader.gotBlockNumbers)
	}

	block100 := resp.Blocks[0]
	if block100.Number != 100 {
		t.Fatalf("expected block 100, got %d", block100.Number)
	}

	if block100.Hash != "0xblock100" {
		t.Fatalf("expected block hash 0xblock100, got %q", block100.Hash)
	}

	if block100.ParentHash != "0xblock99" {
		t.Fatalf("expected parent hash 0xblock99, got %q", block100.ParentHash)
	}

	if len(block100.Transactions) != 2 {
		t.Fatalf("expected 2 transactions for block 100, got %d", len(block100.Transactions))
	}

	if block100.Transactions[0].TxHash != "0xtx100a" {
		t.Fatalf("expected tx hash 0xtx100a, got %q", block100.Transactions[0].TxHash)
	}

	if block100.Transactions[0].FromAddress != "0xfrom100a" {
		t.Fatalf("expected from address 0xfrom100a, got %q", block100.Transactions[0].FromAddress)
	}

	if block100.Transactions[0].ToAddress != "0xto100a" {
		t.Fatalf("expected to address 0xto100a, got %q", block100.Transactions[0].ToAddress)
	}

	if block100.Transactions[0].AmountWei != "100" {
		t.Fatalf("expected amount wei 100, got %q", block100.Transactions[0].AmountWei)
	}

	if block100.Transactions[0].ReceiptStatus != 1 {
		t.Fatalf("expected receipt status 1, got %d", block100.Transactions[0].ReceiptStatus)
	}

	if block100.Transactions[1].ReceiptStatus != 0 {
		t.Fatalf("expected receipt status 0 to be included, got %d", block100.Transactions[1].ReceiptStatus)
	}

	block101 := resp.Blocks[1]
	if block101.Number != 101 {
		t.Fatalf("expected block 101, got %d", block101.Number)
	}

	if len(block101.Transactions) != 1 {
		t.Fatalf("expected 1 transaction for block 101, got %d", len(block101.Transactions))
	}

	if block101.Transactions[0].TxHash != "0xtx101a" {
		t.Fatalf("expected tx hash 0xtx101a, got %q", block101.Transactions[0].TxHash)
	}
}

func TestWalletInternalService_ListCompletedBlocks_BlockWithoutTransactionsReturnsEmptyTransactionsSlice(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
			{
				Number:     101,
				Hash:       "0xblock101",
				ParentHash: "0xblock100",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				BlockNumber:   100,
				Hash:          "0xtx100",
				FromAddress:   "0xfrom",
				ToAddress:     "0xto",
				ValueWei:      "100",
				ReceiptStatus: uint64Ptr(1),
			},
		},
	}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(resp.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(resp.Blocks))
	}

	block101 := resp.Blocks[1]
	if block101.Number != 101 {
		t.Fatalf("expected block 101, got %d", block101.Number)
	}

	if block101.Transactions == nil {
		t.Fatal("expected empty transactions slice, got nil")
	}

	if len(block101.Transactions) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(block101.Transactions))
	}
}

func TestWalletInternalService_ListCompletedBlocks_NilReceiptStatusReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				BlockNumber:   100,
				Hash:          "0xtx100",
				FromAddress:   "0xfrom",
				ToAddress:     "0xto",
				ValueWei:      "100",
				ReceiptStatus: nil,
			},
		},
	}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "receipt_status is nil") {
		t.Fatalf("expected nil receipt status error, got %q", err.Error())
	}
}

func TestWalletInternalService_ListCompletedBlocks_InvalidReceiptStatusReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				BlockNumber:   100,
				Hash:          "0xtx100",
				FromAddress:   "0xfrom",
				ToAddress:     "0xto",
				ValueWei:      "100",
				ReceiptStatus: uint64Ptr(2),
			},
		},
	}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "invalid receipt_status") {
		t.Fatalf("expected invalid receipt status error, got %q", err.Error())
	}
}

func TestWalletInternalService_ListCompletedBlocks_BlockReaderErrorReturnsError(t *testing.T) {
	blockErr := errors.New("block repo failed")

	blockReader := &fakeWalletCompletedBlockReader{
		err: blockErr,
	}

	txReader := &fakeWalletCompletedTransactionReader{}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "list wallet completed block rows") {
		t.Fatalf("expected block reader context, got %q", err.Error())
	}

	if !errors.Is(err, blockErr) {
		t.Fatalf("expected error to wrap blockErr, got %v", err)
	}
}

func TestWalletInternalService_ListCompletedBlocks_TransactionReaderErrorReturnsError(t *testing.T) {
	txErr := errors.New("tx repo failed")

	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		err: txErr,
	}

	svc := NewWalletInternalService(11155111, blockReader, txReader)

	resp, err := svc.ListCompletedBlocks(context.Background(), 11155111, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "list wallet completed transaction rows") {
		t.Fatalf("expected transaction reader context, got %q", err.Error())
	}

	if !errors.Is(err, txErr) {
		t.Fatalf("expected error to wrap txErr, got %v", err)
	}
}

func TestMapReceiptStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  *uint64
		want    int16
		wantErr string
	}{
		{
			name:   "status zero",
			status: uint64Ptr(0),
			want:   0,
		},
		{
			name:   "status one",
			status: uint64Ptr(1),
			want:   1,
		},
		{
			name:    "nil status",
			status:  nil,
			wantErr: "receipt_status is nil",
		},
		{
			name:    "invalid status",
			status:  uint64Ptr(2),
			wantErr: "invalid receipt_status",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			got, err := mapReceiptStatus("0xtx", tt.status)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error to contain %q, got %q", tt.wantErr, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

type fakeWalletCompletedBlockReader struct {
	blocks []models.Block
	err    error

	gotFromBlock int64
	gotLimit     int
}

func (f *fakeWalletCompletedBlockReader) ListWalletCompletedBlockRows(
	ctx context.Context,
	fromBlock int64,
	limit int,
) ([]models.Block, error) {
	f.gotFromBlock = fromBlock
	f.gotLimit = limit

	if f.err != nil {
		return nil, f.err
	}

	return f.blocks, nil
}

type fakeWalletCompletedTransactionReader struct {
	txs []models.Transaction
	err error

	called          bool
	gotBlockNumbers []uint64
}

func (f *fakeWalletCompletedTransactionReader) ListWalletCompletedTransactionRows(
	ctx context.Context,
	blockNumbers []uint64,
) ([]models.Transaction, error) {
	f.called = true
	f.gotBlockNumbers = append([]uint64(nil), blockNumbers...)

	if f.err != nil {
		return nil, f.err
	}

	return f.txs, nil
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func uint64SlicesEqual(a []uint64, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

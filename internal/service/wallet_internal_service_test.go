package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
)

const (
	walletServiceTestChainID    int64  = 11155111
	walletServiceOtherChainID   int64  = 1
	walletServiceTestSyncTarget string = "safe"
)

func TestWalletInternalService_ListCompletedBlocks_ChainIDMismatchReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}
	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceOtherChainID, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "unexpected chain_id") {
		t.Fatalf("expected unexpected chain_id error, got %q", err.Error())
	}

	if blockReader.called {
		t.Fatal("expected block reader not called")
	}

	if txReader.called {
		t.Fatal("expected transaction reader not called")
	}
}

func TestWalletInternalService_ListCompletedBlocks_InvalidFromBlockReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}
	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, -1, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "from_block must be non-negative") {
		t.Fatalf("expected from_block error, got %q", err.Error())
	}

	if blockReader.called {
		t.Fatal("expected block reader not called")
	}

	if txReader.called {
		t.Fatal("expected transaction reader not called")
	}
}

func TestWalletInternalService_ListCompletedBlocks_InvalidLimitReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}
	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "limit must be positive") {
		t.Fatalf("expected limit error, got %q", err.Error())
	}

	if blockReader.called {
		t.Fatal("expected block reader not called")
	}

	if txReader.called {
		t.Fatal("expected transaction reader not called")
	}
}

func TestWalletInternalService_ListCompletedBlocks_EmptyBlocksReturnsEmptySlice(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{},
	}

	txReader := &fakeWalletCompletedTransactionReader{}
	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.ChainID != walletServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", walletServiceTestChainID, resp.ChainID)
	}

	if resp.Blocks == nil {
		t.Fatal("expected empty blocks slice, got nil")
	}

	if len(resp.Blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(resp.Blocks))
	}

	if !blockReader.called {
		t.Fatal("expected block reader called")
	}

	if blockReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected block reader chain_id=%d, got %d", walletServiceTestChainID, blockReader.gotChainID)
	}

	if blockReader.gotFromBlock != 100 {
		t.Fatalf("expected from_block=100, got %d", blockReader.gotFromBlock)
	}

	if blockReader.gotLimit != 10 {
		t.Fatalf("expected limit=10, got %d", blockReader.gotLimit)
	}

	if txReader.called {
		t.Fatal("expected transaction reader not called when blocks are empty")
	}
}

func TestWalletInternalService_ListCompletedBlocks_MapsBlocksAndTransactions(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				ChainID:    walletServiceTestChainID,
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
			{
				ChainID:    walletServiceTestChainID,
				Number:     101,
				Hash:       "0xblock101",
				ParentHash: "0xblock100",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				ChainID:       walletServiceTestChainID,
				BlockNumber:   100,
				Hash:          "0xtx100a",
				FromAddress:   "0xfrom100a",
				ToAddress:     "0xto100a",
				ValueWei:      "100",
				ReceiptStatus: uint64Ptr(1),
			},
			{
				ChainID:       walletServiceTestChainID,
				BlockNumber:   100,
				Hash:          "0xtx100b",
				FromAddress:   "0xfrom100b",
				ToAddress:     "0xto100b",
				ValueWei:      "200",
				ReceiptStatus: uint64Ptr(0),
			},
			{
				ChainID:       walletServiceTestChainID,
				BlockNumber:   101,
				Hash:          "0xtx101a",
				FromAddress:   "0xfrom101a",
				ToAddress:     "0xto101a",
				ValueWei:      "300",
				ReceiptStatus: uint64Ptr(1),
			},
		},
	}

	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.ChainID != walletServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", walletServiceTestChainID, resp.ChainID)
	}

	if !blockReader.called {
		t.Fatal("expected block reader called")
	}

	if blockReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected block reader chain_id=%d, got %d", walletServiceTestChainID, blockReader.gotChainID)
	}

	if !txReader.called {
		t.Fatal("expected transaction reader called")
	}

	if txReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected transaction reader chain_id=%d, got %d", walletServiceTestChainID, txReader.gotChainID)
	}

	if !uint64SlicesEqual(txReader.gotBlockNumbers, []uint64{100, 101}) {
		t.Fatalf("expected block numbers [100 101], got %+v", txReader.gotBlockNumbers)
	}

	if len(resp.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(resp.Blocks))
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
				ChainID:    walletServiceTestChainID,
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
			{
				ChainID:    walletServiceTestChainID,
				Number:     101,
				Hash:       "0xblock101",
				ParentHash: "0xblock100",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				ChainID:       walletServiceTestChainID,
				BlockNumber:   100,
				Hash:          "0xtx100",
				FromAddress:   "0xfrom",
				ToAddress:     "0xto",
				ValueWei:      "100",
				ReceiptStatus: uint64Ptr(1),
			},
		},
	}

	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp.ChainID != walletServiceTestChainID {
		t.Fatalf("expected chain_id=%d, got %d", walletServiceTestChainID, resp.ChainID)
	}

	if txReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected tx reader chain_id=%d, got %d", walletServiceTestChainID, txReader.gotChainID)
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
				ChainID:    walletServiceTestChainID,
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				ChainID:       walletServiceTestChainID,
				BlockNumber:   100,
				Hash:          "0xtx100",
				FromAddress:   "0xfrom",
				ToAddress:     "0xto",
				ValueWei:      "100",
				ReceiptStatus: nil,
			},
		},
	}

	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if txReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected tx reader chain_id=%d, got %d", walletServiceTestChainID, txReader.gotChainID)
	}

	if !strings.Contains(err.Error(), "receipt_status is nil") {
		t.Fatalf("expected nil receipt status error, got %q", err.Error())
	}
}

func TestWalletInternalService_ListCompletedBlocks_InvalidReceiptStatusReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				ChainID:    walletServiceTestChainID,
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		txs: []models.Transaction{
			{
				ChainID:       walletServiceTestChainID,
				BlockNumber:   100,
				Hash:          "0xtx100",
				FromAddress:   "0xfrom",
				ToAddress:     "0xto",
				ValueWei:      "100",
				ReceiptStatus: uint64Ptr(2),
			},
		},
	}

	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if txReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected tx reader chain_id=%d, got %d", walletServiceTestChainID, txReader.gotChainID)
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
	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !blockReader.called {
		t.Fatal("expected block reader called")
	}

	if blockReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected block reader chain_id=%d, got %d", walletServiceTestChainID, blockReader.gotChainID)
	}

	if !strings.Contains(err.Error(), "list wallet completed block rows") {
		t.Fatalf("expected block reader context, got %q", err.Error())
	}

	if !errors.Is(err, blockErr) {
		t.Fatalf("expected error to wrap blockErr, got %v", err)
	}

	if txReader.called {
		t.Fatal("expected transaction reader not called")
	}
}

func TestWalletInternalService_ListCompletedBlocks_TransactionReaderErrorReturnsError(t *testing.T) {
	txErr := errors.New("tx repo failed")

	blockReader := &fakeWalletCompletedBlockReader{
		blocks: []models.Block{
			{
				ChainID:    walletServiceTestChainID,
				Number:     100,
				Hash:       "0xblock100",
				ParentHash: "0xblock99",
			},
		},
	}

	txReader := &fakeWalletCompletedTransactionReader{
		err: txErr,
	}

	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.ListCompletedBlocks(context.Background(), walletServiceTestChainID, 100, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if blockReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected block reader chain_id=%d, got %d", walletServiceTestChainID, blockReader.gotChainID)
	}

	if txReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected tx reader chain_id=%d, got %d", walletServiceTestChainID, txReader.gotChainID)
	}

	if !strings.Contains(err.Error(), "list wallet completed transaction rows") {
		t.Fatalf("expected transaction reader context, got %q", err.Error())
	}

	if !errors.Is(err, txErr) {
		t.Fatalf("expected error to wrap txErr, got %v", err)
	}
}

func TestWalletInternalService_GetSyncStatus_ChainIDMismatchReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}
	statusReader := &fakeWalletSyncStatusReader{}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.GetSyncStatus(context.Background(), walletServiceOtherChainID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !strings.Contains(err.Error(), "unexpected chain_id") {
		t.Fatalf("expected unexpected chain_id error, got %q", err.Error())
	}

	if statusReader.called {
		t.Fatal("expected sync status reader not called")
	}
}

func TestWalletInternalService_GetSyncStatus_Success(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}

	statusReader := &fakeWalletSyncStatusReader{
		block: &models.Block{
			ChainID: walletServiceTestChainID,
			Number:  123,
			Hash:    "0xblock123",
		},
		found: true,
	}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.GetSyncStatus(context.Background(), walletServiceTestChainID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if !statusReader.called {
		t.Fatal("expected sync status reader called")
	}

	if statusReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected status reader chain_id=%d, got %d", walletServiceTestChainID, statusReader.gotChainID)
	}

	if resp.ChainID != walletServiceTestChainID {
		t.Fatalf("expected response chain_id=%d, got %d", walletServiceTestChainID, resp.ChainID)
	}

	if resp.SyncTarget != walletServiceTestSyncTarget {
		t.Fatalf("expected sync target=%s, got %s", walletServiceTestSyncTarget, resp.SyncTarget)
	}

	if resp.LatestCompletedBlock == nil {
		t.Fatal("expected latest completed block, got nil")
	}

	if resp.LatestCompletedBlock.Number != 123 {
		t.Fatalf("expected latest block number=123, got %d", resp.LatestCompletedBlock.Number)
	}

	if resp.LatestCompletedBlock.Hash != "0xblock123" {
		t.Fatalf("expected latest block hash=0xblock123, got %s", resp.LatestCompletedBlock.Hash)
	}
}

func TestWalletInternalService_GetSyncStatus_NotFoundReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}

	statusReader := &fakeWalletSyncStatusReader{
		block: nil,
		found: false,
	}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.GetSyncStatus(context.Background(), walletServiceTestChainID)
	if !errors.Is(err, types.ErrLatestCompletedBlockNotFound) {
		t.Fatalf("expected ErrLatestCompletedBlockNotFound, got %v", err)
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !statusReader.called {
		t.Fatal("expected sync status reader called")
	}

	if statusReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected status reader chain_id=%d, got %d", walletServiceTestChainID, statusReader.gotChainID)
	}
}

func TestWalletInternalService_GetSyncStatus_ReaderErrorReturnsError(t *testing.T) {
	blockReader := &fakeWalletCompletedBlockReader{}
	txReader := &fakeWalletCompletedTransactionReader{}

	statusErr := errors.New("status repo failed")
	statusReader := &fakeWalletSyncStatusReader{
		err: statusErr,
	}

	svc := NewWalletInternalService(
		walletServiceTestChainID,
		walletServiceTestSyncTarget,
		blockReader,
		txReader,
		statusReader,
	)

	resp, err := svc.GetSyncStatus(context.Background(), walletServiceTestChainID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}

	if !statusReader.called {
		t.Fatal("expected sync status reader called")
	}

	if statusReader.gotChainID != walletServiceTestChainID {
		t.Fatalf("expected status reader chain_id=%d, got %d", walletServiceTestChainID, statusReader.gotChainID)
	}

	if !strings.Contains(err.Error(), "get latest completed block") {
		t.Fatalf("expected wrapped get latest completed block error, got %q", err.Error())
	}

	if !errors.Is(err, statusErr) {
		t.Fatalf("expected error to wrap statusErr, got %v", err)
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

	called       bool
	gotChainID   int64
	gotFromBlock int64
	gotLimit     int
}

func (f *fakeWalletCompletedBlockReader) ListWalletCompletedBlockRows(
	ctx context.Context,
	chainID int64,
	fromBlock int64,
	limit int,
) ([]models.Block, error) {
	f.called = true
	f.gotChainID = chainID
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
	gotChainID      int64
	gotBlockNumbers []uint64
}

func (f *fakeWalletCompletedTransactionReader) ListWalletCompletedTransactionRows(
	ctx context.Context,
	chainID int64,
	blockNumbers []uint64,
) ([]models.Transaction, error) {
	f.called = true
	f.gotChainID = chainID
	f.gotBlockNumbers = append([]uint64(nil), blockNumbers...)

	if f.err != nil {
		return nil, f.err
	}

	return f.txs, nil
}

type fakeWalletSyncStatusReader struct {
	block *models.Block
	found bool
	err   error

	called     bool
	gotChainID int64
}

func (f *fakeWalletSyncStatusReader) GetLatestCompletedBlock(
	ctx context.Context,
	chainID int64,
) (*models.Block, bool, error) {
	f.called = true
	f.gotChainID = chainID

	if f.err != nil {
		return nil, false, f.err
	}

	return f.block, f.found, nil
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

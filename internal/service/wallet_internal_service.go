package service

import (
	"context"
	"fmt"

	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
)

// WalletCompletedBlockReader from block_repo
type WalletCompletedBlockReader interface {
	ListWalletCompletedBlockRows(
		ctx context.Context,
		chainID int64,
		fromBlock int64,
		limit int,
	) ([]models.Block, error)
}

// WalletCompletedBlockReader from tx_repo
type WalletCompletedTransactionReader interface {
	ListWalletCompletedTransactionRows(
		ctx context.Context,
		chainID int64,
		blockNumbers []uint64,
	) ([]models.Transaction, error)
}

type WalletSyncStatusReader interface {
	GetLatestCompletedBlock(
		ctx context.Context,
		chainID int64,
	) (*models.Block, bool, error)
}

type WalletInternalService struct {
	chainID    int64
	syncTarget string

	walletCompletedBlockReader       WalletCompletedBlockReader
	walletCompletedTransactionReader WalletCompletedTransactionReader
	walletSyncStatusReader           WalletSyncStatusReader
}

func NewWalletInternalService(
	chainID int64,
	syncTarget string,
	walletCompletedBlockReader WalletCompletedBlockReader,
	walletCompletedTransactionReader WalletCompletedTransactionReader,
	walletSyncStatusReader WalletSyncStatusReader,
) *WalletInternalService {
	return &WalletInternalService{
		chainID:                          chainID,
		syncTarget:                       syncTarget,
		walletCompletedBlockReader:       walletCompletedBlockReader,
		walletCompletedTransactionReader: walletCompletedTransactionReader,
		walletSyncStatusReader:           walletSyncStatusReader,
	}
}

func (s *WalletInternalService) ListCompletedBlocks(
	ctx context.Context,
	chainID int64,
	fromBlock int64,
	limit int,
) (*types.WalletCompletedBlocksResponse, error) {
	if chainID != s.chainID {
		return nil, fmt.Errorf("unexpected chain_id: got=%d expected=%d", chainID, s.chainID)
	}

	if fromBlock < 0 {
		return nil, fmt.Errorf("from_block must be non-negative")
	}

	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	blocks, err := s.walletCompletedBlockReader.ListWalletCompletedBlockRows(ctx, s.chainID, fromBlock, limit)
	if err != nil {
		return nil, fmt.Errorf("list wallet completed block rows: %w", err)
	}

	if len(blocks) == 0 {
		return &types.WalletCompletedBlocksResponse{
			ChainID: s.chainID,
			Blocks:  []types.WalletCompletedBlock{},
		}, nil
	}

	blockNumbers := make([]uint64, 0, len(blocks))
	for _, block := range blocks {
		blockNumbers = append(blockNumbers, block.Number)
	}

	txs, err := s.walletCompletedTransactionReader.ListWalletCompletedTransactionRows(ctx, s.chainID, blockNumbers)
	if err != nil {
		return nil, fmt.Errorf("list wallet completed transaction rows: %w", err)
	}

	txsByBlockNumber := make(map[uint64][]types.WalletCompletedTransaction)
	for _, tx := range txs {
		receiptStatus, err := mapReceiptStatus(tx.Hash, tx.ReceiptStatus)
		if err != nil {
			return nil, fmt.Errorf("map wallet transaction receipt status: %w", err)
		}

		txsByBlockNumber[tx.BlockNumber] = append(
			txsByBlockNumber[tx.BlockNumber],
			types.WalletCompletedTransaction{
				TxHash:        tx.Hash,
				FromAddress:   tx.FromAddress,
				ToAddress:     tx.ToAddress,
				AmountWei:     tx.ValueWei,
				ReceiptStatus: receiptStatus,
			})
	}

	responseBlocks := make([]types.WalletCompletedBlock, 0, len(blocks))
	for _, block := range blocks {
		transactions := txsByBlockNumber[block.Number]
		if transactions == nil {
			transactions = []types.WalletCompletedTransaction{}
		}

		responseBlocks = append(responseBlocks, types.WalletCompletedBlock{
			Number:       int64(block.Number),
			Hash:         block.Hash,
			ParentHash:   block.ParentHash,
			Transactions: transactions,
		})
	}

	return &types.WalletCompletedBlocksResponse{
		ChainID: s.chainID,
		Blocks:  responseBlocks,
	}, nil
}

func mapReceiptStatus(txHash string, status *uint64) (int16, error) {
	if status == nil {
		return 0, fmt.Errorf("receipt_status is nil: tx_hash=%s", txHash)
	}

	if *status > 1 {
		return 0, fmt.Errorf("invalid receipt_status: tx_hash=%s status=%d", txHash, *status)
	}

	return int16(*status), nil
}

func (s *WalletInternalService) GetSyncStatus(
	ctx context.Context,
	chainID int64,
) (*types.GetSyncStatusResponse, error) {
	if chainID != s.chainID {
		return nil, fmt.Errorf("unexpected chain_id: got=%d expected=%d", chainID, s.chainID)
	}

	block, found, err := s.walletSyncStatusReader.GetLatestCompletedBlock(ctx, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("get latest completed block: %w", err)
	}
	if !found {
		return nil, types.ErrLatestCompletedBlockNotFound
	}

	return &types.GetSyncStatusResponse{
		ChainID:    s.chainID,
		SyncTarget: s.syncTarget,
		LatestCompletedBlock: &types.CompletedBlockSummary{
			Number: int64(block.Number),
			Hash:   block.Hash,
		},
	}, nil
}

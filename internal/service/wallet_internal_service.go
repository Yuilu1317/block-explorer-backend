package service

import (
	"context"
	"fmt"

	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
)

type WalletCompletedBlockReader interface {
	ListWalletCompletedBlockRows(
		ctx context.Context,
		fromBlock int64,
		limit int,
	) ([]models.Block, error)
}

type WalletCompletedTransactionReader interface {
	ListWalletCompletedTransactionRows(
		ctx context.Context,
		blockNumbers []uint64,
	) ([]models.Transaction, error)
}

type WalletInternalService struct {
	chainID int64

	walletCompletedBlockReader       WalletCompletedBlockReader
	walletCompletedTransactionReader WalletCompletedTransactionReader
}

func NewWalletInternalService(
	chainID int64,
	walletCompletedBlockReader WalletCompletedBlockReader,
	walletCompletedTransactionReader WalletCompletedTransactionReader,
) *WalletInternalService {
	return &WalletInternalService{
		chainID:                          chainID,
		walletCompletedBlockReader:       walletCompletedBlockReader,
		walletCompletedTransactionReader: walletCompletedTransactionReader,
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

	blocks, err := s.walletCompletedBlockReader.ListWalletCompletedBlockRows(ctx, fromBlock, limit)
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

	txs, err := s.walletCompletedTransactionReader.ListWalletCompletedTransactionRows(ctx, blockNumbers)
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

package models

import "time"

type Block struct {
	ID         uint   `gorm:"primaryKey"`
	ChainID    int64  `gorm:"not null;uniqueIndex:idx_blocks_chain_number;uniqueIndex:idx_blocks_chain_hash"`
	Number     uint64 `gorm:"not null;uniqueIndex:idx_blocks_chain_number"`
	Hash       string `gorm:"size:66;not null;uniqueIndex:idx_blocks_chain_hash"`
	ParentHash string `gorm:"size:66;not null"`
	Timestamp  uint64 `gorm:"not null"`
	Miner      string `gorm:"size:42"`
	TxCount    int    `gorm:"not null"`
	GasUsed    uint64 `gorm:"not null"`
	GasLimit   uint64 `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	TransactionsSynced bool `gorm:"column:transactions_synced;not null;default:false"`
	ReceiptsSynced     bool `gorm:"column:receipts_synced;not null;default:false"`

	SyncStatus    string  `gorm:"column:sync_status;not null;default:'pending'"`
	LastSyncError *string `gorm:"column:last_sync_error"`
}

const (
	BlockSyncStatusPending            = "pending"
	BlockSyncStatusTransactionsSynced = "transactions_synced"
	BlockSyncStatusReceiptsFailed     = "receipts_failed"
	BlockSyncStatusCompleted          = "completed"
)

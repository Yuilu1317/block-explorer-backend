package models

import "time"

type Block struct {
	ID         uint   `gorm:"primaryKey"`
	Number     uint64 `gorm:"uniqueIndex;not null"`
	Hash       string `gorm:"size:66;uniqueIndex;not null"`
	ParentHash string `gorm:"size:66;not null"`
	Timestamp  uint64 `gorm:"not null"`
	Miner      string `gorm:"size:42"`
	TxCount    int    `gorm:"not null"`
	GasUsed    string `gorm:"type:varchar(50)"`
	GasLimit   string `gorm:"type:varchar(50)"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

package models

import "time"

type Transaction struct {
	ID uint `gorm:"primaryKey"`

	Hash        string `gorm:"size:66;uniqueIndex;not null"`
	BlockNumber uint64 `gorm:"index;not null"`
	BlockHash   string `gorm:"size:66;index;not null"`
	TxIndex     uint   `gorm:"not null"`

	FromAddress      string `gorm:"column:from_address;type:varchar(42);not null;index"`
	FromAddressLower string `gorm:"column:from_address_lower;type:varchar(42);not null;index"`
	
	ToAddress      string `gorm:"column:to_address;type:varchar(42);index"`
	ToAddressLower string `gorm:"column:to_address_lower;type:varchar(42);index"`

	Nonce       uint64 `gorm:"not null"`
	ValueWei    string `gorm:"type:numeric(78,0);not null"`
	GasLimit    uint64 `gorm:"not null"`
	GasPriceWei string `gorm:"type:numeric(78,0);not null"`
	InputData   string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

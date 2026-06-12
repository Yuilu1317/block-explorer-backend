package mapper

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func ToTransactionModel(
	chainID int64,
	block *ethtypes.Block,
	tx *ethtypes.Transaction,
	txIndex uint,
	from common.Address,
) *models.Transaction {
	fromAddress := from.Hex()
	fromAddressLower := strings.ToLower(fromAddress)

	toAddress := ""
	toAddressLower := ""
	if tx.To() != nil {
		toAddress = tx.To().Hex()
		toAddressLower = strings.ToLower(toAddress)
	}

	return &models.Transaction{
		ChainID:     chainID,
		Hash:        tx.Hash().Hex(),
		BlockNumber: block.NumberU64(),
		BlockHash:   block.Hash().Hex(),
		TxIndex:     txIndex,

		FromAddress:      fromAddress,
		FromAddressLower: fromAddressLower,
		ToAddress:        toAddress,
		ToAddressLower:   toAddressLower,

		Nonce:       tx.Nonce(),
		ValueWei:    tx.Value().String(),
		GasLimit:    tx.Gas(),
		GasPriceWei: tx.GasPrice().String(),
		InputData:   "0x" + common.Bytes2Hex(tx.Data()),
	}
}

func ToTxDetailDTO(chainID int64, raw *types.TxRaw) *types.TxDetailDTO {
	if raw == nil || raw.Tx == nil {
		return nil
	}

	tx := raw.Tx

	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	gasPriceWei := tx.GasPrice()
	if gasPriceWei == nil {
		gasPriceWei = big.NewInt(0)
	}

	var (
		status      *uint64
		gasUsed     *uint64
		blockNumber *uint64
	)

	if raw.Receipt != nil {
		receiptStatus := raw.Receipt.Status
		status = &receiptStatus

		used := raw.Receipt.GasUsed
		gasUsed = &used

		if raw.Receipt.BlockNumber != nil {
			bn := raw.Receipt.BlockNumber.Uint64()
			blockNumber = &bn
		}
	}

	return &types.TxDetailDTO{
		ChainID:     chainID,
		Hash:        tx.Hash().Hex(),
		FromAddress: raw.From,
		ToAddress:   to,
		ValueWei:    tx.Value().String(),
		Nonce:       tx.Nonce(),
		GasLimit:    tx.Gas(),
		GasPriceWei: gasPriceWei.String(),
		Data:        fmt.Sprintf("0x%x", tx.Data()),
		IsPending:   raw.IsPending,
		BlockNumber: blockNumber,
		Status:      status,
		GasUsed:     gasUsed,
	}
}

func ToIndexedTransactionDTO(tx *models.Transaction) *types.IndexedTransactionDTO {
	if tx == nil {
		return nil
	}

	return &types.IndexedTransactionDTO{
		ChainID:     tx.ChainID,
		Hash:        tx.Hash,
		BlockNumber: tx.BlockNumber,
		BlockHash:   tx.BlockHash,
		TxIndex:     tx.TxIndex,

		FromAddress: tx.FromAddress,
		ToAddress:   tx.ToAddress,

		Status:  tx.ReceiptStatus,
		GasUsed: tx.ReceiptGasUsed,

		Nonce:       tx.Nonce,
		ValueWei:    tx.ValueWei,
		GasLimit:    tx.GasLimit,
		GasPriceWei: tx.GasPriceWei,
		InputData:   tx.InputData,
	}
}

func ToAddressTransactionDTO(
	tx *models.Transaction,
	queryAddress string,
) *types.AddressTransactionDTO {
	if tx == nil {
		return nil
	}

	direction := "unknown"
	counterparty := ""

	switch {
	case tx.FromAddressLower == queryAddress && tx.ToAddressLower == queryAddress:
		direction = "self"
		counterparty = tx.ToAddress
	case tx.FromAddressLower == queryAddress:
		direction = "out"
		counterparty = tx.ToAddress
	case tx.ToAddressLower == queryAddress:
		direction = "in"
		counterparty = tx.FromAddress
	}

	return &types.AddressTransactionDTO{
		ChainID:     tx.ChainID,
		Hash:        tx.Hash,
		BlockNumber: tx.BlockNumber,
		BlockHash:   tx.BlockHash,
		TxIndex:     tx.TxIndex,

		FromAddress: tx.FromAddress,
		ToAddress:   tx.ToAddress,

		Status:  tx.ReceiptStatus,
		GasUsed: tx.ReceiptGasUsed,

		Direction:           direction,
		CounterpartyAddress: counterparty,

		Nonce:       tx.Nonce,
		ValueWei:    tx.ValueWei,
		GasLimit:    tx.GasLimit,
		GasPriceWei: tx.GasPriceWei,
		InputData:   tx.InputData,
	}
}

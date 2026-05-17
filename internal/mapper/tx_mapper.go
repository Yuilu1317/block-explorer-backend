package mapper

import (
	"block-explorer-backend/internal/db/models"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func ToTransactionModel(block *ethtypes.Block, tx *ethtypes.Transaction, txIndex uint, from common.Address) *models.Transaction {
	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}
	return &models.Transaction{
		Hash:        tx.Hash().Hex(),
		BlockNumber: block.NumberU64(),
		BlockHash:   block.Hash().Hex(),
		TxIndex:     txIndex,
		FromAddress: from.Hex(),
		ToAddress:   to,
		Nonce:       tx.Nonce(),
		ValueWei:    tx.Value().String(),
		GasLimit:    tx.Gas(),
		GasPriceWei: tx.GasPrice().String(),
		InputData:   "0x" + common.Bytes2Hex(tx.Data()),
	}
}

package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"

	"github.com/ethereum/go-ethereum"
)

type TxRPC interface {
	GetTransactionByHash(ctx context.Context, hash string) (*types.TxRaw, error)
}

type TxService struct {
	// txRPC is the RPC dependency used to fetch transaction data from the blockchain
	txRPC TxRPC
}

func NewTxService(txRPC TxRPC) *TxService {
	return &TxService{
		txRPC: txRPC,
	}
}

func (s *TxService) GetTxByHash(ctx context.Context, hash string) (*types.TxDetailDTO, error) {
	hash = strings.TrimSpace(hash)

	if err := utils.ValidateTxHash(hash); err != nil {
		return nil, types.ErrInvalidTxHash
	}

	raw, err := s.txRPC.GetTransactionByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, types.ErrTxNotFound
		}
		return nil, mapRPCError(err)
	}

	return s.toTxDetailDTO(raw), nil
}

func (s *TxService) toTxDetailDTO(raw *types.TxRaw) *types.TxDetailDTO {
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
		Hash:        tx.Hash().Hex(),
		From:        raw.From,
		To:          to,
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

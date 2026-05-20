package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/mapper"
	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"
	"context"
	"fmt"
	"strings"
)

type AddressRPC interface {
	GetBalance(ctx context.Context, address string) (string, error)
	GetNonce(ctx context.Context, address string) (uint64, error)
	GetCode(ctx context.Context, address string) (string, error)
}

type TxRepoToAddressService interface {
	ListTransactionsByAddress(
		ctx context.Context,
		address string,
		limit int,
		offset int,
	) ([]models.Transaction, error)
}

type AddressService struct {
	addressRPC             AddressRPC
	txRepoToAddressService TxRepoToAddressService
}

func NewAddressService(addressRPC AddressRPC, txRepoToAddressService TxRepoToAddressService) *AddressService {
	return &AddressService{
		addressRPC:             addressRPC,
		txRepoToAddressService: txRepoToAddressService,
	}
}

func (s *AddressService) GetAddress(ctx context.Context, address string) (*types.AddressInfo, error) {
	address = strings.TrimSpace(address)
	validateAddress := strings.ToLower(address)

	if err := utils.ValidateAddress(validateAddress); err != nil {
		return nil, types.ErrInvalidAddress
	}

	balance, err := s.addressRPC.GetBalance(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("get balance of address %s: %w", address, err)
	}

	nonce, err := s.addressRPC.GetNonce(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("get nonce of address %s: %w", address, err)
	}

	code, err := s.addressRPC.GetCode(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("get code of address %s: %w", address, err)
	}

	return &types.AddressInfo{
		Address:    address,
		Balance:    balance,
		Nonce:      nonce,
		IsContract: code != "0x",
	}, nil
}

func (s *AddressService) GetIndexedTransactionsByAddress(
	ctx context.Context,
	address string,
	page int,
	pageSize int,
) (*types.AddressTransactionListDTO, error) {
	address = strings.TrimSpace(address)
	address = strings.ToLower(address)

	if err := utils.ValidateAddress(address); err != nil {
		return nil, types.ErrInvalidAddress
	}

	if page <= 0 || pageSize <= 0 || pageSize > 100 {
		return nil, types.ErrInvalidPagination
	}
	limit := pageSize
	offset := (page - 1) * pageSize

	txs, err := s.txRepoToAddressService.ListTransactionsByAddress(ctx, address, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list indexed transactions by address %s: %w", address, err)
	}

	result := make([]*types.AddressTransactionDTO, 0, len(txs))
	for i := range txs {
		result = append(result, mapper.ToAddressTransactionDTO(&txs[i], address))
	}

	return &types.AddressTransactionListDTO{
		Items:    result,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

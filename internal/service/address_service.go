package service

import (
	"block-explorer-backend/internal/types"
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type AddressRPC interface {
	GetBalance(ctx context.Context, address string) (string, error)
	GetNonce(ctx context.Context, address string) (uint64, error)
	GetCode(ctx context.Context, address string) (string, error)
}

type AddressService struct {
	addressRPC AddressRPC
}

func NewAddressService(addressRPC AddressRPC) *AddressService {
	return &AddressService{
		addressRPC: addressRPC,
	}
}

func (s *AddressService) GetAddress(ctx context.Context, address string) (*types.AddressInfo, error) {
	address = strings.TrimSpace(address)

	if !isValidAddress(address) {
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

func isValidAddress(address string) bool {
	return common.IsHexAddress(address)
}

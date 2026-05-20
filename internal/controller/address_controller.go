package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AddressService interface {
	GetAddress(ctx context.Context, address string) (*types.AddressInfo, error)
	GetIndexedTransactionsByAddress(
		ctx context.Context,
		address string,
		page int,
		pageSize int,
	) (*types.AddressTransactionListDTO, error)
}

type AddressController struct {
	addressService AddressService
}

func NewAddressController(addressService AddressService) *AddressController {
	return &AddressController{
		addressService: addressService,
	}
}

func (ctl *AddressController) GetAddress(c *gin.Context) {
	ctx := c.Request.Context()

	address := c.Param("address")
	if address == "" {
		types.WriteBadRequest(c, "address is required")
		return
	}

	addressInfo, err := ctl.addressService.GetAddress(ctx, address)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, addressInfo)
}

func (ctl *AddressController) GetIndexedTransactionsByAddress(c *gin.Context) {
	ctx := c.Request.Context()

	address := c.Param("address")
	if address == "" {
		types.WriteBadRequest(c, "address is required")
		return
	}

	page := 1
	pageSize := 20

	if v := c.Query("page"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			types.WriteError(c, types.ErrInvalidPagination)
			return
		}
		page = parsed
	}

	if v := c.Query("page_size"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			types.WriteError(c, types.ErrInvalidPagination)
			return
		}
		pageSize = parsed
	}

	result, err := ctl.addressService.GetIndexedTransactionsByAddress(
		ctx,
		address,
		page,
		pageSize,
	)
	if err != nil {
		types.WriteError(c, err)
		return
	}

	types.WriteSuccess(c, result)
}

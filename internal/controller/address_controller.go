package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AddressQueryService interface {
	GetAddress(ctx context.Context, address string) (*types.AddressInfo, error)
	GetIndexedTransactionsByAddress(
		ctx context.Context,
		address string,
		page int,
		pageSize int,
	) (*types.AddressTransactionListDTO, error)
}

type AddressController struct {
	addressQueryService AddressQueryService
}

func NewAddressController(addressService AddressQueryService) *AddressController {
	return &AddressController{
		addressQueryService: addressService,
	}
}

func (ctl *AddressController) GetAddress(c *gin.Context) {
	ctx := c.Request.Context()

	address := c.Param("address")
	if address == "" {
		types.WriteBadRequest(c, "address is required")
		return
	}

	addressInfo, err := ctl.addressQueryService.GetAddress(ctx, address)
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

	result, err := ctl.addressQueryService.GetIndexedTransactionsByAddress(
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

package controller

import (
	"block-explorer-backend/internal/types"
	"context"

	"github.com/gin-gonic/gin"
)

type AddressService interface {
	GetAddress(ctx context.Context, address string) (*types.AddressInfo, error)
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

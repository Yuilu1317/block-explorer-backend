package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BlockService interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockDetailDTO, error)
}

type BlockController struct {
	blockService BlockService
}

func NewBlockController(blockService BlockService) *BlockController {
	return &BlockController{blockService: blockService}
}

func (ctl *BlockController) GetBlock(c *gin.Context) {
	ctx := c.Request.Context()

	numberStr := c.Param("number")
	if numberStr == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "block number is required",
		})
		return
	}
	number, err := strconv.ParseUint(numberStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid block number",
		})
		return
	}
	block, err := ctl.blockService.GetBlockByNumber(ctx, number)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "internal server error"

		switch {
		case errors.Is(err, types.ErrInvalidBlockNumber):
			statusCode = http.StatusBadRequest
			message = "invalid block number"
		case errors.Is(err, types.ErrBlockNotFound):
			statusCode = http.StatusNotFound
			message = "block not found"
		case errors.Is(err, types.ErrRPCTimeout):
			statusCode = http.StatusGatewayTimeout
			message = "ethereum rpc timeout"
		default:
			statusCode = http.StatusInternalServerError
			message = "internal server error"
		}

		c.JSON(statusCode, types.ErrorResponse{
			Code:    statusCode,
			Message: message,
		})
		return
	}
	c.JSON(http.StatusOK, types.SuccessResponse[*types.BlockDetailDTO]{
		Code:    0,
		Message: "success",
		Data:    block,
	})
}

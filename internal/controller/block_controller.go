package controller

import (
	"block-explorer-backend/internal/types"
	"context"
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
		types.WriteBadRequest(c, "block number is required")
		return
	}
	number, err := strconv.ParseUint(numberStr, 10, 64)
	if err != nil {
		types.WriteBadRequest(c, "invalid block number")
		return
	}
	block, err := ctl.blockService.GetBlockByNumber(ctx, number)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, block)
}

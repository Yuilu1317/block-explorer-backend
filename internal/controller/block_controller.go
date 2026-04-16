package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BlockService interface {
	GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockDetailDTO, error)
	SyncBlockToDB(ctx context.Context, number uint64) error
	SyncBlockRangeToDB(ctx context.Context, start, end uint64) (*types.BlockRangeSyncResult, error)
}

type BlockController struct {
	blockService BlockService
}

func NewBlockController(blockService BlockService) *BlockController {
	return &BlockController{blockService: blockService}
}

func parseBlockNumber(c *gin.Context) (uint64, bool) {
	numberStr := c.Param("number")
	if numberStr == "" {
		types.WriteBadRequest(c, "block number is required")
		return 0, false
	}

	number, err := strconv.ParseUint(numberStr, 10, 64)
	if err != nil {
		types.WriteBadRequest(c, "invalid block number")
		return 0, false
	}

	return number, true
}

func (ctl *BlockController) GetBlock(c *gin.Context) {
	ctx := c.Request.Context()

	number, ok := parseBlockNumber(c)
	if !ok {
		return
	}

	block, err := ctl.blockService.GetBlockByNumber(ctx, number)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, block)
}

func (ctl *BlockController) SyncBlock(c *gin.Context) {
	ctx := c.Request.Context()

	number, ok := parseBlockNumber(c)
	if !ok {
		return
	}

	err := ctl.blockService.SyncBlockToDB(ctx, number)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, number)
}

func (ctl *BlockController) SyncBlockRange(c *gin.Context) {
	ctx := c.Request.Context()

	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		types.WriteBadRequest(c, "start and end are required")
		return
	}

	start, err := strconv.ParseUint(startStr, 10, 64)
	if err != nil {
		types.WriteBadRequest(c, "invalid start")
		return
	}

	end, err := strconv.ParseUint(endStr, 10, 64)
	if err != nil {
		types.WriteBadRequest(c, "invalid end")
		return
	}

	result, err := ctl.blockService.SyncBlockRangeToDB(ctx, start, end)
	if err != nil {
		types.WriteError(c, err)
		return
	}

	types.WriteSuccess(c, result)
}

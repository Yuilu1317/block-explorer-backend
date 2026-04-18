package controller

import (
	"block-explorer-backend/internal/types"

	"github.com/gin-gonic/gin"
)

type IndexerController struct {
	blockService BlockService
}

func NewIndexerController(blockService BlockService) *IndexerController {
	return &IndexerController{blockService: blockService}
}

func (ctl *IndexerController) GetStatus(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := ctl.blockService.GetNextBlockToSync(ctx)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, status)
}

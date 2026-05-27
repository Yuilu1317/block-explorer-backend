package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"log"

	"github.com/gin-gonic/gin"
)

type BlockIndexer interface {
	GetNextBlockToSync(ctx context.Context) (*types.IndexerStatus, error)
	RunIndexerOnce(ctx context.Context) (*types.IndexerOnceResult, error)
}

type IndexerController struct {
	blockIndexer BlockIndexer
}

func NewIndexerController(blockIndexer BlockIndexer) *IndexerController {
	return &IndexerController{blockIndexer: blockIndexer}
}

func (ctl *IndexerController) GetSyncStatus(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := ctl.blockIndexer.GetNextBlockToSync(ctx)
	if err != nil {
		log.Printf("[indexer-status] error: %v\n", err)
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, status)
}

func (ctl *IndexerController) RunOnce(c *gin.Context) {
	ctx := c.Request.Context()

	onceResult, err := ctl.blockIndexer.RunIndexerOnce(ctx)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, onceResult)
}

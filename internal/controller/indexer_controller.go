package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"log"

	"github.com/gin-gonic/gin"
)

type Indexer interface {
	GetNextBlockToSync(ctx context.Context) (*types.IndexerStatus, error)
	RunIndexerOnce(ctx context.Context) (*types.IndexerOnceResult, error)
}

type IndexerController struct {
	indexer Indexer
}

func NewIndexerController(indexer Indexer) *IndexerController {
	return &IndexerController{indexer: indexer}
}

func (ctl *IndexerController) GetSyncStatus(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := ctl.indexer.GetNextBlockToSync(ctx)
	if err != nil {
		log.Printf("[indexer-status] error: %v\n", err)
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, status)
}

func (ctl *IndexerController) RunOnce(c *gin.Context) {
	ctx := c.Request.Context()

	onceResult, err := ctl.indexer.RunIndexerOnce(ctx)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	types.WriteSuccess(c, onceResult)
}

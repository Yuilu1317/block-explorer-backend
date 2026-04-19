package api

import (
	"github.com/gin-gonic/gin"

	"block-explorer-backend/internal/controller"
)

func NewRouter(
	txController *controller.TxController,
	blockController *controller.BlockController,
	addressController *controller.AddressController,
	debugController *controller.DebugController,
	indexerController *controller.IndexerController,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	debugGroup := r.Group("/debug")
	{
		debugGroup.GET("/db-stats", debugController.DBStats)
	}

	indexerGroup := r.Group("/indexer")
	{
		indexerGroup.GET("/status", indexerController.GetSyncStatus)
		indexerGroup.POST("/run-once", indexerController.RunOnce)
	}

	txGroup := r.Group("/tx")
	{
		txGroup.GET("/:hash", txController.GetTx)
	}

	blockGroup := r.Group("/block")
	{
		blockGroup.GET("/:number", blockController.GetBlock)
		blockGroup.POST("/sync/:number", blockController.SyncBlock)
	}

	r.POST("/blocks/sync", blockController.SyncBlockRange)

	addressGroup := r.Group("/address")
	{
		addressGroup.GET("/:address", addressController.GetAddress)
	}

	return r
}

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
	walletController *controller.WalletController,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	// Public read APIs.
	txGroup := r.Group("/tx")
	{
		txGroup.GET("/:hash", txController.GetTxDetailByHashFromRPC)
	}

	blockGroup := r.Group("/block")
	{
		blockGroup.GET("/:number", blockController.GetBlock)
	}

	addressGroup := r.Group("/address")
	{
		addressGroup.GET("/:address", addressController.GetAddress)
	}

	indexedGroup := r.Group("/indexed")
	{
		indexedGroup.GET("/tx/:hash", txController.GetIndexedTransactionByHash)
		indexedGroup.GET("/address/:address/transactions", addressController.GetIndexedTransactionsByAddress)
	}

	// Internal service/admin APIs.
	internalGroup := r.Group("/internal")
	{
		walletGroup := internalGroup.Group("/wallet")
		{
			walletGroup.GET("/completed-blocks", walletController.ListCompletedBlocks)
		}

		debugGroup := internalGroup.Group("/debug")
		{
			debugGroup.GET("/db-stats", debugController.DBStats)
		}

		indexerGroup := internalGroup.Group("/indexer")
		{
			indexerGroup.GET("/status", indexerController.GetSyncStatus)
			indexerGroup.POST("/run-once", indexerController.RunOnce)
		}

		blockSyncGroup := internalGroup.Group("/blocks")
		{
			blockSyncGroup.POST("/sync", blockController.SyncBlockRange)
			blockSyncGroup.POST("/sync/:number", blockController.SyncBlock)
		}
	}

	return r
}

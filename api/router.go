package api

import (
	"github.com/gin-gonic/gin"

	"block-explorer-backend/internal/controller"
)

func NewRouter(
	txController *controller.TxController,
	blockController *controller.BlockController,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	txGroup := r.Group("/tx")
	{
		txGroup.GET("/:hash", txController.GetTx)
	}

	blockGroup := r.Group("/block")
	{
		blockGroup.GET("/:number", blockController.GetBlock)
	}
	return r
}

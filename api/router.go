package api

import (
	"github.com/gin-gonic/gin"

	"block-explorer-backend/internal/controller"
)

func NewRouter(txController *controller.TxController) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	r.GET("/tx/:hash", txController.GetTx)

	return r
}

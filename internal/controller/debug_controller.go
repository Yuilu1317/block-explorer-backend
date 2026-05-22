package controller

import (
	"block-explorer-backend/internal/db"
	"database/sql"

	"github.com/gin-gonic/gin"
)

type DebugController struct {
	sqlDB *sql.DB
}

func NewDebugController(sqlDB *sql.DB) *DebugController {
	return &DebugController{
		sqlDB: sqlDB,
	}
}

func (c *DebugController) DBStats(ctx *gin.Context) {
	stats := db.GetStats(c.sqlDB)
	ctx.JSON(200, stats)
}

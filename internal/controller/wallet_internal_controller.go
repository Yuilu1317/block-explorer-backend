package controller

import (
	"block-explorer-backend/internal/types"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type WalletService interface {
	ListCompletedBlocks(
		ctx context.Context,
		chainID int64,
		fromBlock int64,
		limit int,
	) (*types.WalletCompletedBlocksResponse, error)
}

type WalletController struct {
	walletService WalletService
}

func NewWalletController(walletService WalletService) *WalletController {
	return &WalletController{walletService: walletService}
}

const maxWalletCompletedBlocksLimit = 100

func (ctl *WalletController) ListCompletedBlocks(c *gin.Context) {
	ctx := c.Request.Context()
	chainID, err := parseRequiredPositiveInt64Query(c, "chain_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fromBlock, err := parseRequiredInt64Query(c, "from_block")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := parseRequiredIntQuery(c, "limit")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if limit > maxWalletCompletedBlocksLimit {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("limit must be less than or equal to %d", maxWalletCompletedBlocksLimit),
		})
		return
	}

	resp, err := ctl.walletService.ListCompletedBlocks(ctx, chainID, fromBlock, limit)
	if err != nil {
		types.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func parseRequiredInt64Query(c *gin.Context, name string) (int64, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", name)
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid int64", name)
	}

	if value < 0 {
		return 0, fmt.Errorf("%s must be non-negative", name)
	}

	return value, nil
}

func parseRequiredPositiveInt64Query(c *gin.Context, name string) (int64, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", name)
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid int64", name)
	}

	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return value, nil
}

func parseRequiredIntQuery(c *gin.Context, name string) (int, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", name)
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", name)
	}

	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return value, nil
}

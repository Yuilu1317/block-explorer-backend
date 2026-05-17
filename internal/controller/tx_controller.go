package controller

import (
	"block-explorer-backend/internal/types"
	"context"

	"github.com/gin-gonic/gin"
)

// TxService defines the business logic abstraction for transaction queries.
// The controller depends on this interface instead of a concrete implementation,
// which enables loose coupling and easier testing.
type TxService interface {
	GetTxDetailByHashFromRPC(ctx context.Context, hash string) (*types.TxDetailDTO, error)
	GetIndexedTransactionByHash(ctx context.Context, hash string) (*types.IndexedTransactionDTO, error)
}

// TxController handles HTTP requests related to transactions.
// It delegates business logic to the TxService layer.
type TxController struct {
	txService TxService
}

// NewTxController creates a new TxController instance with dependency injection.
// The service layer is passed in from the outside, improving modularity and testability.
func NewTxController(txService TxService) *TxController {
	return &TxController{
		txService: txService,
	}
}

// GetTxDetailByHashFromRPC handles the HTTP request for querying a transaction by hash.
// It extracts the hash from the URL, calls the service layer,
// and returns a JSON response.
func (ctl *TxController) GetTxDetailByHashFromRPC(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract the transaction hash from URL path parameter (e.g. /tx/:hash)
	hash := c.Param("hash")
	if hash == "" {
		types.WriteBadRequest(c, "transaction hash is required")
		return
	}

	// Delegate the business logic to the service layer
	tx, err := ctl.txService.GetTxDetailByHashFromRPC(ctx, hash)
	// Handle errors and map them to appropriate HTTP status codes

	if err != nil {
		types.WriteError(c, err)
		return
	}

	// Return successful response with transaction data
	types.WriteSuccess(c, tx)
}

func (ctl *TxController) GetIndexedTransactionByHash(c *gin.Context) {
	ctx := c.Request.Context()

	hash := c.Param("hash")
	if hash == "" {
		types.WriteBadRequest(c, "transaction hash is required")
		return
	}

	tx, err := ctl.txService.GetIndexedTransactionByHash(ctx, hash)
	if err != nil {
		types.WriteError(c, err)
		return
	}

	types.WriteSuccess(c, tx)
}

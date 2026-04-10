package controller

import (
	"context"
	"errors"
	"net/http"

	"block-explorer-backend/internal/types"

	"github.com/gin-gonic/gin"
)

// TxService defines the business logic abstraction for transaction queries.
// The controller depends on this interface instead of a concrete implementation,
// which enables loose coupling and easier testing.
type TxService interface {
	GetTxByHash(ctx context.Context, hash string) (*types.TxDetailDTO, error)
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

// GetTx handles the HTTP request for querying a transaction by hash.
// It extracts the hash from the URL, calls the service layer,
// and returns a JSON response.
func (ctl *TxController) GetTx(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract the transaction hash from URL path parameter (e.g. /tx/:hash)
	hash := c.Param("hash")

	// Delegate the business logic to the service layer
	tx, err := ctl.txService.GetTxByHash(ctx, hash)
	// Handle errors and map them to appropriate HTTP status codes

	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "internal server error"

		switch {
		// Invalid input (e.g. malformed hash) → 400 Bad Request
		case errors.Is(err, types.ErrInvalidTxHash):
			statusCode = http.StatusBadRequest
			message = "invalid tx hash"
		case errors.Is(err, types.ErrTxNotFound):
			statusCode = http.StatusNotFound
			message = "transaction not found"
		}

		// Return standardized error response
		c.JSON(statusCode, types.ErrorResponse{
			Code:    statusCode,
			Message: message,
		})
		return
	}

	// Return successful response with transaction data
	c.JSON(http.StatusOK, types.SuccessResponse[*types.TxDetailDTO]{
		Code:    0,
		Message: "success",
		Data:    tx,
	})
}

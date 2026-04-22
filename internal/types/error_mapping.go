package types

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MapError(err error) (int, ErrorResponse) {
	switch {
	case errors.Is(err, ErrInvalidTxHash):
		return http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid transaction hash",
		}

	case errors.Is(err, ErrTxNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "transaction not found",
		}

	case errors.Is(err, ErrInvalidBlockNumber):
		return http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid block number",
		}

	case errors.Is(err, ErrBlockNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "block not found",
		}

	case errors.Is(err, ErrInvalidAddress):
		return http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid address",
		}

	case errors.Is(err, ErrInvalidBlockRange):
		return http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid block range",
		}

	case errors.Is(err, ErrBlockRangeTooLarge):
		return http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "block range too large",
		}

	case errors.Is(err, ErrRPCTimeout), errors.Is(err, ErrDBTimeout):
		return http.StatusGatewayTimeout, ErrorResponse{
			Code:    http.StatusGatewayTimeout,
			Message: "upstream timeout",
		}

	case errors.Is(err, ErrRequestCanceled):
		return http.StatusRequestTimeout, ErrorResponse{
			Code:    http.StatusRequestTimeout,
			Message: "request canceled",
		}

	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		}
	}
}

func WriteError(c *gin.Context, err error) {
	statusCode, resp := MapError(err)
	switch {
	case errors.Is(err, ErrRequestCanceled):
		log.Printf("[warn] type=request canceled err=%v", err)

	case errors.Is(err, ErrRPCTimeout), errors.Is(err, ErrDBTimeout):
		log.Printf("[error] type=time err=%v", err)

	case statusCode >= http.StatusInternalServerError:
		log.Printf("[error] type=internal err=%v", err)

	default:
		log.Printf("[warn] type=client error err=%v", err)
	}

	c.JSON(statusCode, resp)
}

func WriteSuccess[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, SuccessResponse[T]{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func WriteBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:    http.StatusBadRequest,
		Message: message,
	})
}

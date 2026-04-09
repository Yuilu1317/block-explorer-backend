package types

import "errors"

var ErrInvalidTxHash = errors.New("invalid transaction hash")

var ErrTxNotFound = errors.New("transaction not found")

type SuccessResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

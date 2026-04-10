package types

import "errors"

var ErrInvalidTxHash = errors.New("invalid transaction hash")

var ErrTxNotFound = errors.New("transaction not found")

var ErrInvalidBlockNumber = errors.New("invalid block number")

var ErrBlockNotFound = errors.New("block not found")

var ErrRPCTimeout = errors.New("rpc timeout")

type SuccessResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

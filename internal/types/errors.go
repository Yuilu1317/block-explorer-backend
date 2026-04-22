package types

import "errors"

var (
	ErrInvalidTxHash = errors.New("invalid transaction hash")
	ErrTxNotFound    = errors.New("transaction not found")

	ErrInvalidBlockNumber = errors.New("invalid block number")
	ErrBlockNotFound      = errors.New("block not found")

	ErrInvalidAddress = errors.New("invalid address")

	ErrInvalidBlockRange  = errors.New("invalid block range")
	ErrBlockRangeTooLarge = errors.New("block range too large")

	ErrRPCTimeout = errors.New("rpc timeout")
	ErrDBTimeout  = errors.New("db timeout")

	ErrRequestCanceled = errors.New("request canceled")
)

package types

import "errors"

var (
	ErrInvalidTxHash = errors.New("invalid transaction hash")
	ErrInvalidBlockNumber = errors.New("invalid block number")
	ErrInvalidAddress = errors.New("invalid address")
	ErrInvalidBlockRange  = errors.New("invalid block range")

	ErrBlockNotFound      = errors.New("block not found")
	ErrTxNotFound    = errors.New("transaction not found")

	ErrBlockRangeTooLarge = errors.New("block range too large")

	ErrRPCTimeout = errors.New("rpc timeout")
	ErrDBTimeout  = errors.New("db timeout")

	ErrRequestCanceled = errors.New("request canceled")
)

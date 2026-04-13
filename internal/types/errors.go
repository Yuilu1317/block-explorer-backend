package types

import "errors"

var (
	ErrInvalidTxHash = errors.New("invalid transaction hash")
	ErrTxNotFound    = errors.New("transaction not found")

	ErrInvalidBlockNumber = errors.New("invalid block number")
	ErrBlockNotFound      = errors.New("block not found")

	ErrInvalidAddress = errors.New("invalid address")

	ErrRPCTimeout = errors.New("rpc timeout")
)

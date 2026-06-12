package types

import "errors"

var (
	ErrInvalidTxHash      = errors.New("invalid transaction hash")
	ErrInvalidBlockNumber = errors.New("invalid block number")
	ErrInvalidAddress     = errors.New("invalid address")
	ErrInvalidBlockRange  = errors.New("invalid block range")
	ErrInvalidPagination  = errors.New("invalid pagination")

	ErrBlockNotFound     = errors.New("block not found")
	ErrTxNotFound        = errors.New("transaction not found")
	ErrTxReceiptNotFound = errors.New("transaction receipt not found")

	ErrReorgDetected      = errors.New("reorg detected")
	ErrChainDiscontinuity = errors.New("chain discontinuity detected")

	ErrBlockRangeTooLarge = errors.New("block range too large")

	ErrRPCTimeout = errors.New("rpc timeout")
	ErrDBTimeout  = errors.New("db timeout")

	ErrRequestCanceled = errors.New("request canceled")

	ErrChainDataConflict = errors.New("chain data conflict")

	ErrLatestCompletedBlockNotFound = errors.New("latest completed block not found")
)

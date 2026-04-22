package rpc

import (
	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"
)

func mapRPCError(err error) error {
	switch {
	case utils.IsTimeout(err):
		return types.ErrRPCTimeout
	case utils.IsCanceled(err):
		return types.ErrRequestCanceled
	default:
		return err
	}
}

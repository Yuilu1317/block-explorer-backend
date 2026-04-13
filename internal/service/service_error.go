package service

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
)

func mapRPCError(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, context.Canceled):
		return types.ErrRPCTimeout
	default:
		return err
	}
}

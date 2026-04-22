package repo

import (
	"block-explorer-backend/internal/types"
	"block-explorer-backend/internal/utils"
)

func mapDBError(err error) error {
	switch {
	case utils.IsTimeout(err):
		return types.ErrDBTimeout
	case utils.IsCanceled(err):
		return types.ErrRequestCanceled
	default:
		return err
	}
}

package utils

import (
	"context"
	"errors"
)

func IsTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

func IsCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}

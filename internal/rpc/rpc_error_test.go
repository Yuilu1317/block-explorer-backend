package rpc

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"testing"
)

func TestMapRPCError_ReturnsErrRequestCanceledWhenContextCanceled(t *testing.T) {
	mapped := mapRPCError(context.Canceled)

	if !errors.Is(mapped, types.ErrRequestCanceled) {
		t.Fatalf("expected ErrRequestCanceled, got %v", mapped)
	}
}

func TestMapRPCError_ReturnsErrRPCTimeoutWhenDeadlineExceeded(t *testing.T) {
	mapped := mapRPCError(context.DeadlineExceeded)

	if !errors.Is(mapped, types.ErrRPCTimeout) {
		t.Fatalf("expected ErrRPCTimeout, got %v", mapped)
	}
}

func TestMapRPCError_ReturnsNilWhenErrorIsUnknown(t *testing.T) {
	mapped := mapRPCError(errors.New("unknown error"))

	if mapped != nil {
		t.Fatalf("expected nil, got %v", mapped)
	}
}

func TestMapRPCError_ReturnsNilWhenErrIsNil(t *testing.T) {
	mapped := mapRPCError(nil)
	if mapped != nil {
		t.Fatalf("expected nil, got %v", mapped)
	}
}

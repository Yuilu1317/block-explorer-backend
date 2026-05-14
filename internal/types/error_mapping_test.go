package types

import (
	"fmt"
	"net/http"
	"testing"
)

func TestMapError_ReturnsConflictWhenReorgDetected(t *testing.T) {
	err := fmt.Errorf("sync block 100: %w", ErrReorgDetected)

	statusCode, resp := MapError(err)

	if statusCode != http.StatusConflict {
		t.Fatalf("expected status code %d, got %d", http.StatusConflict, statusCode)
	}

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected response code %d, got %d", http.StatusConflict, resp.Code)
	}

	if resp.Message != "reorg detected" {
		t.Fatalf("expected message %q, got %q", "reorg detected", resp.Message)
	}
}

func TestMapError_ReturnsConflictWhenChainDiscontinuityDetected(t *testing.T) {
	err := fmt.Errorf("sync block 100: %w", ErrChainDiscontinuity)

	statusCode, resp := MapError(err)

	if statusCode != http.StatusConflict {
		t.Fatalf("expected status code %d, got %d", http.StatusConflict, statusCode)
	}

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected response code %d, got %d", http.StatusConflict, resp.Code)
	}

	if resp.Message != "chain discontinuity detected" {
		t.Fatalf("expected message %q, got %q", "chain discontinuity detected", resp.Message)
	}
}

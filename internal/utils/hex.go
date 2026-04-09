package utils

import (
	"strings"

	"block-explorer-backend/internal/types"
)

func ValidateTxHash(hash string) error {
	if !strings.HasPrefix(hash, "0x") {
		return types.ErrInvalidTxHash
	}

	if len(hash) != 66 {
		return types.ErrInvalidTxHash
	}

	for _, ch := range hash[2:] {
		if !isHexChar(ch) {
			return types.ErrInvalidTxHash
		}
	}

	return nil
}

func isHexChar(ch rune) bool {
	switch {
	case ch >= '0' && ch <= '9':
		return true
	case ch >= 'a' && ch <= 'f':
		return true
	case ch >= 'A' && ch <= 'F':
		return true
	default:
		return false
	}
}

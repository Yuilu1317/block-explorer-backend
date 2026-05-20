package ethutils

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

func RecoverSender(signer gethtypes.Signer, tx *gethtypes.Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, errors.New("nil transaction")
	}

	return gethtypes.Sender(signer, tx)
}

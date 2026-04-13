package types

import gethtypes "github.com/ethereum/go-ethereum/core/types"

type TxRaw struct {
	Tx        *gethtypes.Transaction
	From      string
	IsPending bool
	Receipt   *gethtypes.Receipt
}

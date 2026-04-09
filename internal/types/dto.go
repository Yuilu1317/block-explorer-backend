package types

type TxDetailDTO struct {
	Hash        string  `json:"hash"`
	From        string  `json:"from"`
	To          string  `json:"to"`
	ValueWei    string  `json:"value_wei"`
	Nonce       uint64  `json:"nonce"`
	GasLimit    uint64  `json:"gas_limit"`
	GasPriceWei string  `json:"gas_price_wei"`
	Data        string  `json:"data"`
	IsPending   bool    `json:"is_pending"`
	BlockNumber *string `json:"block_number,omitempty"`
	Status      *uint64 `json:"status,omitempty"`
	GasUsed     *uint64 `json:"gas_used,omitempty"`
}

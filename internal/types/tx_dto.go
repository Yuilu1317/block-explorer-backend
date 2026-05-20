package types

type TxDetailDTO struct {
	Hash        string  `json:"hash"`
	FromAddress string  `json:"from_address"`
	ToAddress   string  `json:"to_address"`
	ValueWei    string  `json:"value_wei"`
	Nonce       uint64  `json:"nonce"`
	GasLimit    uint64  `json:"gas_limit"`
	GasPriceWei string  `json:"gas_price_wei"`
	Data        string  `json:"data"`
	IsPending   bool    `json:"is_pending"`
	BlockNumber *uint64 `json:"block_number,omitempty"`
	Status      *uint64 `json:"status,omitempty"`
	GasUsed     *uint64 `json:"gas_used,omitempty"`
}

type IndexedTransactionDTO struct {
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"block_number"`
	BlockHash   string `json:"block_hash"`
	TxIndex     uint   `json:"tx_index"`

	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`

	Nonce       uint64 `json:"nonce"`
	ValueWei    string `json:"value_wei"`
	GasLimit    uint64 `json:"gas_limit"`
	GasPriceWei string `json:"gas_price_wei"`
	InputData   string `json:"input_data"`
}

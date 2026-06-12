package types

type TxDetailDTO struct {
	ChainID     int64   `json:"chain_id"`
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
	Status      *uint64 `json:"status"`
	GasUsed     *uint64 `json:"gas_used"`
}

type IndexedTransactionDTO struct {
	ChainID     int64  `json:"chain_id"`
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"block_number"`
	BlockHash   string `json:"block_hash"`
	TxIndex     uint   `json:"tx_index"`

	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`

	Status  *uint64 `json:"status"`
	GasUsed *uint64 `json:"gas_used"`

	Nonce       uint64 `json:"nonce"`
	ValueWei    string `json:"value_wei"`
	GasLimit    uint64 `json:"gas_limit"`
	GasPriceWei string `json:"gas_price_wei"`
	InputData   string `json:"input_data"`
}

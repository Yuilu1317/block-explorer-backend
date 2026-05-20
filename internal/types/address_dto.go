package types

type AddressInfo struct {
	Address    string `json:"address"`
	Balance    string `json:"balance"`
	Nonce      uint64 `json:"nonce"`
	IsContract bool   `json:"is_contract"`
}

type AddressTransactionDTO struct {
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"block_number"`
	BlockHash   string `json:"block_hash"`
	TxIndex     uint   `json:"tx_index"`

	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`

	Direction           string `json:"direction"`
	CounterpartyAddress string `json:"counterparty_address"`

	Nonce       uint64 `json:"nonce"`
	ValueWei    string `json:"value_wei"`
	GasLimit    uint64 `json:"gas_limit"`
	GasPriceWei string `json:"gas_price_wei"`
	InputData   string `json:"input_data"`
}

type AddressTransactionListDTO struct {
	Items    []*AddressTransactionDTO `json:"items"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

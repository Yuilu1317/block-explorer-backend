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
	BlockNumber *uint64 `json:"block_number,omitempty"`
	Status      *uint64 `json:"status,omitempty"`
	GasUsed     *uint64 `json:"gas_used,omitempty"`
}
type BlockDetailDTO struct {
	Number     uint64 `json:"number"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parent_hash"`
	Timestamp  uint64 `json:"timestamp"`
	TxCount    int    `json:"tx_count"`
	GasUsed    uint64 `json:"gas_used"`
	GasLimit   uint64 `json:"gas_limit"`
}

type AddressInfo struct {
	Address    string `json:"address"`
	Balance    string `json:"balance"`
	Nonce      uint64 `json:"nonce"`
	IsContract bool   `json:"is_contract"`
}

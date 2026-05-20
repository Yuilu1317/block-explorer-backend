package types

type BlockDetailDTO struct {
	Number     uint64 `json:"number"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parent_hash"`
	Timestamp  uint64 `json:"timestamp"`
	TxCount    int    `json:"tx_count"`
	Miner      string `json:"miner"`
	GasUsed    uint64 `json:"gas_used"`
	GasLimit   uint64 `json:"gas_limit"`
}

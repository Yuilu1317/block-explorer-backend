package types

type WalletCompletedBlocksResponse struct {
	ChainID int64                  `json:"chain_id"`
	Blocks  []WalletCompletedBlock `json:"blocks"`
}

type WalletCompletedBlock struct {
	Number       int64                        `json:"number"`
	Hash         string                       `json:"hash"`
	ParentHash   string                       `json:"parent_hash"`
	Transactions []WalletCompletedTransaction `json:"transactions"`
}

type WalletCompletedTransaction struct {
	TxHash        string `json:"tx_hash"`
	FromAddress   string `json:"from_address"`
	ToAddress     string `json:"to_address"`
	AmountWei     string `json:"amount_wei"`
	ReceiptStatus int16  `json:"receipt_status"`
}

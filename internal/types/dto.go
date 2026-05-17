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

type AddressInfo struct {
	Address    string `json:"address"`
	Balance    string `json:"balance"`
	Nonce      uint64 `json:"nonce"`
	IsContract bool   `json:"is_contract"`
}

type IndexerStatus struct {
	DBLatest   *uint64 `json:"db_latest"`
	SyncTarget string  `json:"sync_target"`
	RPCTarget  uint64  `json:"rpc_target"`
	Next       uint64  `json:"next_to_sync"`
	ShouldSync bool    `json:"should_sync"`
}

type IndexerOnceResult struct {
	DBLatest    *uint64 `json:"db_latest"`
	SyncTarget  string  `json:"sync_target"`
	RPCTarget   uint64  `json:"rpc_target"`
	NextToSync  uint64  `json:"next_to_sync"`
	Synced      bool    `json:"synced"`
	SyncedBlock *uint64 `json:"synced_block"`
}

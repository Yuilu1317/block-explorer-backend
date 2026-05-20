package types

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

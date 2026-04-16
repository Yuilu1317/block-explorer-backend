package types

type BlockRangeSyncResult struct {
	Start        uint64   `json:"start"`
	End          uint64   `json:"end"`
	Requested    uint64   `json:"requested"`
	Succeeded    uint64   `json:"succeeded"`
	Failed       uint64   `json:"failed"`
	FailedBlocks []uint64 `json:"failed_blocks,omitempty"`
}

package model

type BlockDetail struct {
	Number     uint64
	Hash       string
	ParentHash string
	Timestamp  uint64
	Miner      string
	GasLimit   uint64
	GasUsed    uint64
	TxCount    int
}

type DataSource string

const (
	DataSourceDB  DataSource = "db"
	DataSourceRPC DataSource = "rpc"
)

type BlockQueryResult struct {
	Block  BlockDetail
	Source DataSource
}

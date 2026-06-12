package mapper

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/service/model"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const blockMapperTestChainID int64 = 11155111

type blockHeaderFixture struct {
	chainID    int64
	number     uint64
	parentHash common.Hash
	timestamp  uint64
	miner      common.Address
	gasLimit   uint64
	gasUsed    uint64
	txCount    int
}

func newBlockHeaderFixture() blockHeaderFixture {
	return blockHeaderFixture{
		chainID:    blockMapperTestChainID,
		number:     100,
		parentHash: common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		timestamp:  1710000000,
		miner:      common.HexToAddress("0x000000000000000000000000000000000000beef"),
		gasLimit:   30000000,
		gasUsed:    21000,
		txCount:    0,
	}
}

func TestMapRPCBlockToQueryResult_MapsHeaderFields(t *testing.T) {
	f := newBlockHeaderFixture()

	header := &ethtypes.Header{
		Number:     big.NewInt(int64(f.number)),
		ParentHash: f.parentHash,
		Coinbase:   f.miner,
		GasLimit:   f.gasLimit,
		GasUsed:    f.gasUsed,
		Time:       f.timestamp,
	}

	block := ethtypes.NewBlockWithHeader(header)

	wantHash := block.Hash().Hex()

	got := MapRPCBlockToQueryResult(f.chainID, block)

	if got.Source != model.DataSourceRPC {
		t.Fatalf("Source mismatch: got %s, want %s", got.Source, model.DataSourceRPC)
	}

	if got.Block.ChainID != f.chainID {
		t.Fatalf("ChainID mismatch: got %d, want %d", got.Block.ChainID, f.chainID)
	}

	if got.Block.Number != f.number {
		t.Fatalf("Number mismatch: got %d, want %d", got.Block.Number, f.number)
	}

	if got.Block.Hash != wantHash {
		t.Fatalf("Hash mismatch: got %s, want %s", got.Block.Hash, wantHash)
	}

	if got.Block.ParentHash != f.parentHash.Hex() {
		t.Fatalf("ParentHash mismatch: got %s, want %s", got.Block.ParentHash, f.parentHash.Hex())
	}

	if got.Block.Timestamp != f.timestamp {
		t.Fatalf("Timestamp mismatch: got %d, want %d", got.Block.Timestamp, f.timestamp)
	}

	if got.Block.Miner != f.miner.Hex() {
		t.Fatalf("Miner mismatch: got %s, want %s", got.Block.Miner, f.miner.Hex())
	}

	if got.Block.TxCount != f.txCount {
		t.Fatalf("TxCount mismatch: got %d, want %d", got.Block.TxCount, f.txCount)
	}

	if got.Block.GasLimit != f.gasLimit {
		t.Fatalf("GasLimit mismatch: got %d, want %d", got.Block.GasLimit, f.gasLimit)
	}

	if got.Block.GasUsed != f.gasUsed {
		t.Fatalf("GasUsed mismatch: got %d, want %d", got.Block.GasUsed, f.gasUsed)
	}
}

func TestMapBlockEntityToQueryResult_MapsHeaderFields(t *testing.T) {
	f := newBlockHeaderFixture()

	wantHash := "0x1111111111111111111111111111111111111111111111111111111111111111"

	block := &models.Block{
		ChainID:    f.chainID,
		Number:     f.number,
		Hash:       wantHash,
		ParentHash: f.parentHash.Hex(),
		Timestamp:  f.timestamp,
		Miner:      f.miner.Hex(),
		GasLimit:   f.gasLimit,
		GasUsed:    f.gasUsed,
		TxCount:    f.txCount,
	}

	got := MapBlockEntityToQueryResult(block)

	if got.Source != model.DataSourceDB {
		t.Fatalf("Source mismatch: got %s, want %s", got.Source, model.DataSourceDB)
	}

	if got.Block.ChainID != f.chainID {
		t.Fatalf("ChainID mismatch: got %d, want %d", got.Block.ChainID, f.chainID)
	}

	if got.Block.Number != f.number {
		t.Fatalf("Number mismatch: got %d, want %d", got.Block.Number, f.number)
	}

	if got.Block.Hash != wantHash {
		t.Fatalf("Hash mismatch: got %s, want %s", got.Block.Hash, wantHash)
	}

	if got.Block.ParentHash != f.parentHash.Hex() {
		t.Fatalf("ParentHash mismatch: got %s, want %s", got.Block.ParentHash, f.parentHash.Hex())
	}

	if got.Block.Timestamp != f.timestamp {
		t.Fatalf("Timestamp mismatch: got %d, want %d", got.Block.Timestamp, f.timestamp)
	}

	if got.Block.Miner != f.miner.Hex() {
		t.Fatalf("Miner mismatch: got %s, want %s", got.Block.Miner, f.miner.Hex())
	}

	if got.Block.GasLimit != f.gasLimit {
		t.Fatalf("GasLimit mismatch: got %d, want %d", got.Block.GasLimit, f.gasLimit)
	}

	if got.Block.GasUsed != f.gasUsed {
		t.Fatalf("GasUsed mismatch: got %d, want %d", got.Block.GasUsed, f.gasUsed)
	}

	if got.Block.TxCount != f.txCount {
		t.Fatalf("TxCount mismatch: got %d, want %d", got.Block.TxCount, f.txCount)
	}
}

func TestMapBlockQueryResultToDTO_MapsHeaderFields(t *testing.T) {
	f := newBlockHeaderFixture()

	wantHash := "0x1111111111111111111111111111111111111111111111111111111111111111"

	block := model.BlockQueryResult{
		Block: model.BlockDetail{
			ChainID:    f.chainID,
			Number:     f.number,
			Hash:       wantHash,
			ParentHash: f.parentHash.Hex(),
			Timestamp:  f.timestamp,
			GasLimit:   f.gasLimit,
			GasUsed:    f.gasUsed,
			TxCount:    f.txCount,
			Miner:      f.miner.Hex(),
		},
		Source: model.DataSourceDB,
	}

	got := MapBlockQueryResultToDTO(block)

	if got.ChainID != f.chainID {
		t.Fatalf("ChainID mismatch: got %d, want %d", got.ChainID, f.chainID)
	}

	if got.Number != f.number {
		t.Fatalf("Number mismatch: got %d, want %d", got.Number, f.number)
	}

	if got.Hash != wantHash {
		t.Fatalf("Hash mismatch: got %s, want %s", got.Hash, wantHash)
	}

	if got.ParentHash != f.parentHash.Hex() {
		t.Fatalf("ParentHash mismatch: got %s, want %s", got.ParentHash, f.parentHash.Hex())
	}

	if got.Timestamp != f.timestamp {
		t.Fatalf("Timestamp mismatch: got %d, want %d", got.Timestamp, f.timestamp)
	}

	if got.GasLimit != f.gasLimit {
		t.Fatalf("GasLimit mismatch: got %d, want %d", got.GasLimit, f.gasLimit)
	}

	if got.GasUsed != f.gasUsed {
		t.Fatalf("GasUsed mismatch: got %d, want %d", got.GasUsed, f.gasUsed)
	}

	if got.TxCount != f.txCount {
		t.Fatalf("TxCount mismatch: got %d, want %d", got.TxCount, f.txCount)
	}

	if got.Miner != f.miner.Hex() {
		t.Fatalf("Miner mismatch: got %s, want %s", got.Miner, f.miner.Hex())
	}
}

func TestToBlockModel_MapsHeaderFields(t *testing.T) {
	f := newBlockHeaderFixture()

	header := &ethtypes.Header{
		Number:     big.NewInt(int64(f.number)),
		ParentHash: f.parentHash,
		Coinbase:   f.miner,
		GasLimit:   f.gasLimit,
		GasUsed:    f.gasUsed,
		Time:       f.timestamp,
	}

	block := ethtypes.NewBlockWithHeader(header)

	got := ToBlockModel(f.chainID, block)

	if got.ChainID != f.chainID {
		t.Fatalf("ChainID mismatch: got %d, want %d", got.ChainID, f.chainID)
	}

	if got.Number != f.number {
		t.Fatalf("Number mismatch: got %d, want %d", got.Number, f.number)
	}

	if got.Hash != block.Hash().Hex() {
		t.Fatalf("Hash mismatch: got %s, want %s", got.Hash, block.Hash().Hex())
	}

	if got.ParentHash != f.parentHash.Hex() {
		t.Fatalf("ParentHash mismatch: got %s, want %s", got.ParentHash, f.parentHash.Hex())
	}

	if got.Timestamp != f.timestamp {
		t.Fatalf("Timestamp mismatch: got %d, want %d", got.Timestamp, f.timestamp)
	}

	if got.Miner != f.miner.Hex() {
		t.Fatalf("Miner mismatch: got %s, want %s", got.Miner, f.miner.Hex())
	}

	if got.GasLimit != f.gasLimit {
		t.Fatalf("GasLimit mismatch: got %d, want %d", got.GasLimit, f.gasLimit)
	}

	if got.GasUsed != f.gasUsed {
		t.Fatalf("GasUsed mismatch: got %d, want %d", got.GasUsed, f.gasUsed)
	}

	if got.TxCount != f.txCount {
		t.Fatalf("TxCount mismatch: got %d, want %d", got.TxCount, f.txCount)
	}
}

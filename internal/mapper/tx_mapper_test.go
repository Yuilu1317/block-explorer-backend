package mapper

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func TestToIndexedTransactionDTO(t *testing.T) {
	tx := &models.Transaction{
		Hash:        "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		BlockNumber: 100,
		BlockHash:   "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		TxIndex:     2,

		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",

		Nonce:       7,
		ValueWei:    "1000000000000000000",
		GasLimit:    21000,
		GasPriceWei: "1000000000",
		InputData:   "0xabcdef",
	}

	dto := ToIndexedTransactionDTO(tx)
	if dto == nil {
		t.Fatal("expected dto, got nil")
	}

	if dto.Hash != tx.Hash {
		t.Fatalf("expected hash %q, got %q", tx.Hash, dto.Hash)
	}

	if dto.BlockNumber != tx.BlockNumber {
		t.Fatalf("expected block number %d, got %d", tx.BlockNumber, dto.BlockNumber)
	}

	if dto.BlockHash != tx.BlockHash {
		t.Fatalf("expected block hash %q, got %q", tx.BlockHash, dto.BlockHash)
	}

	if dto.TxIndex != tx.TxIndex {
		t.Fatalf("expected tx index %d, got %d", tx.TxIndex, dto.TxIndex)
	}

	if dto.FromAddress != tx.FromAddress {
		t.Fatalf("expected from address %q, got %q", tx.FromAddress, dto.FromAddress)
	}

	if dto.ToAddress != tx.ToAddress {
		t.Fatalf("expected to address %q, got %q", tx.ToAddress, dto.ToAddress)
	}

	if dto.Nonce != tx.Nonce {
		t.Fatalf("expected nonce %d, got %d", tx.Nonce, dto.Nonce)
	}

	if dto.ValueWei != tx.ValueWei {
		t.Fatalf("expected value wei %q, got %q", tx.ValueWei, dto.ValueWei)
	}

	if dto.GasLimit != tx.GasLimit {
		t.Fatalf("expected gas limit %d, got %d", tx.GasLimit, dto.GasLimit)
	}

	if dto.GasPriceWei != tx.GasPriceWei {
		t.Fatalf("expected gas price wei %q, got %q", tx.GasPriceWei, dto.GasPriceWei)
	}

	if dto.InputData != tx.InputData {
		t.Fatalf("expected input data %q, got %q", tx.InputData, dto.InputData)
	}
}

func TestToIndexedTransactionDTO_ReturnsNilWhenInputIsNil(t *testing.T) {
	dto := ToIndexedTransactionDTO(nil)
	if dto != nil {
		t.Fatalf("expected nil dto, got %+v", dto)
	}
}

func TestToTxDetailDTO(t *testing.T) {
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	tx := ethtypes.NewTransaction(
		7,
		to,
		big.NewInt(1000000000000000000),
		21000,
		big.NewInt(1000000000),
		[]byte{0xab, 0xcd},
	)

	raw := &types.TxRaw{
		Tx:        tx,
		From:      "0x1111111111111111111111111111111111111111",
		IsPending: false,
		Receipt: &ethtypes.Receipt{
			Status:      1,
			GasUsed:     21000,
			BlockNumber: big.NewInt(100),
		},
	}

	dto := ToTxDetailDTO(raw)
	if dto == nil {
		t.Fatal("expected dto, got nil")
	}

	if dto.Hash != tx.Hash().Hex() {
		t.Fatalf("expected hash %q, got %q", tx.Hash().Hex(), dto.Hash)
	}

	if dto.FromAddress != raw.From {
		t.Fatalf("expected from address %q, got %q", raw.From, dto.FromAddress)
	}

	if dto.ToAddress != to.Hex() {
		t.Fatalf("expected to address %q, got %q", to.Hex(), dto.ToAddress)
	}

	if dto.ValueWei != tx.Value().String() {
		t.Fatalf("expected value wei %q, got %q", tx.Value().String(), dto.ValueWei)
	}

	if dto.Nonce != tx.Nonce() {
		t.Fatalf("expected nonce %d, got %d", tx.Nonce(), dto.Nonce)
	}

	if dto.GasLimit != tx.Gas() {
		t.Fatalf("expected gas limit %d, got %d", tx.Gas(), dto.GasLimit)
	}

	if dto.GasPriceWei != tx.GasPrice().String() {
		t.Fatalf("expected gas price wei %q, got %q", tx.GasPrice().String(), dto.GasPriceWei)
	}

	if dto.Data != "0xabcd" {
		t.Fatalf("expected data %q, got %q", "0xabcd", dto.Data)
	}

	if dto.IsPending != raw.IsPending {
		t.Fatalf("expected is_pending %v, got %v", raw.IsPending, dto.IsPending)
	}

	if dto.Status == nil || *dto.Status != raw.Receipt.Status {
		t.Fatalf("expected status %d, got %v", raw.Receipt.Status, dto.Status)
	}

	if dto.GasUsed == nil || *dto.GasUsed != raw.Receipt.GasUsed {
		t.Fatalf("expected gas used %d, got %v", raw.Receipt.GasUsed, dto.GasUsed)
	}

	if dto.BlockNumber == nil || *dto.BlockNumber != raw.Receipt.BlockNumber.Uint64() {
		t.Fatalf("expected block number %d, got %v", raw.Receipt.BlockNumber.Uint64(), dto.BlockNumber)
	}
}

func TestToTxDetailDTO_ReturnsNilWhenInputIsNil(t *testing.T) {
	dto := ToTxDetailDTO(nil)
	if dto != nil {
		t.Fatalf("expected nil dto, got %+v", dto)
	}
}

func TestToTxDetailDTO_ReturnsNilWhenRawTxIsNil(t *testing.T) {
	dto := ToTxDetailDTO(&types.TxRaw{})
	if dto != nil {
		t.Fatalf("expected nil dto, got %+v", dto)
	}
}

func TestToTransactionModel(t *testing.T) {
	header := &ethtypes.Header{
		Number: big.NewInt(100),
	}

	block := ethtypes.NewBlockWithHeader(header)

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")

	tx := ethtypes.NewTransaction(
		7,
		to,
		big.NewInt(1000000000000000000),
		21000,
		big.NewInt(1000000000),
		[]byte{0xab, 0xcd},
	)

	txIndex := uint(2)

	model := ToTransactionModel(block, tx, txIndex, from)
	if model == nil {
		t.Fatal("expected model, got nil")
	}

	if model.Hash != tx.Hash().Hex() {
		t.Fatalf("expected hash %q, got %q", tx.Hash().Hex(), model.Hash)
	}

	if model.BlockNumber != block.NumberU64() {
		t.Fatalf("expected block number %d, got %d", block.NumberU64(), model.BlockNumber)
	}

	if model.BlockHash != block.Hash().Hex() {
		t.Fatalf("expected block hash %q, got %q", block.Hash().Hex(), model.BlockHash)
	}

	if model.TxIndex != txIndex {
		t.Fatalf("expected tx index %d, got %d", txIndex, model.TxIndex)
	}

	if model.FromAddress != from.Hex() {
		t.Fatalf("expected from address %q, got %q", from.Hex(), model.FromAddress)
	}

	if model.ToAddress != to.Hex() {
		t.Fatalf("expected to address %q, got %q", to.Hex(), model.ToAddress)
	}

	if model.Nonce != tx.Nonce() {
		t.Fatalf("expected nonce %d, got %d", tx.Nonce(), model.Nonce)
	}

	if model.ValueWei != tx.Value().String() {
		t.Fatalf("expected value wei %q, got %q", tx.Value().String(), model.ValueWei)
	}

	if model.GasLimit != tx.Gas() {
		t.Fatalf("expected gas limit %d, got %d", tx.Gas(), model.GasLimit)
	}

	if model.GasPriceWei != tx.GasPrice().String() {
		t.Fatalf("expected gas price wei %q, got %q", tx.GasPrice().String(), model.GasPriceWei)
	}

	if model.InputData != "0xabcd" {
		t.Fatalf("expected input data %q, got %q", "0xabcd", model.InputData)
	}
}

package mapper

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/service/model"
	"block-explorer-backend/internal/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func MapBlockEntityToQueryResult(b *models.Block) model.BlockQueryResult {
	return model.BlockQueryResult{
		Block: model.BlockDetail{
			Number:     b.Number,
			Hash:       b.Hash,
			ParentHash: b.ParentHash,
			Timestamp:  b.Timestamp,
			Miner:      b.Miner,
			GasLimit:   b.GasLimit,
			GasUsed:    b.GasUsed,
			TxCount:    b.TxCount,
		},
		Source: model.DataSourceDB,
	}
}

func MapRPCBlockToQueryResult(b *ethtypes.Block) model.BlockQueryResult {
	return model.BlockQueryResult{
		Block: model.BlockDetail{
			Number:     b.NumberU64(),
			Hash:       b.Hash().Hex(),
			ParentHash: b.ParentHash().Hex(),
			Timestamp:  b.Time(),
			Miner:      b.Coinbase().Hex(),
			GasLimit:   b.GasUsed(),
			GasUsed:    b.GasLimit(),
			TxCount:    len(b.Transactions()),
		},
		Source: model.DataSourceRPC,
	}
}

func MapBlockQueryResultToDTO(r model.BlockQueryResult) types.BlockDetailDTO {
	return types.BlockDetailDTO{
		Number:     r.Block.Number,
		Hash:       r.Block.Hash,
		ParentHash: r.Block.ParentHash,
		Timestamp:  r.Block.Timestamp,
		GasLimit:   r.Block.GasLimit,
		GasUsed:    r.Block.GasUsed,
		TxCount:    r.Block.TxCount,
	}
}

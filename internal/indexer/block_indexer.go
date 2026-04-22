package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"fmt"
)

type BlockRPC interface {
	GetLatestBlockNumber(ctx context.Context) (uint64, error)
}

type BlockRepository interface {
	GetLatestBlockNumber(ctx context.Context) (uint64, bool, error)
}

type BlockService interface {
	SyncBlockToDB(ctx context.Context, number uint64) error
}

type BlockIndexer struct {
	blockRPC     BlockRPC
	blockRepo    BlockRepository
	blockService BlockService
}

func NewBlockIndexer(blockRPC BlockRPC, blockRepo BlockRepository, blockService BlockService) *BlockIndexer {
	return &BlockIndexer{
		blockRPC:     blockRPC,
		blockRepo:    blockRepo,
		blockService: blockService,
	}
}

func (s *BlockIndexer) GetNextBlockToSync(ctx context.Context) (*types.IndexerStatus, error) {
	dbLatest, exists, err := s.blockRepo.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("db latest block: %w", err)
	}

	rpcLatest, err := s.blockRPC.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("rpc latest block: %w", err)
	}

	var (
		next  uint64
		dbPtr *uint64
	)

	if exists {
		dbPtr = &dbLatest
		next = dbLatest + 1
	} else {
		next = 0
	}

	return &types.IndexerStatus{
		DBLatest:   dbPtr,
		RPCLatest:  rpcLatest,
		Next:       next,
		ShouldSync: next <= rpcLatest,
	}, nil
}

func (s *BlockIndexer) RunIndexerOnce(ctx context.Context) (*types.IndexerOnceResult, error) {
	status, err := s.GetNextBlockToSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("run indexer once: get sync status: %w", err)
	}

	result := &types.IndexerOnceResult{
		DBLatest:   status.DBLatest,
		RPCLatest:  status.RPCLatest,
		NextToSync: status.Next,
		Synced:     false,
	}
	if status.Next > status.RPCLatest {
		return result, nil
	}

	if err := s.blockService.SyncBlockToDB(ctx, status.Next); err != nil {
		return nil, fmt.Errorf("run indexer once: sync block %d: %w", status.Next, err)
	}

	syncedBlock := status.Next
	result.Synced = true
	result.SyncedBlock = &syncedBlock

	return result, nil
}

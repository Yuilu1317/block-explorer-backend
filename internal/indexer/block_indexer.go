package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"fmt"
)

type SyncTargetReader interface {
	GetBlockNumberByTag(ctx context.Context, tag string) (uint64, error)
}

type BlockSyncProgressReader interface {
	GetLatestFullySyncedBlockNumber(ctx context.Context) (uint64, bool, error)
}

type BlockSynchronizer interface {
	SyncBlockToDB(ctx context.Context, number uint64) error
}

type BlockIndexer struct {
	syncTargetReader        SyncTargetReader
	blockSyncProgressReader BlockSyncProgressReader
	blockSynchronizer       BlockSynchronizer
	syncTarget              string
	startBlock              uint64
}

func NewBlockIndexer(
	syncTargetReader SyncTargetReader,
	blockSyncProgressReader BlockSyncProgressReader,
	blockSynchronizer BlockSynchronizer,
	syncTarget string,
	startBlock uint64,
) *BlockIndexer {
	return &BlockIndexer{
		syncTargetReader:        syncTargetReader,
		blockSyncProgressReader: blockSyncProgressReader,
		blockSynchronizer:       blockSynchronizer,
		syncTarget:              syncTarget,
		startBlock:              startBlock,
	}
}

func (s *BlockIndexer) GetNextBlockToSync(ctx context.Context) (*types.IndexerStatus, error) {
	dbLatest, exists, err := s.blockSyncProgressReader.GetLatestFullySyncedBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("db latest block: %w", err)
	}

	rpcTarget, err := s.syncTargetReader.GetBlockNumberByTag(ctx, s.syncTarget)
	if err != nil {
		return nil, fmt.Errorf("rpc target block by tag %s: %w", s.syncTarget, err)
	}

	var (
		next  uint64
		dbPtr *uint64
	)

	if exists {
		dbPtr = &dbLatest
		next = dbLatest + 1
	} else {
		next = s.startBlock
	}

	return &types.IndexerStatus{
		DBLatest:   dbPtr,
		SyncTarget: s.syncTarget,
		RPCTarget:  rpcTarget,
		Next:       next,
		ShouldSync: next <= rpcTarget,
	}, nil
}

func (s *BlockIndexer) RunIndexerOnce(ctx context.Context) (*types.IndexerOnceResult, error) {
	status, err := s.GetNextBlockToSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("run indexer once: get sync status: %w", err)
	}

	result := &types.IndexerOnceResult{
		DBLatest:   status.DBLatest,
		SyncTarget: status.SyncTarget,
		RPCTarget:  status.RPCTarget,
		NextToSync: status.Next,
		Synced:     false,
	}
	if !status.ShouldSync {
		return result, nil
	}

	if err := s.blockSynchronizer.SyncBlockToDB(ctx, status.Next); err != nil {
		return nil, fmt.Errorf("run indexer once: sync block %d: %w", status.Next, err)
	}

	syncedBlock := status.Next
	result.Synced = true
	result.SyncedBlock = &syncedBlock

	return result, nil
}

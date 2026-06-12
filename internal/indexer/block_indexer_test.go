package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"testing"
)

const indexerTestChainID int64 = 11155111

type fakeSyncTargetReader struct {
	targetNumber uint64
	err          error

	getTargetCalled bool
	gotTag          string
}

func (f *fakeSyncTargetReader) GetBlockNumberByTag(ctx context.Context, tag string) (uint64, error) {
	f.getTargetCalled = true
	f.gotTag = tag

	if f.err != nil {
		return 0, f.err
	}

	return f.targetNumber, nil
}

type fakeBlockSyncProgressReader struct {
	latestNumber uint64
	exists       bool
	err          error

	getLatestCalled bool
	gotChainID      int64
}

func (f *fakeBlockSyncProgressReader) GetLatestFullySyncedBlockNumber(
	ctx context.Context,
	chainID int64,
) (uint64, bool, error) {
	f.getLatestCalled = true
	f.gotChainID = chainID

	if f.err != nil {
		return 0, false, f.err
	}

	return f.latestNumber, f.exists, nil
}

type fakeBlockSynchronizer struct {
	syncCalled  bool
	syncedBlock uint64
	err         error
}

func (f *fakeBlockSynchronizer) SyncBlockToDB(ctx context.Context, number uint64) error {
	f.syncCalled = true
	f.syncedBlock = number

	if f.err != nil {
		return f.err
	}

	return nil
}

func setupTestIndexer(t *testing.T) (*BlockIndexer, *fakeSyncTargetReader, *fakeBlockSyncProgressReader, *fakeBlockSynchronizer) {
	t.Helper()

	return setupTestIndexerWithConfig(t, "safe", 0)
}

func setupTestIndexerWithConfig(
	t *testing.T,
	syncTarget string,
	startBlock uint64,
) (*BlockIndexer, *fakeSyncTargetReader, *fakeBlockSyncProgressReader, *fakeBlockSynchronizer) {
	t.Helper()

	targetReader := &fakeSyncTargetReader{}
	progressReader := &fakeBlockSyncProgressReader{}
	synchronizer := &fakeBlockSynchronizer{}

	indexer := NewBlockIndexer(
		indexerTestChainID,
		targetReader,
		progressReader,
		synchronizer,
		syncTarget,
		startBlock,
	)

	return indexer, targetReader, progressReader, synchronizer
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsDBLatestPlusOne(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)

	ctx := context.Background()

	rpc.targetNumber = 1
	repo.latestNumber = 1
	repo.exists = true

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if !repo.getLatestCalled {
		t.Fatalf("expected repo progress reader to be called")
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected rpc target reader to be called")
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest == nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=%d, got nil", repo.latestNumber)
	}

	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=1, got %v", *result.DBLatest)
	}

	if result.SyncTarget != "safe" {
		t.Fatalf("GetNextBlockToSync(): expected SyncTarget=safe, got %s", result.SyncTarget)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCTarget=1, got %v", result.RPCTarget)
	}

	if result.Next != repo.latestNumber+1 {
		t.Fatalf("GetNextBlockToSync(): expected Next=2, got %v", result.Next)
	}

	if result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=false, got true")
	}
}

func TestBlockIndexer_GetNextBlockToSync_ShouldSyncWhenRPCIsAhead(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)

	ctx := context.Background()

	rpc.targetNumber = 2
	repo.latestNumber = 1
	repo.exists = true

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest == nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=%d, got nil", repo.latestNumber)
	}

	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=1, got %v", *result.DBLatest)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCTarget=2, got %v", result.RPCTarget)
	}

	if result.Next != repo.latestNumber+1 {
		t.Fatalf("GetNextBlockToSync(): expected Next=2, got %v", result.Next)
	}

	if !result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=true, got false")
	}
}

func TestBlockIndexer_GetNextBlockToSync_StartsFromConfiguredStartBlockWhenDBIsEmpty(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexerWithConfig(t, "safe", 100)

	ctx := context.Background()

	rpc.targetNumber = 150
	repo.exists = false

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest != nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest nil, got %d", *result.DBLatest)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}

	if result.Next != 100 {
		t.Fatalf("GetNextBlockToSync(): expected Next=100, got %d", result.Next)
	}

	if !result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=true, got false")
	}
}

func TestBlockIndexer_GetNextBlockToSync_DoesNotSyncWhenStartBlockIsAboveRPCTarget(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexerWithConfig(t, "safe", 100)

	ctx := context.Background()

	rpc.targetNumber = 99
	repo.exists = false

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest != nil {
		t.Fatalf("expected DBLatest nil, got %d", *result.DBLatest)
	}

	if result.Next != 100 {
		t.Fatalf("expected Next=100, got %d", result.Next)
	}

	if result.RPCTarget != 99 {
		t.Fatalf("expected RPCTarget=99, got %d", result.RPCTarget)
	}

	if result.ShouldSync {
		t.Fatalf("expected ShouldSync=false, got true")
	}
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsErrorWhenRepoFails(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)

	ctx := context.Background()

	repo.err = types.ErrDBTimeout

	result, err := svc.GetNextBlockToSync(ctx)
	if err == nil {
		t.Fatalf("GetNextBlockToSync(): expected error")
	}

	if result != nil {
		t.Fatalf("GetNextBlockToSync(): expected nil, got %v", result)
	}

	if !errors.Is(err, types.ErrDBTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}

	if !repo.getLatestCalled {
		t.Fatalf("expected repo progress reader to be called")
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if rpc.getTargetCalled {
		t.Fatalf("expected rpc not to be called when repo fails")
	}
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsErrorWhenRPCFails(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)

	ctx := context.Background()

	repo.latestNumber = 1
	repo.exists = true
	rpc.err = types.ErrRPCTimeout

	result, err := svc.GetNextBlockToSync(ctx)
	if err == nil {
		t.Fatalf("GetNextBlockToSync(): expected error")
	}

	if result != nil {
		t.Fatalf("GetNextBlockToSync(): expected nil, got %v", result)
	}

	if !errors.Is(err, types.ErrRPCTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}

	if !repo.getLatestCalled {
		t.Fatalf("expected DB to be called when rpc fails")
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC to be called")
	}
}

func TestBlockIndexer_RunIndexerOnce_SyncsNextBlockWhenShouldSyncTrue(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)

	ctx := context.Background()

	repo.latestNumber = 1
	repo.exists = true
	rpc.targetNumber = 2

	result, err := svc.RunIndexerOnce(ctx)
	if err != nil {
		t.Fatalf("RunIndexerOnce(): %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest == nil {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got nil", repo.latestNumber)
	}

	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got %d", repo.latestNumber, *result.DBLatest)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("RunIndexerOnce(): expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}

	if result.SyncTarget != "safe" {
		t.Fatalf("RunIndexerOnce(): expected SyncTarget=safe, got %s", result.SyncTarget)
	}

	wantNext := repo.latestNumber + 1
	if result.NextToSync != wantNext {
		t.Fatalf("RunIndexerOnce(): expected NextToSync=%d, got %d", wantNext, result.NextToSync)
	}

	if !result.Synced {
		t.Fatalf("RunIndexerOnce(): expected Synced=true, got false")
	}

	if result.SyncedBlock == nil {
		t.Fatalf("RunIndexerOnce(): expected SyncedBlock=%d, got nil", wantNext)
	}

	if *result.SyncedBlock != wantNext {
		t.Fatalf("RunIndexerOnce(): expected SyncedBlock=%d, got %d", wantNext, *result.SyncedBlock)
	}

	if !service.syncCalled {
		t.Fatalf("RunIndexerOnce(): expected SyncBlockToDB called")
	}

	if service.syncedBlock != wantNext {
		t.Fatalf("RunIndexerOnce(): expected syncedBlock=%d, got %d", wantNext, service.syncedBlock)
	}
}

func TestBlockIndexer_RunIndexerOnce_DoesNotSyncWhenNoNewBlock(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)

	ctx := context.Background()

	repo.latestNumber = 1
	repo.exists = true
	rpc.targetNumber = 1

	result, err := svc.RunIndexerOnce(ctx)
	if err != nil {
		t.Fatalf("RunIndexerOnce(): %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest == nil {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got nil", repo.latestNumber)
	}

	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got %d", repo.latestNumber, *result.DBLatest)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("RunIndexerOnce(): expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}

	wantNext := repo.latestNumber + 1
	if result.NextToSync != wantNext {
		t.Fatalf("RunIndexerOnce(): expected NextToSync=%d, got %d", wantNext, result.NextToSync)
	}

	if result.Synced {
		t.Fatalf("RunIndexerOnce(): expected Synced=false, got true")
	}

	if result.SyncedBlock != nil {
		t.Fatalf("RunIndexerOnce(): expected nil SyncedBlock, got %d", *result.SyncedBlock)
	}

	if service.syncCalled {
		t.Fatalf("RunIndexerOnce(): expected SyncBlockToDB not called")
	}
}

func TestBlockIndexer_RunIndexerOnce_ReturnsErrorWhenGetNextBlockFails(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)

	ctx := context.Background()

	repo.latestNumber = 1
	repo.exists = true
	rpc.err = types.ErrRPCTimeout

	result, err := svc.RunIndexerOnce(ctx)
	if err == nil {
		t.Fatalf("RunIndexerOnce(): expected error")
	}

	if result != nil {
		t.Fatalf("RunIndexerOnce(): expected nil, got %v", result)
	}

	if !errors.Is(err, types.ErrRPCTimeout) {
		t.Fatalf("RunIndexerOnce(): expected timeout error, got %v", err)
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC to be called after DB succeeds")
	}

	if service.syncCalled {
		t.Fatalf("expected SyncBlockToDB not called")
	}
}

func TestBlockIndexer_RunIndexerOnce_ReturnsErrorWhenSyncFails(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)

	ctx := context.Background()

	repo.latestNumber = 1
	repo.exists = true
	rpc.targetNumber = 2
	service.err = types.ErrDBTimeout

	result, err := svc.RunIndexerOnce(ctx)
	if err == nil {
		t.Fatalf("RunIndexerOnce(): expected error")
	}

	if result != nil {
		t.Fatalf("RunIndexerOnce(): expected nil, got %v", result)
	}

	if !errors.Is(err, types.ErrDBTimeout) {
		t.Fatalf("RunIndexerOnce(): expected timeout error, got %v", err)
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	wantNext := repo.latestNumber + 1
	if service.syncedBlock != wantNext {
		t.Fatalf("expected syncedBlock=%d, got %d", wantNext, service.syncedBlock)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC to be called before sync")
	}

	if !service.syncCalled {
		t.Fatalf("expected SyncBlockToDB called")
	}
}

func TestBlockIndexer_GetNextBlockToSync_UsesSyncTarget(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexerWithConfig(t, "finalized", 0)

	ctx := context.Background()

	repo.latestNumber = 10
	repo.exists = true
	rpc.targetNumber = 20

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC target to be called")
	}

	if rpc.gotTag != "finalized" {
		t.Fatalf("expected sync target finalized, got %s", rpc.gotTag)
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.SyncTarget != "finalized" {
		t.Fatalf("expected result SyncTarget finalized, got %s", result.SyncTarget)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}
}

func TestBlockIndexer_GetNextBlockToSync_UsesLatestFullySyncedBlockAsCursor(t *testing.T) {
	indexer, rpc, repo, _ := setupTestIndexer(t)

	ctx := context.Background()

	repo.latestNumber = 99
	repo.exists = true
	rpc.targetNumber = 105

	result, err := indexer.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if !repo.getLatestCalled {
		t.Fatalf("expected GetLatestFullySyncedBlockNumber to be called")
	}

	if repo.gotChainID != indexerTestChainID {
		t.Fatalf("expected repo chain_id=%d, got %d", indexerTestChainID, repo.gotChainID)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC target to be called")
	}

	if result.ChainID != indexerTestChainID {
		t.Fatalf("expected ChainID=%d, got %d", indexerTestChainID, result.ChainID)
	}

	if result.DBLatest == nil {
		t.Fatalf("expected DBLatest=%d, got nil", repo.latestNumber)
	}

	if *result.DBLatest != 99 {
		t.Fatalf("expected DBLatest=99, got %d", *result.DBLatest)
	}

	if result.RPCTarget != 105 {
		t.Fatalf("expected RPCTarget=105, got %d", result.RPCTarget)
	}

	if result.Next != 100 {
		t.Fatalf("expected Next=100, got %d", result.Next)
	}

	if !result.ShouldSync {
		t.Fatalf("expected ShouldSync=true")
	}
}

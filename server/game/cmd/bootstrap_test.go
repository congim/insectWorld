package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	gameevent "insectworld/server/game/interfaces/event"
	domainevent "insectworld/server/shared/pkg/eventbus"
	"insectworld/server/shared/pkg/gamepack"
)

// TestBuildRuntimeWiresRegistrationDelivery 验证生产装配包含Growth服务和注册事件发布器。
func TestBuildRuntimeWiresRegistrationDelivery(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	pack, err := gamepack.LoadAndCompile(filepath.Join("..", "..", "..", "gamepacks", "frontier-demo"), defaultEngineVersion)
	require.NoError(t, err)
	runtime, err := buildRuntime(context.Background(), db, pack, 2, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, runtime.Growth)
	require.NotNil(t, runtime.publisher)
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"event_id", "aggregate_id", "event_type", "event_version", "payload", "status", "retry_count", "create_time", "publish_time", "available_time"})
	mock.ExpectQuery("SELECT .* FROM t_outbox").WithArgs(gameevent.EventTypePlayerRegistered, domainevent.OutboxStatusPending, domainevent.OutboxStatusFailed, sqlmock.AnyArg(), domainevent.OutboxStatusProcessing, sqlmock.AnyArg(), defaultBatchSize).WillReturnRows(rows)
	mock.ExpectCommit()
	require.NoError(t, runtime.publisher.PollOnce(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestValidateStartupConfigRejectsMissingPack 验证游戏包必须由部署配置显式选择。
func TestValidateStartupConfigRejectsMissingPack(t *testing.T) {
	t.Parallel()
	err := validateStartupConfig(StartupConfig{MySQLDSN: "user:pass@tcp(localhost:3306)/game", EngineVersion: defaultEngineVersion, WorkerID: 2})
	require.Error(t, err)
}

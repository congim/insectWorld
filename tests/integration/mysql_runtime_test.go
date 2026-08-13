package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	economyapp "insectworld/server/economy/application/resourceaccount"
	economypersistence "insectworld/server/economy/infrastructure/persistence"
	gamecommand "insectworld/server/game/application/command"
	gamecatalog "insectworld/server/game/infrastructure/catalog"
	"insectworld/server/game/infrastructure/memory"
	gamepersistence "insectworld/server/game/infrastructure/persistence"
	gameevent "insectworld/server/game/interfaces/event"
	domainaccount "insectworld/server/gateway/domain/account"
	gatewayevent "insectworld/server/gateway/domain/event"
	accountpersistence "insectworld/server/gateway/infrastructure/persistence/account"
	sharedeventbus "insectworld/server/shared/infrastructure/eventbus"
	domainevent "insectworld/server/shared/pkg/eventbus"
	"insectworld/server/shared/pkg/eventbus/publisher"
	"insectworld/server/shared/pkg/gamepack"
	"insectworld/server/shared/schema/tables"
)

const mysqlIntegrationPlayerID = int64(900001)

// TestMySQLRegistrationGrowthFlow 使用真实MySQL验证注册、Outbox、资源、建造和训练事务链路。
func TestMySQLRegistrationGrowthFlow(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("未配置TEST_MYSQL_DSN，跳过真实MySQL集成测试")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))
	cleanMySQLGrowthData(t, ctx, db)

	pack, err := gamepack.LoadAndCompile(filepath.Join("..", "..", "gamepacks", "frontier-demo"), "0.1.0")
	require.NoError(t, err)
	catalogReader, err := gamecatalog.NewGamePackReader(pack)
	require.NoError(t, err)
	resourceService := economyapp.NewService(economypersistence.NewResourceAccountRepository(db, zap.NewNop()))
	roster := gamepersistence.NewUnitRoster(db, zap.NewNop())
	growth := gamecommand.NewService(
		gamepersistence.NewPlayerRepository(db, zap.NewNop()),
		gamepersistence.NewBuildingRepository(db, zap.NewNop()),
		gamepersistence.NewTrainingRepository(db, zap.NewNop()),
		roster,
		catalogReader,
		resourceService,
		memory.NewIDGenerator(1000),
		zap.NewNop(),
	)

	registeredAt := time.Now().UnixMilli()
	account := domainaccount.NewPlayerAccount(mysqlIntegrationPlayerID, "mysql-player", "bcrypt-hash", "", "127.0.0.1", registeredAt)
	registered, err := (gatewayevent.PlayerRegisteredEvent{PlayerID: mysqlIntegrationPlayerID, Username: account.Username(), RegisteredAt: registeredAt}).ToDomainEvent()
	require.NoError(t, err)
	require.NoError(t, accountpersistence.NewAccountRepoMySQL(db, zap.NewNop()).SaveRegistered(ctx, account, registered))

	localBus := sharedeventbus.NewLocalBus(zap.NewNop())
	require.NoError(t, localBus.Subscribe(ctx, gameevent.EventTypePlayerRegistered, gameevent.NewPlayerRegisteredHandler(growth).Handle))
	outboxRepository := sharedeventbus.NewOutboxRepository(db)
	delivery, err := publisher.New(outboxRepository, localBus, publisher.Config{
		EventTypes:     []string{gameevent.EventTypePlayerRegistered},
		BatchSize:      10,
		PollInterval:   time.Second,
		LeaseDuration:  time.Minute,
		BaseRetryDelay: time.Second,
		MaxRetryDelay:  time.Minute,
	}, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, delivery.PollOnce(ctx))

	balances, err := resourceService.Balances(ctx, mysqlIntegrationPlayerID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balances["supplies"])
	require.NoError(t, resetOutboxForDuplicateDelivery(ctx, db, registered.EventID))
	require.NoError(t, delivery.PollOnce(ctx))
	balances, err = resourceService.Balances(ctx, mysqlIntegrationPlayerID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balances["supplies"], "重复事件不得重复发放初始资源")

	building, err := growth.ConstructBuilding(ctx, gamecommand.ConstructBuildingCommand{CommandID: "mysql-build", PlayerID: mysqlIntegrationPlayerID, BuildingTypeID: "outpost", NowMs: registeredAt + 1})
	require.NoError(t, err)
	_, err = growth.CompleteBuilding(ctx, mysqlIntegrationPlayerID, building.ID(), building.CompleteAt(), "mysql-complete-building")
	require.NoError(t, err)
	training, err := growth.StartTraining(ctx, gamecommand.StartTrainingCommand{CommandID: "mysql-train", PlayerID: mysqlIntegrationPlayerID, BuildingID: building.ID(), UnitTypeID: "scout", Count: 2, NowMs: building.CompleteAt() + 1})
	require.NoError(t, err)
	_, err = growth.CompleteTraining(ctx, mysqlIntegrationPlayerID, training.ID(), training.CompleteAt(), "mysql-complete-training")
	require.NoError(t, err)
	unitCount, err := roster.Count(ctx, mysqlIntegrationPlayerID, "scout")
	require.NoError(t, err)
	assert.Equal(t, int64(2), unitCount)
	balances, err = resourceService.Balances(ctx, mysqlIntegrationPlayerID)
	require.NoError(t, err)
	assert.Equal(t, int64(55), balances["supplies"])
	assert.Equal(t, int64(2), delivery.Metrics().PublishedTotal)
}

func cleanMySQLGrowthData(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	tableNames := []string{tables.TUnitGrantOperation, tables.TUnitRoster, tables.TTrainingTask, tables.TPlayerBuilding, tables.TPlayerProfile, tables.TResourceOperation, tables.TResourceAccountBalance, tables.TOutbox, tables.TPlayerAccount}
	for _, tableName := range tableNames {
		_, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", tableName))
		require.NoError(t, err)
	}
}

func resetOutboxForDuplicateDelivery(ctx context.Context, db *sql.DB, eventID string) error {
	query := fmt.Sprintf("UPDATE %s SET status = ?, publish_time = 0, available_time = 0 WHERE event_id = ?", tables.TOutbox)
	_, err := db.ExecContext(ctx, query, domainevent.OutboxStatusPending, eventID)
	return err
}

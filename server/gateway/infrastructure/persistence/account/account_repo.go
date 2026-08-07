// Package account 账号仓储infrastructure层实现，提供MySQL适配。
//
// AccountRepoMySQL基于database/sql实现AccountRepository接口，
// 表名通过tables.TPlayerAccount常量引用（规范2），参数化查询防SQL注入。
package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"

	domainaccount "insectworld/server/gateway/domain/account"
	gatewayerr "insectworld/server/gateway/domain/errors"
	"insectworld/server/shared/schema/tables"
)

// AccountRepoMySQL MySQL账号仓储，实现AccountRepository接口。
type AccountRepoMySQL struct {
	db     *sql.DB     // MySQL数据库连接
	logger *zap.Logger // 结构化日志
}

// NewAccountRepoMySQL 创建MySQL账号仓储实例。
func NewAccountRepoMySQL(db *sql.DB, logger *zap.Logger) *AccountRepoMySQL {
	return &AccountRepoMySQL{
		db:     db,
		logger: logger,
	}
}

// Save 保存账号聚合根，INSERT或UPDATE到t_player_account表。
//
// 使用INSERT ... ON DUPLICATE KEY UPDATE实现UPSERT语义。
// 存储故障返回ErrAccountRepoUnavailable包裹底层error。
func (r *AccountRepoMySQL) Save(ctx context.Context, account *domainaccount.PlayerAccount) error {
	query := fmt.Sprintf(
		`INSERT INTO %s (player_id, username, password_hash, salt, status, ban_reason, ban_expire_time, register_time, register_ip)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE password_hash=VALUES(password_hash), salt=VALUES(salt), status=VALUES(status),
		 ban_reason=VALUES(ban_reason), ban_expire_time=VALUES(ban_expire_time)`,
		tables.TPlayerAccount,
	)
	_, err := r.db.ExecContext(ctx, query,
		account.PlayerID(), account.Username(), account.PasswordHash(), account.Salt(),
		account.Status(), account.BanReason(), account.BanExpireTime(),
		account.RegisterTime(), account.RegisterIP(),
	)
	if err != nil {
		r.logger.Error("账号持久化失败",
			zap.Int64("player_id", account.PlayerID()),
			zap.String("username", account.Username()),
			zap.Error(err),
		)
		return fmt.Errorf("账号持久化失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}
	return nil
}

// FindByID 按玩家ID查询账号档案。
//
// 账号不存在返回ErrAccountNotFoundSentinel（可用errors.Is判断）。
func (r *AccountRepoMySQL) FindByID(ctx context.Context, playerID int64) (*domainaccount.PlayerAccount, error) {
	query := fmt.Sprintf(
		`SELECT player_id, username, password_hash, salt, status, ban_reason, ban_expire_time, register_time, register_ip
		 FROM %s WHERE player_id = ?`,
		tables.TPlayerAccount,
	)
	return r.queryOne(ctx, query, playerID)
}

// FindByUsername 按用户名查询账号档案。
//
// 账号不存在返回ErrAccountNotFoundSentinel。
func (r *AccountRepoMySQL) FindByUsername(ctx context.Context, username string) (*domainaccount.PlayerAccount, error) {
	query := fmt.Sprintf(
		`SELECT player_id, username, password_hash, salt, status, ban_reason, ban_expire_time, register_time, register_ip
		 FROM %s WHERE username = ?`,
		tables.TPlayerAccount,
	)
	return r.queryOne(ctx, query, username)
}

// ExistsByUsername 判断用户名是否已存在，返回true表示已占用。
func (r *AccountRepoMySQL) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := fmt.Sprintf(`SELECT COUNT(1) FROM %s WHERE username = ?`, tables.TPlayerAccount)
	var count int
	if err := r.db.QueryRowContext(ctx, query, username).Scan(&count); err != nil {
		r.logger.Error("用户名存在性查询失败",
			zap.String("username", username),
			zap.Error(err),
		)
		return false, fmt.Errorf("用户名存在性查询失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}
	return count > 0, nil
}

// queryOne 执行查询并扫描单行结果为PlayerAccount聚合根。
func (r *AccountRepoMySQL) queryOne(ctx context.Context, query string, args ...interface{}) (*domainaccount.PlayerAccount, error) {
	var (
		playerID      int64
		username      string
		passwordHash  string
		salt          string
		status        int
		banReason     string
		banExpireTime int64
		registerTime  int64
		registerIP    string
	)
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&playerID, &username, &passwordHash, &salt, &status, &banReason, &banExpireTime, &registerTime, &registerIP,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gatewayerr.ErrAccountNotFoundSentinel
		}
		r.logger.Error("账号查询失败", zap.Error(err))
		return nil, fmt.Errorf("账号查询失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}

	account := domainaccount.NewPlayerAccount(playerID, username, passwordHash, salt, registerIP, registerTime)
	if status == domainaccount.AccountStatusBanned {
		_ = account.Ban(banReason, banExpireTime)
	}
	return account, nil
}

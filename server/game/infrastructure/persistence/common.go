// Package persistence 提供Growth上下文MySQL持久化适配器。
package persistence

import (
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	gameerr "insectworld/server/game/domain/errors"
)

const mysqlDuplicateEntryCode uint16 = 1062

type scanner interface {
	Scan(dest ...any) error
}

func isDuplicateEntry(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == mysqlDuplicateEntryCode
}

func unavailable(operation string, err error) error {
	return fmt.Errorf("%s失败: %v: %w", operation, err, gameerr.ErrRepositoryUnavailable)
}

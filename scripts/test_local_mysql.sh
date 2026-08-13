#!/bin/bash
# 重建本地MySQL测试库并运行真实仓储集成测试。

set -euo pipefail

repository_root=$(cd "$(dirname "$0")/.." && pwd)
mysql_host="${MYSQL_HOST:-127.0.0.1}"
mysql_port="${MYSQL_PORT:-3306}"
mysql_user="${MYSQL_USER:-root}"
mysql_password="${MYSQL_PASSWORD:-123456}"
test_database="${MYSQL_TEST_DATABASE:-insect_world_test}"

"$repository_root/scripts/reset_local_mysql.sh"

test_dsn="$mysql_user:$mysql_password@tcp($mysql_host:$mysql_port)/$test_database?parseTime=true&charset=utf8mb4"
(cd "$repository_root/tests/integration" && TEST_MYSQL_DSN="$test_dsn" go test -race -count=1 -run TestMySQLRegistrationGrowthFlow ./...)

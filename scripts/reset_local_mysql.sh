#!/bin/bash
# 重建本地隔离MySQL测试库，并按顺序执行开发期数据库基线。

set -euo pipefail

repository_root=$(cd "$(dirname "$0")/.." && pwd)
schema_root="$repository_root/server/shared/schema/baseline"
mysql_host="${MYSQL_HOST:-127.0.0.1}"
mysql_port="${MYSQL_PORT:-3306}"
mysql_user="${MYSQL_USER:-root}"
mysql_password="${MYSQL_PASSWORD:-123456}"
test_database="${MYSQL_TEST_DATABASE:-insect_world_test}"

if [[ ! "$test_database" =~ ^[a-z0-9_]+_test$ ]]; then
    echo "拒绝重建非测试数据库：$test_database" >&2
    exit 1
fi

export MYSQL_PWD="$mysql_password"
mysql_args=(--protocol=TCP --host="$mysql_host" --port="$mysql_port" --user="$mysql_user" --default-character-set=utf8mb4)

mysql "${mysql_args[@]}" --execute="DROP DATABASE IF EXISTS \`$test_database\`; CREATE DATABASE \`$test_database\` CHARACTER SET utf8mb4;"

for schema_file in "$schema_root"/*.sql; do
    mysql "${mysql_args[@]}" "$test_database" < "$schema_file"
done

table_count=$(mysql "${mysql_args[@]}" --batch --skip-column-names --execute="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$test_database';")
unset MYSQL_PWD

echo "本地MySQL测试库重建完成：database=$test_database tables=$table_count"

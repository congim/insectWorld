// Package persistence Config服务持久化层。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

const (
	TableConfigVersion  = tables.TConfigVersion  // 配置版本表
	TableConfigSnapshot = tables.TConfigSnapshot // 配置快照表
)

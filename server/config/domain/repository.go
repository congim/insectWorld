// Package domain Config服务domain层，声明Repository接口与领域值对象。
//
// domain层零外部依赖（规范3），Repository接口在此声明，
// infrastructure层实现，application层通过接口依赖，cmd/main.go组装注入。
package domain

import "context"

// ConfigStorage 配置存储接口，infrastructure层etcd实现。
// 提供配置包的读写与变更通知能力，Put时产生配置变更事件供热更watcher消费。
type ConfigStorage interface {
	// Put 写入配置包并通知热更watcher。
	// key为配置包ID，value为配置包二进制数据，configVersion为版本号，configPack为解析后的配置数据。
	Put(ctx context.Context, key string, value []byte, configVersion int64, configPack map[string]any) error
	// Get 读取配置包二进制数据。
	Get(ctx context.Context, key string) ([]byte, error)
}

// VersionInfo 配置版本信息值对象。
type VersionInfo struct {
	VersionID  int64  // 版本ID，全局唯一，用时间戳生成
	Version    string // 版本号字符串，对应配置包ID
	ConfigType int    // 配置类型：1=全量配置包 2=增量配置包
	Operator   string // 操作人，运营管理面用户标识
	CreateTime int64  // 创建时间戳，int64毫秒级（规范8）
}

// VersionStorage 版本存储接口，infrastructure层versionstore实现。
// 提供配置版本的持久化与历史查询能力，支持10版本回滚。
type VersionStorage interface {
	// SaveVersion 保存配置版本记录。
	SaveVersion(ctx context.Context, versionID int64, version string, configType int, operator string) error
	// FindVersions 查询指定配置类型的版本历史，按创建时间降序返回。
	FindVersions(ctx context.Context, configType int, limit int) ([]VersionInfo, error)
}

// 审计操作类型枚举常量。
// 取值映射：1=创建 2=发布 3=回滚 4=删除
const (
	AuditActionCreate   = 1 // 创建配置版本
	AuditActionPublish  = 2 // 发布配置版本
	AuditActionRollback = 3 // 回滚配置版本
	AuditActionDelete   = 4 // 删除配置版本
)

// AuditRecord 审计日志记录值对象。
type AuditRecord struct {
	VersionID   int64  // 版本ID，关联配置版本
	Operator    string // 操作人，运营管理面用户标识
	Action      int    // 操作类型：1=创建 2=发布 3=回滚 4=删除
	BeforeValue string // 操作前值，JSON序列化的配置快照
	AfterValue  string // 操作后值，JSON序列化的配置快照
}

// AuditStorage 审计存储接口，infrastructure层audit实现。
// 审计日志独立存储，含操作人/操作时间/操作内容/操作结果/操作前后值（规范7）。
type AuditStorage interface {
	// Save 保存审计日志记录。
	Save(ctx context.Context, record AuditRecord) error
}

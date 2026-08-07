// Package audit 认证审计日志能力接口与记录结构，domain层声明，infrastructure层实现MySQL适配。
//
// domain层零外部依赖（规范3），审计日志独立于AdminHandler的运营审计日志（spec 4.4 可维护性2）。
package audit

import "context"

// 审计操作类型枚举常量，表示认证审计日志的操作类型。
// 取值映射：1=注册成功 2=登录成功 3=登录失败 4=登出 5=封禁拦截 6=暴力破解锁定
const (
	OpTypeRegisterSuccess = 1 // 注册成功
	OpTypeLoginSuccess    = 2 // 登录成功
	OpTypeLoginFailure    = 3 // 登录失败
	OpTypeLogout          = 4 // 登出
	OpTypeBanIntercept    = 5 // 封禁拦截
	OpTypeBruteForceLock  = 6 // 暴力破解锁定
)

// AuditRecord 审计日志记录，封装一次认证操作的审计信息。
//
// 所有时间戳用int64毫秒（规范8），OpType用int枚举（规范8）。
type AuditRecord struct {
	OpType   int    // 操作类型：1=注册成功 2=登录成功 3=登录失败 4=登出 5=封禁拦截 6=暴力破解锁定
	Subject  string // 操作主体，如用户名或玩家ID字符串
	Result   bool   // 操作结果：true=成功 false=失败
	SourceIP string // 来源IP
	OpTime   int64  // 操作时间戳，毫秒级
	Extra    string // 附加信息，JSON格式扩展字段
}

// AuditLogger 审计日志能力接口，infrastructure层实现MySQL异步落库适配。
//
// 接口在domain层声明（规范3 DDD），异步落库不阻塞主流程（design.md 2.2.2.7节）。
type AuditLogger interface {
	// LogRecord 异步记录审计日志到t_auth_audit_log表。
	// MySQL故障时记录Error日志但不返回错误（审计日志失败不阻塞主流程）。
	LogRecord(ctx context.Context, record *AuditRecord) error
}

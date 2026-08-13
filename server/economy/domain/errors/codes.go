// Package errors Economy服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 12000-12999 为 Economy 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），区间12000-12999。
const (
	ErrCodeInvalidParams         = 12001 // 参数非法
	ErrCodeRuleViolation         = 12002 // 规则违反
	ErrCodeResourceInsufficient  = 12003 // 资源不足
	ErrCodeStorageOverflow       = 12004 // 存储溢出
	ErrCodeTradeNotAllowed       = 12005 // 交易不允许
	ErrCodeConversionInvalid     = 12006 // 转换非法
	ErrCodeOperationConflict     = 12007 // 幂等操作载荷冲突
	ErrCodeRepositoryUnavailable = 12008 // 资源账户持久化不可用
)

// EconomyError Economy服务错误，携带错误码。
type EconomyError struct {
	Code int    // 错误码
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *EconomyError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义（规范9）。
var (
	ErrInvalidParams         = &EconomyError{ErrCodeInvalidParams, "参数非法"}
	ErrRuleViolation         = &EconomyError{ErrCodeRuleViolation, "规则违反"}
	ErrResourceInsufficient  = &EconomyError{ErrCodeResourceInsufficient, "资源不足"}
	ErrStorageOverflow       = &EconomyError{ErrCodeStorageOverflow, "存储溢出"}
	ErrTradeNotAllowed       = &EconomyError{ErrCodeTradeNotAllowed, "交易不允许"}
	ErrConversionInvalid     = &EconomyError{ErrCodeConversionInvalid, "转换非法"}
	ErrOperationConflict     = &EconomyError{ErrCodeOperationConflict, "资源操作冲突"}
	ErrRepositoryUnavailable = &EconomyError{ErrCodeRepositoryUnavailable, "资源账户暂不可用"}
)

// Package errors Match服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 17000-17999 为 Match 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），使用显式整型赋值，区间17000-17999。
const (
	// ErrCodeInvalidParams 参数非法
	ErrCodeInvalidParams = 17001
	// ErrCodeMatchTimeout 匹配超时
	ErrCodeMatchTimeout = 17002
	// ErrCodeBattlefieldNotFound 战场不存在
	ErrCodeBattlefieldNotFound = 17003
	// ErrCodeBattlefieldFull 战场已满
	ErrCodeBattlefieldFull = 17004
	// ErrCodeBattlefieldEnded 战场已结束
	ErrCodeBattlefieldEnded = 17005
)

// MatchError Match服务错误，携带错误码便于业务层判断。
type MatchError struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *MatchError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义。
var (
	// ErrInvalidParams 参数非法错误
	ErrInvalidParams = &MatchError{ErrCodeInvalidParams, "参数非法"}
	// ErrMatchTimeout 匹配超时错误
	ErrMatchTimeout = &MatchError{ErrCodeMatchTimeout, "匹配超时"}
	// ErrBattlefieldNotFound 战场不存在错误
	ErrBattlefieldNotFound = &MatchError{ErrCodeBattlefieldNotFound, "战场不存在"}
	// ErrBattlefieldFull 战场已满错误
	ErrBattlefieldFull = &MatchError{ErrCodeBattlefieldFull, "战场已满"}
	// ErrBattlefieldEnded 战场已结束错误
	ErrBattlefieldEnded = &MatchError{ErrCodeBattlefieldEnded, "战场已结束"}
)

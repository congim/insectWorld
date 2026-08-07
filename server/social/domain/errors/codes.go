// Package errors Social服务错误码定义。
// 错误码区间 13000-13999 为 Social 服务保留。
package errors

import "fmt"

const (
	ErrCodeInvalidParams     = 13001 // 参数非法
	ErrCodeRuleViolation     = 13002 // 规则违反
	ErrCodeAllianceFull      = 13003 // 联盟已满
	ErrCodeJoinCooldown      = 13004 // 加入冷却中
	ErrCodePermissionDenied  = 13005 // 权限不足
	ErrCodeDiplomacyConflict = 13006 // 外交冲突
)

// SocialError Social服务错误。
type SocialError struct {
	Code int    // 错误码
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *SocialError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

var (
	ErrInvalidParams     = &SocialError{ErrCodeInvalidParams, "参数非法"}
	ErrRuleViolation     = &SocialError{ErrCodeRuleViolation, "规则违反"}
	ErrAllianceFull      = &SocialError{ErrCodeAllianceFull, "联盟已满"}
	ErrJoinCooldown      = &SocialError{ErrCodeJoinCooldown, "加入冷却中"}
	ErrPermissionDenied  = &SocialError{ErrCodePermissionDenied, "权限不足"}
	ErrDiplomacyConflict = &SocialError{ErrCodeDiplomacyConflict, "外交冲突"}
)

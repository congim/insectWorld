// Package errors Combat服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 11000-11999 为 Combat 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），区间11000-11999。
const (
	// ErrCodeInvalidParams 参数非法
	ErrCodeInvalidParams = 11001
	// ErrCodeRuleViolation 规则违反
	ErrCodeRuleViolation = 11002
	// ErrCodeMaxRoundsExceeded 轮数超限，战斗轮次超过最大轮数
	ErrCodeMaxRoundsExceeded = 11003
	// ErrCodeSkillCooldown 技能冷却中
	ErrCodeSkillCooldown = 11004
	// ErrCodeCombatNotFound 战斗不存在
	ErrCodeCombatNotFound = 11005
)

// CombatError Combat服务错误，携带错误码。
type CombatError struct {
	Code int    // 错误码
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *CombatError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义（规范9）。
var (
	// ErrInvalidParams 参数非法错误
	ErrInvalidParams = &CombatError{ErrCodeInvalidParams, "参数非法"}
	// ErrRuleViolation 规则违反错误
	ErrRuleViolation = &CombatError{ErrCodeRuleViolation, "规则违反"}
	// ErrMaxRoundsExceeded 轮数超限错误
	ErrMaxRoundsExceeded = &CombatError{ErrCodeMaxRoundsExceeded, "轮数超限"}
	// ErrSkillCooldown 技能冷却中错误
	ErrSkillCooldown = &CombatError{ErrCodeSkillCooldown, "技能冷却中"}
	// ErrCombatNotFound 战斗不存在错误
	ErrCombatNotFound = &CombatError{ErrCodeCombatNotFound, "战斗不存在"}
)

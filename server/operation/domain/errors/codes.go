// Package errors Operation服务错误码定义。
// 错误码区间 16000-16999 为 Operation 服务保留（避免与共享内核config 14000-14999冲突）。
package errors

import "fmt"

const (
	ErrCodeInvalidParams            = 16001 // 参数非法
	ErrCodeRuleViolation            = 16002 // 规则违反
	ErrCodeSeasonNotFound           = 16003 // 赛季不存在
	ErrCodePhaseTransitionInvalid   = 16004 // 阶段切换非法
	ErrCodeRewardDistributionFailed = 16005 // 奖励发放失败
)

// OperationError Operation服务错误。
type OperationError struct {
	Code int    // 错误码
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *OperationError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

var (
	ErrInvalidParams            = &OperationError{ErrCodeInvalidParams, "参数非法"}
	ErrRuleViolation            = &OperationError{ErrCodeRuleViolation, "规则违反"}
	ErrSeasonNotFound           = &OperationError{ErrCodeSeasonNotFound, "赛季不存在"}
	ErrPhaseTransitionInvalid   = &OperationError{ErrCodePhaseTransitionInvalid, "阶段切换非法"}
	ErrRewardDistributionFailed = &OperationError{ErrCodeRewardDistributionFailed, "奖励发放失败"}
)

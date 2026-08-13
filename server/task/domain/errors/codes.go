// Package errors Task服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 16000-16999 为 Task 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），使用显式整型赋值，区间16000-16999。
const (
	// ErrCodeInvalidParams 参数非法，请求参数校验未通过
	ErrCodeInvalidParams = 16001
	// ErrCodeTaskNotFound 任务不存在
	ErrCodeTaskNotFound = 16002
	// ErrCodeTaskNotCompleted 任务未完成，不可领取奖励
	ErrCodeTaskNotCompleted = 16003
	// ErrCodeTaskAlreadyClaimed 任务奖励已领取
	ErrCodeTaskAlreadyClaimed = 16004
	// ErrCodeAchievementNotFound 成就不存在
	ErrCodeAchievementNotFound = 16005
	// ErrCodeAchievementAlreadyUnlocked 成就已解锁
	ErrCodeAchievementAlreadyUnlocked = 16006
)

// TaskError Task服务错误，携带错误码便于业务层判断。
type TaskError struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *TaskError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义，业务代码通过errors.Is判断错误类型（规范9）。
var (
	// ErrInvalidParams 参数非法错误
	ErrInvalidParams = &TaskError{ErrCodeInvalidParams, "参数非法"}
	// ErrTaskNotFound 任务不存在错误
	ErrTaskNotFound = &TaskError{ErrCodeTaskNotFound, "任务不存在"}
	// ErrTaskNotCompleted 任务未完成错误
	ErrTaskNotCompleted = &TaskError{ErrCodeTaskNotCompleted, "任务未完成"}
	// ErrTaskAlreadyClaimed 任务奖励已领取错误
	ErrTaskAlreadyClaimed = &TaskError{ErrCodeTaskAlreadyClaimed, "任务奖励已领取"}
	// ErrAchievementNotFound 成就不存在错误
	ErrAchievementNotFound = &TaskError{ErrCodeAchievementNotFound, "成就不存在"}
	// ErrAchievementAlreadyUnlocked 成就已解锁错误
	ErrAchievementAlreadyUnlocked = &TaskError{ErrCodeAchievementAlreadyUnlocked, "成就已解锁"}
)

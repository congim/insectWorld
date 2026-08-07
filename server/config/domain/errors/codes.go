// Package errors Config服务错误码定义。
// 错误码区间 15000-15999 为 Config 服务保留。
package errors

import "fmt"

const (
	ErrCodeConfigInvalid          = 15001 // 配置非法
	ErrCodeSchemaValidationFailed = 15002 // Schema校验失败
	ErrCodeVersionConflict        = 15003 // 版本冲突
	ErrCodeHotReloadInProgress    = 15004 // 热更进行中
)

// ConfigError Config服务错误。
type ConfigError struct {
	Code int    // 错误码
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *ConfigError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

var (
	ErrConfigInvalid          = &ConfigError{ErrCodeConfigInvalid, "配置非法"}
	ErrSchemaValidationFailed = &ConfigError{ErrCodeSchemaValidationFailed, "Schema校验失败"}
	ErrVersionConflict        = &ConfigError{ErrCodeVersionConflict, "版本冲突"}
	ErrHotReloadInProgress    = &ConfigError{ErrCodeHotReloadInProgress, "热更进行中"}
)

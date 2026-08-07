// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义共享内核config模块的错误码常量与错误变量，集中定义便于统一管理与排查。
package config

import "fmt"

// 错误码区间 14000-14999 为共享内核 config 模块保留。
// 各业务服务的错误码在 internal/<service>/errors/codes.go 定义，使用各自区间。
const (
	// ErrCodeExtensionPointNotFound 扩展点未注册，查询的扩展点ID在注册表中不存在
	ErrCodeExtensionPointNotFound = 14001
	// ErrCodeConfigInvalid 配置内容非法，校验未通过
	ErrCodeConfigInvalid = 14002
	// ErrCodeSchemaValidationFailed JSON Schema 校验失败，配置结构不符合定义
	ErrCodeSchemaValidationFailed = 14003
	// ErrCodeRefIntegrityFailed 引用完整性校验失败，配置间引用断裂
	ErrCodeRefIntegrityFailed = 14004
	// ErrCodeCustomRuleFailed 自定义业务规则校验失败
	ErrCodeCustomRuleFailed = 14005
	// ErrCodePackIncomplete 配置包完整性校验失败，必需文件缺失或依赖不满足
	ErrCodePackIncomplete = 14006
	// ErrCodePackIncompatible 配置包兼容性校验失败，与存量数据Schema不兼容
	ErrCodePackIncompatible = 14007
	// ErrCodeVersionConflict 配置版本冲突，热更时版本号非递增
	ErrCodeVersionConflict = 14008
	// ErrCodeHotReloadInProgress 热更进行中，拒绝并发热更请求
	ErrCodeHotReloadInProgress = 14009
)

// ConfigError 配置模块错误，携带错误码便于业务层判断与统一错误处理。
type ConfigError struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 错误消息，中文文案
}

// Error 实现error接口，输出格式为 [错误码] 错误消息。
func (e *ConfigError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义，业务代码通过errors.Is判断错误类型，通过fmt.Errorf包裹上下文（规范9）。
var (
	// ErrExtensionPointNotFound 扩展点未注册错误
	ErrExtensionPointNotFound = &ConfigError{ErrCodeExtensionPointNotFound, "扩展点未注册"}
	// ErrConfigInvalid 配置内容非法错误
	ErrConfigInvalid = &ConfigError{ErrCodeConfigInvalid, "配置内容非法"}
	// ErrSchemaValidationFailed JSON Schema校验失败错误
	ErrSchemaValidationFailed = &ConfigError{ErrCodeSchemaValidationFailed, "JSON Schema校验失败"}
	// ErrRefIntegrityFailed 引用完整性校验失败错误
	ErrRefIntegrityFailed = &ConfigError{ErrCodeRefIntegrityFailed, "引用完整性校验失败"}
	// ErrCustomRuleFailed 自定义业务规则校验失败错误
	ErrCustomRuleFailed = &ConfigError{ErrCodeCustomRuleFailed, "自定义业务规则校验失败"}
	// ErrPackIncomplete 配置包完整性校验失败错误
	ErrPackIncomplete = &ConfigError{ErrCodePackIncomplete, "配置包完整性校验失败"}
	// ErrPackIncompatible 配置包兼容性校验失败错误
	ErrPackIncompatible = &ConfigError{ErrCodePackIncompatible, "配置包兼容性校验失败"}
	// ErrVersionConflict 配置版本冲突错误
	ErrVersionConflict = &ConfigError{ErrCodeVersionConflict, "配置版本冲突"}
	// ErrHotReloadInProgress 热更进行中错误
	ErrHotReloadInProgress = &ConfigError{ErrCodeHotReloadInProgress, "热更进行中"}
)

// ErrMsg 返回错误码对应的中文消息。
func ErrMsg(code int) string {
	switch code {
	case ErrCodeExtensionPointNotFound:
		return ErrExtensionPointNotFound.Msg
	case ErrCodeConfigInvalid:
		return ErrConfigInvalid.Msg
	case ErrCodeSchemaValidationFailed:
		return ErrSchemaValidationFailed.Msg
	case ErrCodeRefIntegrityFailed:
		return ErrRefIntegrityFailed.Msg
	case ErrCodeCustomRuleFailed:
		return ErrCustomRuleFailed.Msg
	case ErrCodePackIncomplete:
		return ErrPackIncomplete.Msg
	case ErrCodePackIncompatible:
		return ErrPackIncompatible.Msg
	case ErrCodeVersionConflict:
		return ErrVersionConflict.Msg
	case ErrCodeHotReloadInProgress:
		return ErrHotReloadInProgress.Msg
	default:
		return "未知错误"
	}
}

// Package logutil 测试端日志工具，封装zap初始化保持与Gateway日志风格一致。
package logutil

import "go.uber.org/zap"

// NewLogger 创建zap生产配置的logger，与Gateway日志风格一致。
//
// 开发环境可切换NewDevelopment获取更可读的日志<output>。
func NewLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

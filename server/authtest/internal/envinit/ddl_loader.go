// Package envinit 测试端环境初始化工具，负责建库/建表/清理/销毁测试数据库。
package envinit

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

// DDLLoader DDL脚本加载器，读取并解析DDL文件为独立SQL语句。
type DDLLoader struct {
	logger *zap.Logger // 结构化日志
}

// NewDDLLoader 创建DDL加载器实例。
func NewDDLLoader(logger *zap.Logger) *DDLLoader {
	return &DDLLoader{logger: logger}
}

// Load 读取DDL文件并按分号拆分为独立SQL语句切片。
//
// 过滤注释行（以--开头）与空白行，返回有效SQL语句。
func (l *DDLLoader) Load(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("DDL脚本加载失败: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	joined := strings.Join(filteredLines, "\n")
	statements := strings.Split(joined, ";")

	var result []string
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	l.logger.Info("DDL脚本加载成功",
		zap.String("file", filePath),
		zap.Int("statement_count", len(result)),
	)
	return result, nil
}

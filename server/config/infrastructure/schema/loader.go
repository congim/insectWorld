// Package schema Config服务JSON Schema校验引擎适配，加载与校验配置Schema。
//
// infrastructure层技术适配，实现domain层SchemaLoader接口。
package schema

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Loader JSON Schema加载与校验器。
type Loader struct {
	schemas map[string][]byte // Schema缓存，key=配置类型
	mu      sync.RWMutex      // 读写锁，保护Schema缓存并发访问
	logger  *zap.Logger       // 结构化日志
}

// NewLoader 创建Schema加载器实例。
func NewLoader(logger *zap.Logger) *Loader {
	return &Loader{
		schemas: make(map[string][]byte),
		logger:  logger,
	}
}

// LoadSchema 加载指定配置类型的JSON Schema。
func (l *Loader) LoadSchema(ctx context.Context, configType string, schemaContent []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.schemas[configType] = schemaContent
	l.logger.Info("JSON Schema加载成功", zap.String("config_type", configType))
	return nil
}

// Validate 校验配置数据是否符合指定类型的Schema。
// TODO 后续接入gojsonschema校验引擎，当前为占位实现。
func (l *Loader) Validate(ctx context.Context, configType string, configData []byte) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if _, ok := l.schemas[configType]; !ok {
		return fmt.Errorf("配置类型 %s 的Schema未加载", configType)
	}
	_ = configData
	return nil
}

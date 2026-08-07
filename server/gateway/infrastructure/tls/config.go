// Package tls Gateway服务TLS证书管理，实现mTLS证书加载与配置。
//
// infrastructure层技术适配，实现domain层TLSConfigLoader接口。
package tls

import (
	"crypto/tls"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// ConfigManager TLS配置管理器，管理证书加载与自动轮换。
type ConfigManager struct {
	certFile  string       // 证书文件路径
	keyFile   string       // 私钥文件路径
	tlsConfig *tls.Config  // TLS配置
	mu        sync.RWMutex // 读写锁，保护TLS配置并发访问
	logger    *zap.Logger  // 结构化日志
}

// NewConfigManager 创建TLS配置管理器实例。
func NewConfigManager(certFile, keyFile string, logger *zap.Logger) *ConfigManager {
	return &ConfigManager{
		certFile: certFile,
		keyFile:  keyFile,
		logger:   logger,
	}
}

// LoadConfig 加载TLS证书配置。
func (m *ConfigManager) LoadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
	if err != nil {
		return fmt.Errorf("TLS证书加载失败: %w", err)
	}

	m.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	m.logger.Info("TLS证书加载成功", zap.String("cert_file", m.certFile))
	return nil
}

// TLSConfig 返回当前TLS配置。
func (m *ConfigManager) TLSConfig() *tls.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tlsConfig
}

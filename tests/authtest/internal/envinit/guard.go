// Package envinit 测试端环境初始化工具，负责建库/建表/清理/销毁测试数据库。
package envinit

import "errors"

// LocalMySQLGuard 本地MySQL连接守卫，防止误连远程数据库。
type LocalMySQLGuard struct{}

// NewLocalMySQLGuard 创建本地MySQL守卫实例。
func NewLocalMySQLGuard() *LocalMySQLGuard {
	return &LocalMySQLGuard{}
}

// Validate 校验host为127.0.0.1或localhost，拒绝远程连接防止误操作生产数据。
func (g *LocalMySQLGuard) Validate(host string) error {
	if host != "127.0.0.1" && host != "localhost" {
		return errors.New("仅允许连接本地MySQL（127.0.0.1或localhost），拒绝远程连接防止误操作生产数据")
	}
	return nil
}

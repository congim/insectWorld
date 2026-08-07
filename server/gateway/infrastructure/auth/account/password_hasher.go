// Package account 密码哈希infrastructure层实现，提供bcrypt适配。
//
// BcryptHasher使用bcrypt算法生成密码哈希，密码不明文存储（spec 4.3 安全性1），
// 单向不可逆哈希，每个账号独立盐（bcrypt自带盐，salt字段存空字符串）。
package account

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// BcryptHasher bcrypt密码哈希器，实现PasswordHasher接口。
type BcryptHasher struct {
	cost   int         // bcrypt计算成本，默认10
	logger *zap.Logger // 结构化日志
}

// NewBcryptHasher 创建bcrypt密码哈希器实例。
//
// cost为bcrypt计算成本，范围[bcrypt.MinCost, bcrypt.MaxCost]，默认10。
func NewBcryptHasher(cost int, logger *zap.Logger) *BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = 10
	}
	return &BcryptHasher{
		cost:   cost,
		logger: logger,
	}
}

// Hash 对明文密码生成bcrypt哈希。
//
// bcrypt自带盐，返回的salt为空字符串（盐内嵌于哈希值）。
// 哈希计算失败返回ErrPasswordHashFailed。
func (h *BcryptHasher) Hash(ctx context.Context, password string) (hash string, salt string, err error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		h.logger.Error("密码哈希生成失败", zap.Error(err))
		return "", "", fmt.Errorf("密码哈希生成失败: %w", gatewayerr.ErrPasswordHashFailed)
	}
	return string(hashBytes), "", nil
}

// Verify 校验明文密码与bcrypt哈希是否匹配。
//
// 匹配返回true，不匹配返回false，哈希计算异常返回error。
func (h *BcryptHasher) Verify(ctx context.Context, password string, hash string, salt string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil
	}
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	return false, fmt.Errorf("密码校验异常: %w", err)
}

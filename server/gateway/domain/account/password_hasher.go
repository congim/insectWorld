// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
//
// domain层零外部依赖（规范3），PasswordHasher等能力接口在本包声明，
// infrastructure层实现bcrypt适配。聚合根字段私有，状态变更通过方法（DDD一致性边界）。
package account

import "context"

// PasswordHasher 密码哈希能力接口，infrastructure层实现bcrypt适配。
//
// 接口在domain层声明（规范3 DDD），保证domain层不依赖第三方bcrypt包。
// 密码不明文存储（spec 4.3 安全性1），单向不可逆哈希，每个账号独立盐。
type PasswordHasher interface {
	// Hash 对明文密码加盐哈希，返回哈希值与盐。
	// 哈希计算失败返回ErrPasswordHashFailed。
	Hash(ctx context.Context, password string) (hash string, salt string, err error)

	// Verify 校验明文密码与哈希值是否匹配。
	// 匹配返回true，不匹配返回false，哈希计算异常返回error。
	Verify(ctx context.Context, password string, hash string, salt string) (bool, error)
}

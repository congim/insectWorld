// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
package account

// Credential 凭证值对象，封装用户名与密码的传输与校验。
//
// 值对象语义：创建后不可变字段通过方法访问，ZeroPassword用于处理完立即置零密码字段，
// 防止内存残留（spec 4.3 安全性2，密码不明文驻留内存）。
type Credential struct {
	username string // 用户名
	password string // 密码明文，仅在创建到哈希前短暂存在，处理完应立即ZeroPassword
}

// NewCredential 创建凭证值对象实例。
func NewCredential(username, password string) *Credential {
	return &Credential{
		username: username,
		password: password,
	}
}

// Username 返回用户名。
func (c *Credential) Username() string { return c.username }

// Password 返回密码明文，仅在哈希前调用，调用后应立即ZeroPassword。
func (c *Credential) Password() string { return c.password }

// ZeroPassword 置零密码字段，防止内存残留。
//
// 密码哈希处理完成后应立即调用，避免明文密码长期驻留内存（spec 4.3 安全性2）。
func (c *Credential) ZeroPassword() {
	c.password = ""
}

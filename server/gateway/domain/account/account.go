// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
package account

import (
	"context"
	"fmt"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// 账号状态枚举常量，表示账号档案的当前状态。
// 取值映射：1=正常 2=封禁
const (
	AccountStatusNormal = 1 // 正常状态，可登录可操作
	AccountStatusBanned = 2 // 封禁状态，登录被拒绝
)

// PlayerAccount 玩家账号聚合根，维护账号档案的一致性边界。
//
// 聚合根字段私有，外部通过方法访问与变更（DDD聚合根一致性边界）。
// 状态变更（封禁/解封）通过方法保证状态机流转合法性。
// 所有ID与时间戳用int64（规范8），状态用int枚举（规范8）。
type PlayerAccount struct {
	playerID      int64  // 玩家ID，雪花算法生成，全局唯一
	username      string // 用户名，注册时填写，唯一
	passwordHash  string // 密码哈希值，bcrypt生成，不明文存储
	salt          string // 密码盐，bcrypt自带盐时存空字符串
	status        int    // 账号状态：1=正常 2=封禁
	banReason     string // 封禁原因，未封禁时为空
	banExpireTime int64  // 封禁过期时间戳，毫秒级，0=永久封禁
	registerTime  int64  // 注册时间戳，毫秒级
	registerIP    string // 注册时来源IP
}

// NewPlayerAccount 创建玩家账号聚合根实例，初始状态为正常。
//
// 构造函数仅赋值不校验，校验逻辑由ValidateUsernameFormat/ValidatePasswordStrength
// 在application层调用后传入。registerTime为int64毫秒时间戳（规范8）。
func NewPlayerAccount(playerID int64, username, passwordHash, salt, registerIP string, registerTime int64) *PlayerAccount {
	return &PlayerAccount{
		playerID:      playerID,
		username:      username,
		passwordHash:  passwordHash,
		salt:          salt,
		status:        AccountStatusNormal,
		banReason:     "",
		banExpireTime: 0,
		registerTime:  registerTime,
		registerIP:    registerIP,
	}
}

// PlayerID 返回玩家ID。
func (a *PlayerAccount) PlayerID() int64 { return a.playerID }

// Username 返回用户名。
func (a *PlayerAccount) Username() string { return a.username }

// PasswordHash 返回密码哈希值，供仓储持久化使用。
func (a *PlayerAccount) PasswordHash() string { return a.passwordHash }

// Salt 返回密码盐，供仓储持久化使用。
func (a *PlayerAccount) Salt() string { return a.salt }

// Status 返回账号状态：1=正常 2=封禁。
func (a *PlayerAccount) Status() int { return a.status }

// BanReason 返回封禁原因，未封禁时为空。
func (a *PlayerAccount) BanReason() string { return a.banReason }

// BanExpireTime 返回封禁过期时间戳，毫秒级，0=永久封禁。
func (a *PlayerAccount) BanExpireTime() int64 { return a.banExpireTime }

// RegisterTime 返回注册时间戳，毫秒级。
func (a *PlayerAccount) RegisterTime() int64 { return a.registerTime }

// RegisterIP 返回注册时来源IP。
func (a *PlayerAccount) RegisterIP() string { return a.registerIP }

// IsBanned 判断账号是否处于封禁状态且未过期。
//
// now为当前时间戳，毫秒级。永久封禁（banExpireTime=0）始终返回true。
// 临时封禁过期后视为未封禁，但状态字段不自动更新（需调用Unban显式解封）。
func (a *PlayerAccount) IsBanned(now int64) bool {
	if a.status != AccountStatusBanned {
		return false
	}
	if a.banExpireTime == 0 {
		return true
	}
	return now < a.banExpireTime
}

// Ban 封禁账号，状态机：正常→封禁。
//
// reason为封禁原因，banExpireTime为封禁过期时间戳（0=永久封禁）。
// 已封禁账号再次封禁将更新封禁原因与过期时间。
func (a *PlayerAccount) Ban(reason string, banExpireTime int64) error {
	a.status = AccountStatusBanned
	a.banReason = reason
	a.banExpireTime = banExpireTime
	return nil
}

// Unban 解封账号，状态机：封禁→正常，清空封禁原因与过期时间。
func (a *PlayerAccount) Unban() error {
	a.status = AccountStatusNormal
	a.banReason = ""
	a.banExpireTime = 0
	return nil
}

// VerifyPassword 校验密码是否匹配，委托PasswordHasher.Verify。
//
// hasher为密码哈希能力实现（infrastructure层注入）。
// 匹配返回true，不匹配返回false，哈希计算异常返回error。
func (a *PlayerAccount) VerifyPassword(ctx context.Context, password string, hasher PasswordHasher) (bool, error) {
	if hasher == nil {
		return false, fmt.Errorf("密码哈希器未注入: %w", gatewayerr.ErrPasswordHashFailed)
	}
	return hasher.Verify(ctx, password, a.passwordHash, a.salt)
}

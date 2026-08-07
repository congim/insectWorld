// Package token 访问令牌值对象与签发/校验能力接口。
//
// domain层零外部依赖（规范3），TokenSigner/TokenBlacklist接口在本包声明，
// infrastructure层实现HMAC-SHA256与Redis适配。
// 令牌负载不含密码/哈希/盐等敏感信息（spec 6.4 禁止项）。
package token

// TokenPayload 令牌负载值对象，封装令牌携带的玩家身份与时效信息。
//
// 所有时间戳用int64毫秒（规范8），版本号用int整型（规范8）。
// 负载不含密码/哈希/盐等敏感信息（spec 6.4 禁止项）。
type TokenPayload struct {
	PlayerID   int64 // 玩家ID，令牌归属的玩家
	IssueTime  int64 // 令牌签发时间戳，毫秒级
	ExpireTime int64 // 令牌过期时间戳，毫秒级
	Version    int   // 令牌版本号，登出/踢下线时递增使旧令牌失效
}

// AccessToken 访问令牌值对象，封装令牌负载与签名。
//
// 值对象语义：创建后不可变，通过方法访问字段。
// 签名由TokenSigner计算，防伪造（spec 4.3 安全性3）。
type AccessToken struct {
	payload   TokenPayload // 令牌负载
	signature string       // 令牌签名，HMAC-SHA256计算
}

// NewAccessToken 创建访问令牌值对象实例。
func NewAccessToken(playerID int64, issueTime, expireTime int64, version int) *AccessToken {
	return &AccessToken{
		payload: TokenPayload{
			PlayerID:   playerID,
			IssueTime:  issueTime,
			ExpireTime: expireTime,
			Version:    version,
		},
	}
}

// PlayerID 返回玩家ID。
func (t *AccessToken) PlayerID() int64 { return t.payload.PlayerID }

// IssueTime 返回令牌签发时间戳，毫秒级。
func (t *AccessToken) IssueTime() int64 { return t.payload.IssueTime }

// ExpireTime 返回令牌过期时间戳，毫秒级。
func (t *AccessToken) ExpireTime() int64 { return t.payload.ExpireTime }

// Version 返回令牌版本号。
func (t *AccessToken) Version() int { return t.payload.Version }

// Signature 返回令牌签名。
func (t *AccessToken) Signature() string { return t.signature }

// SetSignature 设置令牌签名，由TokenSigner.Sign调用。
func (t *AccessToken) SetSignature(sig string) {
	t.signature = sig
}

// ToPayload 返回令牌负载的拷贝。
func (t *AccessToken) ToPayload() TokenPayload {
	return t.payload
}

// IsExpired 判断令牌是否已过期，now为当前时间戳，毫秒级。
// now >= ExpireTime视为过期。
func (t *AccessToken) IsExpired(now int64) bool {
	return now >= t.payload.ExpireTime
}

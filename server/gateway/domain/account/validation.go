// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
package account

import (
	"unicode"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// ValidateUsernameFormat 校验用户名格式，覆盖spec 5.1.1 规则1全部验收条件。
//
// 校验规则：长度区间[minLen,maxLen]、字符集仅字母数字下划线、禁止纯数字、禁止空格特殊符号。
// 校验失败返回对应错误码变量（规范9，不裸返回errors.New）。
func ValidateUsernameFormat(username string, minLen, maxLen int) error {
	length := len(username)
	if length < minLen || length > maxLen {
		return gatewayerr.ErrInvalidUsernameFormat
	}
	hasLetter := false
	for _, r := range username {
		if r == '_' {
			continue
		}
		if isASCIILetter(r) {
			hasLetter = true
			continue
		}
		if unicode.IsDigit(r) {
			continue
		}
		return gatewayerr.ErrInvalidUsernameFormat
	}
	if !hasLetter {
		return gatewayerr.ErrInvalidUsernameFormat
	}
	return nil
}

// isASCIILetter 判断是否为ASCII字母（a-z, A-Z），不允许Unicode字母（'如中文'。
func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// ValidatePasswordStrength 校验密码强度，覆盖spec 5.1.1 规则2全部验收条件。
//
// 校验规则：长度区间[minLen,maxLen]、必须同时含字母与数字。
// 校验失败返回对应错误码变量（规范9）。
func ValidatePasswordStrength(password string, minLen, maxLen int) error {
	length := len(password)
	if length < minLen || length > maxLen {
		return gatewayerr.ErrInvalidPasswordStrength
	}
	hasLetter := false
	hasDigit := false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return gatewayerr.ErrInvalidPasswordStrength
	}
	return nil
}

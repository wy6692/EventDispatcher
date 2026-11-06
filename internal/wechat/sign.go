package wechat

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"sort"
	"strings"
)

// Signature 计算微信 SHA1 签名。extra 在加密模式下传入密文或 echostr。
func Signature(token, timestamp, nonce string, extra ...string) string {
	parts := make([]string, 0, 3+len(extra))
	parts = append(parts, token, timestamp, nonce)
	parts = append(parts, extra...)
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(sum[:])
}

// ValidSignature 比较十六进制签名，长度不同直接失败。
func ValidSignature(expect, actual string) bool {
	if len(expect) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.ToLower(expect)), []byte(strings.ToLower(actual))) == 1
}

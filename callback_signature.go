package wechatease

import (
	"crypto/subtle"
	"sort"
	"strings"
)

// GenerateCallbackSignature 生成微信服务器回调签名。
// 计算规则为：按字典序排序 token、timestamp、nonce，拼接后计算 SHA-1。
func GenerateCallbackSignature(token, timestamp, nonce string) string {
	params := []string{token, timestamp, nonce}
	sort.Strings(params)
	return sha1Hex(strings.Join(params, ""))
}

// VerifyCallbackSignature 校验微信服务器回调签名。
// 任一必需参数为空时直接返回 false，避免不完整请求被误认为有效请求。
func VerifyCallbackSignature(token, timestamp, nonce, signature string) bool {
	if token == "" || timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	calculated := GenerateCallbackSignature(token, timestamp, nonce)
	return subtle.ConstantTimeCompare([]byte(calculated), []byte(signature)) == 1
}

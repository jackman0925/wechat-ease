package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func main() {
	// 按官方规则构造原始字符串（示例，仅演示）。
	// 实际业务请使用你自己的原始内容与 session_key。
	raw := `{"foo":"bar"}`
	sessionKey := "your-session-key"

	signature := hmacSHA256Hex([]byte(raw), []byte(sessionKey))
	fmt.Println("signature:", signature)
}

func hmacSHA256Hex(message, key []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

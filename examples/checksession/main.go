package main

import (
	"context"
	"fmt"
	"time"

	wechatease "github.com/jackman0925/wechat-ease"
)

func main() {
	client := wechatease.NewClient(
		wechatease.WithTimeout(5*time.Second),
		wechatease.WithMaxRetries(3),
	)

	// 注意：signature 需要由你的服务端按官方规则生成（sig_method=hmac_sha256）。
	err := client.CheckSession(
		context.Background(),
		"your-access-token",
		"user-openid",
		"login-signature",
		"hmac_sha256",
	)
	if err != nil {
		fmt.Println("checksession failed:", err)
		return
	}

	fmt.Println("checksession ok")
}

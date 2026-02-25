package main

import (
	"context"
	"fmt"
	"time"

	wechatease "github.com/jackman0925/wechat-ease"
)

func main() {
	// 使用 Option 方式构建客户端：配置超时、重试和重试间隔。
	client := wechatease.NewClient(
		wechatease.WithTimeout(5*time.Second),
		wechatease.WithMaxRetries(3),
		wechatease.WithRetryInterval(200*time.Millisecond),
	)

	// 替换成真实 appid/appsecret 后即可调用。
	token, expiresIn, err := client.FetchAccessToken(context.Background(), "your-app-id", "your-app-secret")
	if err != nil {
		fmt.Println("fetch access token failed:", err)
		return
	}

	fmt.Println("access_token:", token)
	fmt.Println("expires_in:", expiresIn)
}

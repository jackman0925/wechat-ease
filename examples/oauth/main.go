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

	// 1) 通过授权 code 获取 SNS access_token/openid。
	accessToken, openID, refreshToken, unionID, err := client.FetchSnsAccessToken(
		context.Background(),
		"your-app-id",
		"your-app-secret",
		"auth-code-from-wechat",
	)
	if err != nil {
		fmt.Println("fetch sns access token failed:", err)
		return
	}

	fmt.Println("sns access_token:", accessToken)
	fmt.Println("openid:", openID)
	fmt.Println("refresh_token:", refreshToken)
	fmt.Println("unionid:", unionID)

	// 2) 使用 SNS access_token 拉取用户信息。
	nickname, headimgurl, err := client.FetchSnsUserInfo(context.Background(), accessToken, openID)
	if err != nil {
		fmt.Println("fetch sns user info failed:", err)
		return
	}
	fmt.Println("nickname:", nickname)
	fmt.Println("headimgurl:", headimgurl)
}

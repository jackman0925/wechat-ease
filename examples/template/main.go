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

	// 模板消息 data 字段需为合法 JSON 字符串。
	reqData := `{
		"first": {"value": "您好，您有一条新通知"},
		"keyword1": {"value": "订单创建成功"},
		"keyword2": {"value": "2026-02-25 20:00:00"},
		"remark": {"value": "感谢使用 wechat-ease"}
	}`

	err := client.PostTemplate(
		context.Background(),
		"your-access-token",
		reqData,
		"https://your-business-page.example.com",
		"your-template-id",
		"user-openid",
	)
	if err != nil {
		fmt.Println("post template failed:", err)
		return
	}

	fmt.Println("post template success")
}

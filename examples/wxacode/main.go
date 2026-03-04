package main

import (
	"context"
	"fmt"
	"os"
	"time"

	wechatease "github.com/jackman0925/wechat-ease"
)

func main() {
	client := wechatease.NewClient(
		wechatease.WithTimeout(8*time.Second),
		wechatease.WithMaxRetries(3),
	)

	checkPath := true
	autoColor := false
	isHyaline := false

	imgBytes, err := client.FetchWxaCodeUnlimited(context.Background(), "your-access-token", wechatease.WxaCodeUnlimitedRequest{
		Scene:     "order=20260304",
		Page:      "pages/order/detail",
		CheckPath: &checkPath,
		EnvVer:    "release",
		Width:     430,
		AutoColor: &autoColor,
		LineColor: &wechatease.WxaCodeLineColor{
			R: "0",
			G: "0",
			B: "0",
		},
		IsHyaline: &isHyaline,
	})
	if err != nil {
		fmt.Println("fetch wxacode failed:", err)
		return
	}

	if err := os.WriteFile("wxacode.png", imgBytes, 0o644); err != nil {
		fmt.Println("write file failed:", err)
		return
	}
	fmt.Println("wxacode saved to wxacode.png")
}

package main

import (
	"context"
	"fmt"
	"time"

	wechatease "github.com/jackman0925/wechat-ease"
)

// (虚拟支付)通知已经发货完成
func main() {
	client := wechatease.NewClient(
		wechatease.WithTimeout(8*time.Second),
		wechatease.WithMaxRetries(3),
	)

	// pay_sig 需要按微信支付签名规则生成（此处仅示例占位）。
	paySig := "your-pay-signature"

	err := client.NotifyProvideGoods(context.Background(), "your-access-token", paySig, wechatease.NotifyProvideGoodsRequest{
		OrderID: "your-order-id", // 或使用 WxOrderID 二选一
		Env:     0,               // 0-正式环境 1-沙箱环境
	})
	if err != nil {
		fmt.Println("notify_provide_goods failed:", err)
		return
	}

	fmt.Println("notify_provide_goods ok")
}

package main

import (
	"context"
	"fmt"
	"time"

	wechatease "github.com/jackman0925/wechat-ease"
)

// 微信虚拟支付退款功能
func main() {
	client := wechatease.NewClient(
		wechatease.WithTimeout(8*time.Second),
		wechatease.WithMaxRetries(3),
	)

	// pay_sig 需要按微信支付签名规则生成（此处仅示例占位）。
	paySig := "your-pay-signature"

	resp, err := client.RefundOrder(context.Background(), "your-access-token", paySig, wechatease.RefundOrderRequest{
		OpenID:        "user-openid",
		OrderID:       "your-order-id", // 或者使用 WxOrderID 二选一
		RefundOrderID: "refund202603300001",
		LeftFee:       100,
		RefundFee:     50,
		RefundReason:  1,
		ReqFrom:       2,
		Env:           0,
	})
	if err != nil {
		fmt.Println("refund_order failed:", err)
		return
	}

	fmt.Println("refund_order accepted:", resp.RefundOrderID)
}

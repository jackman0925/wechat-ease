# wechat-ease

`wechat-ease` 是一个面向 Go 项目的微信 API 轻量组件库。  
目标是：无第三方依赖、接口简单、生产可用（超时控制、重试机制、统一错误处理、错误拦截）。

## 特性

- 仅使用 Go 标准库
- `Client + Option` 配置模式，接入简单
- 内置超时、重试、退避
- 统一微信业务错误结构（`errcode` / `errmsg`）
- 支持错误拦截器，便于日志、打点、监控
- 提供全局便捷函数，兼容快速接入场景

## 安装与要求

- Go: `1.23.12`

本项目当前是本地库形态，直接在你的工程中以 module 方式引用即可。

## 快速开始

```go
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

	token, expiresIn, err := client.FetchAccessToken(context.Background(), "your-app-id", "your-app-secret")
	if err != nil {
		panic(err)
	}
	fmt.Println("access_token:", token, "expires_in:", expiresIn)
}
```

## 示例目录

- [examples/basic/main.go](examples/basic/main.go)
- [examples/checksession/main.go](examples/checksession/main.go)
- [examples/oauth/main.go](examples/oauth/main.go)
- [examples/refund_order/main.go](examples/refund_order/main.go)
- [examples/signature/main.go](examples/signature/main.go)
- [examples/template/main.go](examples/template/main.go)
- [examples/wxacode/main.go](examples/wxacode/main.go)

## 主要 API

- `FetchUserOpenID`
- `FetchAccessToken`
- `FetchAccessTokenWithRetry`
- `CheckSession`
- `RefundOrder`
- `PostTemplate`
- `PostTemplateDirectly`
- `FetchJsapiTicket`
- `FetchSnsAccessToken`
- `FetchSnsUserInfo`
- `FetchSnsRefreshToken`
- `FetchWxSign`
- `FetchWxaCodeUnlimited`

## 小程序码接口示例

```go
img, err := client.FetchWxaCodeUnlimited(ctx, accessToken, wechatease.WxaCodeUnlimitedRequest{
	Scene:  "order=12345",
	Page:   "pages/order/detail",
	EnvVer: "release",
	Width:  430,
})
if err != nil {
	panic(err)
}
_ = img // 成功时为图片二进制
```

## 运行测试

```bash
go test ./...
```

## 设计原则

- 极简：最小 API 面，直接可用
- 稳定：默认参数可用于生产环境基线
- 可控：重试、超时、错误处理均可配置

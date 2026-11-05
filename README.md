# wechat-ease

`wechat-ease` 是一个面向 Go 项目的微信 API 轻量组件库。  
目标是：无第三方依赖、接口简单、生产可用（超时控制、重试机制、统一错误处理、错误拦截）。

## 特性

- 仅使用 Go 标准库
- `Client + Option` 配置模式，接入简单
- 内置超时、重试、退避
- 统一微信业务错误结构（`errcode` / `errmsg`）
- 支持错误拦截器，便于日志、打点、监控
- 支持微信服务器回调签名生成与校验
- 支持基于 `EncodingAESKey` 的安全模式消息加解密
- 支持已解密小程序平台通知的强类型 JSON 解析
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
- [examples/authorized_user_information_change/main.go](examples/authorized_user_information_change/main.go)
- [examples/callback_signature/main.go](examples/callback_signature/main.go)
- [examples/checksession/main.go](examples/checksession/main.go)
- [examples/encrypted_callback/main.go](examples/encrypted_callback/main.go)
- [examples/notify_provide_goods/main.go](examples/notify_provide_goods/main.go)
- [examples/oauth/main.go](examples/oauth/main.go)
- [examples/penalty_violation/main.go](examples/penalty_violation/main.go)
- [examples/refund_order/main.go](examples/refund_order/main.go)
- [examples/signature/main.go](examples/signature/main.go)
- [examples/submit_pages/main.go](examples/submit_pages/main.go)
- [examples/template/main.go](examples/template/main.go)
- [examples/wxacode/main.go](examples/wxacode/main.go)

## 主要 API

- `FetchUserOpenID`
- `FetchAccessToken`
- `FetchAccessTokenWithRetry`
- `CheckSession`
- `RefundOrder`
- `NotifyProvideGoods`
- `SubmitPages`
- `PostTemplate`
- `PostTemplateDirectly`
- `FetchJsapiTicket`
- `FetchSnsAccessToken`
- `FetchSnsUserInfo`
- `FetchSnsRefreshToken`
- `FetchWxSign`
- `FetchWxaCodeUnlimited`
- `GenerateCallbackSignature`
- `VerifyCallbackSignature`
- `NewMessageCryptor`
- `GenerateEncryptedCallbackSignature`
- `VerifyEncryptedCallbackSignature`
- `event.ParsePlatformEventJSON`
- `event.ParsePenaltyViolationJSON`
- `event.ParseAuthorizedUserInformationChangeJSON`

## 微信服务器回调签名

库只负责签名计算与校验，不绑定 Gin、Echo 等 Web 框架：

```go
if !wechatease.VerifyCallbackSignature(token, timestamp, nonce, signature) {
	// 返回 403
	return
}

// 首次配置服务器 URL 时，验证成功后原样返回 echostr。
```

完整的 `net/http` 接入示例参见 [examples/callback_signature/main.go](examples/callback_signature/main.go)。

## 安全模式消息加解密

```go
cryptor, err := wechatease.NewMessageCryptor(token, encodingAESKey, appID)
if err != nil {
	panic(err)
}

// Encrypt 为请求 XML 或 JSON 中提取的密文字段。
message, err := cryptor.DecryptMessage(timestamp, nonce, msgSignature, encrypt)
if err != nil {
	panic(err)
}

// 安全模式下回复消息时，同样需要加密并返回签名。
reply, err := cryptor.EncryptMessage([]byte(replyXML), timestamp, nonce)
```

`DecryptMessage` 会依次完成消息签名校验、AES-256-CBC 解密、PKCS#7 去填充及 AppID 校验。完整示例参见 [examples/encrypted_callback/main.go](examples/encrypted_callback/main.go)。

## 小程序平台事件解析

`event` 子包只解析外部系统已验签、解密后的 JSON 正文，不依赖 HTTP 或 Web 框架。已支持 `wxa_punish_event`（小程序违规处罚信息通知）和授权用户信息变更事件；未知事件会以 `UnknownPlatformEvent` 返回并保留原始 JSON。

```go
notification, err := wechatevent.ParsePlatformEventJSON(body)
if err != nil {
	panic(err)
}

if penalty, ok := notification.(*wechatevent.PenaltyViolationEvent); ok {
	fmt.Println(penalty.PunishID)
}
```

完整示例参见 [examples/penalty_violation/main.go](examples/penalty_violation/main.go)。

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

## 维护约束

- 每个方法必须有对应的单元测试。
- 任意 Go 代码文件不得超过 500 行。
- `go test ./...` 会自动校验以上两项约束。

## 设计原则

- 极简：最小 API 面，直接可用
- 稳定：默认参数可用于生产环境基线
- 可控：重试、超时、错误处理均可配置

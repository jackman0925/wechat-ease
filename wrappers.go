package wechatease

import "context"

// defaultClient 为函数式 API 提供默认客户端实例。
var defaultClient = NewClient()

// WechatFetchUserOpenId 使用默认客户端执行小程序 code2Session。
func WechatFetchUserOpenId(ctx context.Context, appID, appSecret, code string) (openid, unionid, sessionKey string, err error) {
	return defaultClient.FetchUserOpenID(ctx, appID, appSecret, code)
}

// WechatCheckSession 使用默认客户端校验服务器保存的 session_key。
func WechatCheckSession(ctx context.Context, accessToken, openid, signature, sigMethod string) error {
	return defaultClient.CheckSession(ctx, accessToken, openid, signature, sigMethod)
}

// WechatRefundOrder 使用默认客户端发起微信虚拟支付退款任务。
func WechatRefundOrder(ctx context.Context, accessToken, paySig string, req RefundOrderRequest) (*RefundOrderResponse, error) {
	return defaultClient.RefundOrder(ctx, accessToken, paySig, req)
}

// WechatNotifyProvideGoods 使用默认客户端通知微信虚拟支付订单已发货。
func WechatNotifyProvideGoods(ctx context.Context, accessToken, paySig string, req NotifyProvideGoodsRequest) error {
	return defaultClient.NotifyProvideGoods(ctx, accessToken, paySig, req)
}

// WechatSubmitPages 使用默认客户端向微信搜索提交小程序页面。
func WechatSubmitPages(ctx context.Context, accessToken string, req SubmitPagesRequest) error {
	return defaultClient.SubmitPages(ctx, accessToken, req)
}

// WechatFetchAccessTokenTry3Time 使用默认重试配置获取公众号 access_token。
func WechatFetchAccessTokenTry3Time(ctx context.Context, appID, appSecret string) (token string, expiresIn int64, err error) {
	return defaultClient.FetchAccessTokenWithRetry(ctx, appID, appSecret)
}

// WechatFetchAccessToken 使用默认客户端获取公众号 access_token。
func WechatFetchAccessToken(ctx context.Context, appID, appSecret string) (token string, expiresIn int64, err error) {
	return defaultClient.FetchAccessToken(ctx, appID, appSecret)
}

// WechatPostTemplate 使用默认客户端组装并发送公众号模板消息。
func WechatPostTemplate(ctx context.Context, accessToken, reqData, jumpURL, templateID, openID string) error {
	return defaultClient.PostTemplate(ctx, accessToken, reqData, jumpURL, templateID, openID)
}

// WechatPostTemplateDirectly 使用默认客户端发送模板消息原始 JSON。
func WechatPostTemplateDirectly(ctx context.Context, accessToken, dataBody string) error {
	return defaultClient.PostTemplateDirectly(ctx, accessToken, []byte(dataBody))
}

// WechatFetchJsapiTicket 使用默认客户端获取公众号 JSAPI ticket。
func WechatFetchJsapiTicket(ctx context.Context, accessToken string) (string, error) {
	return defaultClient.FetchJsapiTicket(ctx, accessToken)
}

// WechatFetchSnsAccessToken 使用默认客户端通过网页授权 code 获取 token。
func WechatFetchSnsAccessToken(ctx context.Context, appID, appSecret, code string) (accessToken, openid, refreshToken, unionid string, err error) {
	return defaultClient.FetchSnsAccessToken(ctx, appID, appSecret, code)
}

// WechatFetchSnsUserInfo 使用默认客户端获取网页授权用户信息。
func WechatFetchSnsUserInfo(ctx context.Context, accessToken, openid string) (nickname, headimgurl string, err error) {
	return defaultClient.FetchSnsUserInfo(ctx, accessToken, openid)
}

// WechatFetchSnsRefreshToken 使用默认客户端刷新网页授权 access_token。
func WechatFetchSnsRefreshToken(ctx context.Context, appID, refreshToken string) (accessToken string, err error) {
	return defaultClient.FetchSnsRefreshToken(ctx, appID, refreshToken)
}

// WechatFetchWxSign 使用默认客户端生成公众号 JS-SDK 签名。
func WechatFetchWxSign(ctx context.Context, appID, appSecret, targetURL string) (*WxSignResult, error) {
	return defaultClient.FetchWxSign(ctx, appID, appSecret, targetURL)
}

// WechatFetchWxaCodeUnlimited 使用默认客户端生成不限数量的小程序码。
func WechatFetchWxaCodeUnlimited(ctx context.Context, accessToken string, req WxaCodeUnlimitedRequest) ([]byte, error) {
	return defaultClient.FetchWxaCodeUnlimited(ctx, accessToken, req)
}

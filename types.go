package wechatease

import (
	"encoding/json"
	"fmt"
)

// APIError 表示微信接口返回的业务错误（errcode/errmsg）。
type APIError struct {
	Code int
	Msg  string
}

// Error 返回包含微信错误码和错误信息的文本。
func (e *APIError) Error() string {
	return fmt.Sprintf("wechat error: errcode=%d, errmsg=%s", e.Code, e.Msg)
}

// BaseResponse 是仅包含微信通用错误字段的基础响应。
type BaseResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// Check 将通用响应中的 errcode 转换为 Go error。
func (r BaseResponse) Check() error {
	if r.ErrCode != 0 {
		return &APIError{Code: r.ErrCode, Msg: r.ErrMsg}
	}
	return nil
}

// SessionResponse 是小程序 code2Session 接口响应。
type SessionResponse struct {
	BaseResponse
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
}

// AccessTokenResponse 是公众号或网页授权 access_token 响应。
type AccessTokenResponse struct {
	BaseResponse
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
}

// UserInfoResponse 是微信网页授权用户信息响应。
type UserInfoResponse struct {
	BaseResponse
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
}

// TicketResponse 是公众号 JSAPI ticket 响应。
type TicketResponse struct {
	BaseResponse
	Ticket    string `json:"ticket"`
	ExpiresIn int64  `json:"expires_in"`
}

// WxSignResult 是公众号 JS-SDK 签名结果。
type WxSignResult struct {
	AppID     string `json:"appId"`
	Timestamp string `json:"timestamp"`
	NonceStr  string `json:"noncestr"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

// TemplateMessageRequest 对应公众号模板消息发送体。
type TemplateMessageRequest struct {
	ToUser     string          `json:"touser"`
	TemplateID string          `json:"template_id"`
	URL        string          `json:"url,omitempty"`
	Data       json.RawMessage `json:"data"`
}

// WxaCodeLineColor 小程序码线条颜色（RGB 十进制字符串）。
type WxaCodeLineColor struct {
	R string `json:"r"`
	G string `json:"g"`
	B string `json:"b"`
}

// WxaCodeUnlimitedRequest 对应 wxacode.getUnlimited 请求体。
type WxaCodeUnlimitedRequest struct {
	Scene     string            `json:"scene"`
	Page      string            `json:"page,omitempty"`
	CheckPath *bool             `json:"check_path,omitempty"`
	EnvVer    string            `json:"env_version,omitempty"`
	Width     int               `json:"width,omitempty"`
	AutoColor *bool             `json:"auto_color,omitempty"`
	LineColor *WxaCodeLineColor `json:"line_color,omitempty"`
	IsHyaline *bool             `json:"is_hyaline,omitempty"`
}

// RefundOrderRequest 微信虚拟支付退款请求体（xpay/refund_order）。
type RefundOrderRequest struct {
	OpenID        string `json:"openid"`
	OrderID       string `json:"order_id,omitempty"`
	WxOrderID     string `json:"wx_order_id,omitempty"`
	RefundOrderID string `json:"refund_order_id"`
	LeftFee       int64  `json:"left_fee"`
	RefundFee     int64  `json:"refund_fee"`
	BizMeta       string `json:"biz_meta,omitempty"`
	RefundReason  int    `json:"refund_reason,omitempty"`
	ReqFrom       int    `json:"req_from,omitempty"`
	Env           int    `json:"env,omitempty"`
}

// RefundOrderResponse 微信虚拟支付退款返回。
type RefundOrderResponse struct {
	BaseResponse
	RefundOrderID   string `json:"refund_order_id"`
	RefundWxOrderID string `json:"refund_wx_order_id"`
	PayOrderID      string `json:"pay_order_id"`
	PayWxOrderID    string `json:"pay_wx_order_id"`
}

// NotifyProvideGoodsRequest 通知已发货请求体（xpay/notify_provide_goods）。
type NotifyProvideGoodsRequest struct {
	OrderID   string `json:"order_id,omitempty"`
	WxOrderID string `json:"wx_order_id,omitempty"`
	Env       int    `json:"env,omitempty"`
}

// SubmitPage 表示一个待提交到微信搜索的小程序页面。
type SubmitPage struct {
	Path  string `json:"path"`
	Query string `json:"query,omitempty"`
}

// SubmitPagesRequest 对应小程序搜索 submitPages 请求体。
type SubmitPagesRequest struct {
	Pages []SubmitPage `json:"pages"`
}

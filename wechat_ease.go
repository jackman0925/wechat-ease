package wechatease

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// 默认微信 API 地址和客户端行为参数。
	defaultBaseURL       = "https://api.weixin.qq.com"
	defaultTimeout       = 10 * time.Second
	defaultMaxRetries    = 3
	defaultRetryInterval = 300 * time.Millisecond
)

// APIError 表示微信接口返回的业务错误（errcode/errmsg）。
type APIError struct {
	Code int
	Msg  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("wechat error: errcode=%d, errmsg=%s", e.Code, e.Msg)
}

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

type SessionResponse struct {
	BaseResponse
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
}

type AccessTokenResponse struct {
	BaseResponse
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
}

type UserInfoResponse struct {
	BaseResponse
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
}

type TicketResponse struct {
	BaseResponse
	Ticket    string `json:"ticket"`
	ExpiresIn int64  `json:"expires_in"`
}

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

// RefundOrderRequest 微信虚拟支付退款请求体（xpay/refund_order）。 https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/business-capabilities/virtual-payment.html
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

// ErrorInterceptor 用于在错误返回前做统一拦截（包装、打点、上报等）。
type ErrorInterceptor func(ctx context.Context, err error) error

// Client 是 wechat-ease 的核心客户端，零三方依赖。
type Client struct {
	baseURL        string
	httpClient     *http.Client
	maxRetries     int
	retryInterval  time.Duration
	errorIntercept ErrorInterceptor
}

type Option func(*Client)

// WithBaseURL 覆盖默认微信 API 地址（测试或代理场景常用）。
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) == "" {
			return
		}
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient 注入外部 http.Client，便于复用连接池或自定义传输层。
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithTimeout 设置请求超时。
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.httpClient.Timeout = timeout
		}
	}
}

// WithMaxRetries 设置 access_token 获取的最大重试次数。
func WithMaxRetries(maxRetries int) Option {
	return func(c *Client) {
		if maxRetries > 0 {
			c.maxRetries = maxRetries
		}
	}
}

// WithRetryInterval 设置重试基础间隔（实际按次数线性退避）。
func WithRetryInterval(interval time.Duration) Option {
	return func(c *Client) {
		if interval > 0 {
			c.retryInterval = interval
		}
	}
}

// WithErrorInterceptor 设置统一错误拦截器。
func WithErrorInterceptor(interceptor ErrorInterceptor) Option {
	return func(c *Client) {
		c.errorIntercept = interceptor
	}
}

// NewClient 创建客户端并应用默认生产可用配置。
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:       defaultBaseURL,
		httpClient:    &http.Client{Timeout: defaultTimeout},
		maxRetries:    defaultMaxRetries,
		retryInterval: defaultRetryInterval,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) wrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if c.errorIntercept != nil {
		return c.errorIntercept(ctx, err)
	}
	return err
}

// doJSON 统一处理 GET/POST 请求、状态码校验和 JSON 解码。
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body []byte, out any) error {
	apiURL := c.baseURL + path
	if len(query) > 0 {
		apiURL += "?" + query.Encode()
	}

	var payload io.Reader
	if len(body) > 0 {
		payload = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, payload)
	if err != nil {
		return c.wrapError(ctx, err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.wrapError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyPart, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return c.wrapError(ctx, fmt.Errorf("wechat http status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyPart))))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return c.wrapError(ctx, fmt.Errorf("decode wechat response failed: %w", err))
	}
	return nil
}

// doBinary 处理返回二进制内容的接口（如小程序码）。
// 若微信返回 JSON 错误对象（errcode/errmsg），会自动转成 APIError。
func (c *Client) doBinary(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	apiURL := c.baseURL + path
	if len(query) > 0 {
		apiURL += "?" + query.Encode()
	}

	var payload io.Reader
	if len(body) > 0 {
		payload = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, payload)
	if err != nil {
		return nil, c.wrapError(ctx, err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.wrapError(ctx, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, c.wrapError(ctx, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.wrapError(ctx, fmt.Errorf("wechat http status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes))))
	}

	trimmed := strings.TrimSpace(string(respBytes))
	if strings.HasPrefix(trimmed, "{") {
		var baseResp BaseResponse
		if err := json.Unmarshal(respBytes, &baseResp); err == nil {
			if err := baseResp.Check(); err != nil {
				return nil, c.wrapError(ctx, err)
			}
		}
	}
	return respBytes, nil
}

// shouldRetry 仅在可恢复错误时触发重试，避免无意义重放。
func (c *Client) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	var netErr net.Error
	if ok := errorAs(err, &netErr); ok && netErr.Timeout() {
		return true
	}
	var apiErr *APIError
	if ok := errorAs(err, &apiErr); ok && apiErr.Code == -1 {
		return true
	}
	return false
}

// errorAs 是轻量级 errors.As，避免引入额外依赖并兼容包裹错误。
func errorAs[T error](err error, target *T) bool {
	if err == nil {
		return false
	}
	v, ok := err.(T)
	if ok {
		*target = v
		return true
	}
	type causer interface{ Unwrap() error }
	u, ok := err.(causer)
	if !ok {
		return false
	}
	return errorAs(u.Unwrap(), target)
}

// sleepRetry 使用可取消等待，确保 context 失效时及时退出。
func (c *Client) sleepRetry(ctx context.Context, n int) error {
	wait := time.Duration(n) * c.retryInterval
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// FetchUserOpenID 小程序 code 换 openid/unionid/session_key。 参考: https://developers.weixin.qq.com/minigame/dev/api-backend/open-api/login/auth.code2Session.html
func (c *Client) FetchUserOpenID(ctx context.Context, appID, appSecret, code string) (openid, unionid, sessionKey string, err error) {
	query := url.Values{
		"appid":      {appID},
		"secret":     {appSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	}
	var resp SessionResponse
	if err := c.doJSON(ctx, http.MethodGet, "/sns/jscode2session", query, nil, &resp); err != nil {
		return "", "", "", err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", "", "", err
	}
	return resp.OpenID, resp.UnionID, resp.SessionKey, nil
}

// CheckSession 校验服务器保存的 session_key 是否仍然有效（/wxa/checksession）。参考: https://developers.weixin.qq.com/minigame/dev/api-backend/open-api/login/auth.checkSessionKey.html
func (c *Client) CheckSession(ctx context.Context, accessToken, openid, signature, sigMethod string) error {
	if strings.TrimSpace(accessToken) == "" {
		return c.wrapError(ctx, fmt.Errorf("access_token is required"))
	}
	if strings.TrimSpace(openid) == "" {
		return c.wrapError(ctx, fmt.Errorf("openid is required"))
	}
	if strings.TrimSpace(signature) == "" {
		return c.wrapError(ctx, fmt.Errorf("signature is required"))
	}
	sigMethod = strings.TrimSpace(sigMethod)
	if sigMethod == "" {
		return c.wrapError(ctx, fmt.Errorf("sig_method is required"))
	}
	if sigMethod != "hmac_sha256" {
		return c.wrapError(ctx, fmt.Errorf("sig_method must be hmac_sha256"))
	}

	query := url.Values{
		"access_token": {accessToken},
		"signature":    {signature},
		"openid":       {openid},
		"sig_method":   {sigMethod},
	}
	var resp BaseResponse
	if err := c.doJSON(ctx, http.MethodGet, "/wxa/checksession", query, nil, &resp); err != nil {
		return err
	}
	return c.wrapError(ctx, resp.Check())
}

// RefundOrder 发起微信虚拟支付退款任务（xpay/refund_order）。
// 注意：接口仅启动退款任务，需后续通过 query_order 查询退款状态。
func (c *Client) RefundOrder(ctx context.Context, accessToken, paySig string, req RefundOrderRequest) (*RefundOrderResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("access_token is required"))
	}
	if strings.TrimSpace(paySig) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("pay_sig is required"))
	}
	if strings.TrimSpace(req.OpenID) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("openid is required"))
	}
	if strings.TrimSpace(req.RefundOrderID) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_order_id is required"))
	}
	if l := len(req.RefundOrderID); l < 8 || l > 32 {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_order_id length must be between 8 and 32"))
	}
	if strings.TrimSpace(req.OrderID) == "" && strings.TrimSpace(req.WxOrderID) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("order_id or wx_order_id is required"))
	}
	if req.LeftFee <= 0 {
		return nil, c.wrapError(ctx, fmt.Errorf("left_fee must be greater than 0"))
	}
	if req.RefundFee <= 0 {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_fee must be greater than 0"))
	}
	if req.RefundFee > req.LeftFee {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_fee must be <= left_fee"))
	}

	query := url.Values{
		"access_token": {accessToken},
		"pay_sig":      {paySig},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, c.wrapError(ctx, fmt.Errorf("marshal refund_order request failed: %w", err))
	}
	var resp RefundOrderResponse
	if err := c.doJSON(ctx, http.MethodPost, "/xpay/refund_order", query, body, &resp); err != nil {
		return nil, err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NotifyProvideGoods 通知已经发货完成（xpay/notify_provide_goods）。
// 仅在 xpay_goods_deliver_notify 推送失败的异常场景下使用。
func (c *Client) NotifyProvideGoods(ctx context.Context, accessToken, paySig string, req NotifyProvideGoodsRequest) error {
	if strings.TrimSpace(accessToken) == "" {
		return c.wrapError(ctx, fmt.Errorf("access_token is required"))
	}
	if strings.TrimSpace(paySig) == "" {
		return c.wrapError(ctx, fmt.Errorf("pay_sig is required"))
	}
	if strings.TrimSpace(req.OrderID) == "" && strings.TrimSpace(req.WxOrderID) == "" {
		return c.wrapError(ctx, fmt.Errorf("order_id or wx_order_id is required"))
	}

	query := url.Values{
		"access_token": {accessToken},
		"pay_sig":      {paySig},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return c.wrapError(ctx, fmt.Errorf("marshal notify_provide_goods request failed: %w", err))
	}
	var resp BaseResponse
	if err := c.doJSON(ctx, http.MethodPost, "/xpay/notify_provide_goods", query, body, &resp); err != nil {
		return err
	}
	return c.wrapError(ctx, resp.Check())
}

// FetchAccessToken 获取公众号基础 access_token。
func (c *Client) FetchAccessToken(ctx context.Context, appID, appSecret string) (string, int64, error) {
	query := url.Values{
		"grant_type": {"client_credential"},
		"appid":      {appID},
		"secret":     {appSecret},
	}
	var resp AccessTokenResponse
	if err := c.doJSON(ctx, http.MethodGet, "/cgi-bin/token", query, nil, &resp); err != nil {
		return "", 0, err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", 0, err
	}
	return resp.AccessToken, resp.ExpiresIn, nil
}

// FetchAccessTokenWithRetry 获取 access_token（含重试与退避）。
func (c *Client) FetchAccessTokenWithRetry(ctx context.Context, appID, appSecret string) (string, int64, error) {
	var lastErr error
	for i := 1; i <= c.maxRetries; i++ {
		token, exp, err := c.FetchAccessToken(ctx, appID, appSecret)
		if err == nil && token != "" {
			return token, exp, nil
		}
		if err == nil {
			err = fmt.Errorf("empty access_token in response")
		}
		lastErr = err
		if i == c.maxRetries || !c.shouldRetry(err) {
			break
		}
		if sleepErr := c.sleepRetry(ctx, i); sleepErr != nil {
			return "", 0, c.wrapError(ctx, sleepErr)
		}
	}
	return "", 0, c.wrapError(ctx, fmt.Errorf("fetch access token failed after %d attempts: %w", c.maxRetries, lastErr))
}

// PostTemplate 以 data JSON 字符串发送模板消息。
func (c *Client) PostTemplate(ctx context.Context, accessToken, reqData, jumpURL, templateID, openID string) error {
	if !json.Valid([]byte(reqData)) {
		return c.wrapError(ctx, fmt.Errorf("template data is not valid json"))
	}
	req := TemplateMessageRequest{
		ToUser:     openID,
		TemplateID: templateID,
		URL:        jumpURL,
		Data:       json.RawMessage(reqData),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return c.wrapError(ctx, fmt.Errorf("marshal template request failed: %w", err))
	}
	return c.PostTemplateDirectly(ctx, accessToken, body)
}

// PostTemplateDirectly 直接发送模板消息原始 JSON。
func (c *Client) PostTemplateDirectly(ctx context.Context, accessToken string, dataBody []byte) error {
	query := url.Values{"access_token": {accessToken}}
	var resp BaseResponse
	if err := c.doJSON(ctx, http.MethodPost, "/cgi-bin/message/template/send", query, dataBody, &resp); err != nil {
		return err
	}
	return c.wrapError(ctx, resp.Check())
}

// FetchJsapiTicket 获取公众号 JSAPI ticket。
func (c *Client) FetchJsapiTicket(ctx context.Context, accessToken string) (string, error) {
	query := url.Values{
		"access_token": {accessToken},
		"type":         {"jsapi"},
	}
	var resp TicketResponse
	if err := c.doJSON(ctx, http.MethodGet, "/cgi-bin/ticket/getticket", query, nil, &resp); err != nil {
		return "", err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", err
	}
	return resp.Ticket, nil
}

// FetchSnsAccessToken 网页授权场景 code 换 access_token。
func (c *Client) FetchSnsAccessToken(ctx context.Context, appID, appSecret, code string) (accessToken, openid, refreshToken, unionid string, err error) {
	query := url.Values{
		"appid":      {appID},
		"secret":     {appSecret},
		"code":       {code},
		"grant_type": {"authorization_code"},
	}
	var resp AccessTokenResponse
	if err := c.doJSON(ctx, http.MethodGet, "/sns/oauth2/access_token", query, nil, &resp); err != nil {
		return "", "", "", "", err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", "", "", "", err
	}
	return resp.AccessToken, resp.OpenID, resp.RefreshToken, resp.UnionID, nil
}

// FetchSnsUserInfo 通过 SNS access_token 拉取用户信息。
func (c *Client) FetchSnsUserInfo(ctx context.Context, accessToken, openid string) (nickname, headimgurl string, err error) {
	query := url.Values{
		"access_token": {accessToken},
		"openid":       {openid},
	}
	var resp UserInfoResponse
	if err := c.doJSON(ctx, http.MethodGet, "/sns/userinfo", query, nil, &resp); err != nil {
		return "", "", err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", "", err
	}
	return resp.Nickname, resp.HeadImgURL, nil
}

// FetchSnsRefreshToken 刷新 SNS access_token。
func (c *Client) FetchSnsRefreshToken(ctx context.Context, appID, refreshToken string) (string, error) {
	query := url.Values{
		"appid":         {appID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	var resp AccessTokenResponse
	if err := c.doJSON(ctx, http.MethodGet, "/sns/oauth2/refresh_token", query, nil, &resp); err != nil {
		return "", err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		return "", c.wrapError(ctx, fmt.Errorf("access_token empty in response"))
	}
	return resp.AccessToken, nil
}

// FetchWxSign 生成公众号 JS-SDK 所需签名参数。
func (c *Client) FetchWxSign(ctx context.Context, appID, appSecret, targetURL string) (*WxSignResult, error) {
	accessToken, _, err := c.FetchAccessToken(ctx, appID, appSecret)
	if err != nil {
		return nil, c.wrapError(ctx, fmt.Errorf("fetch access token failed: %w", err))
	}
	if accessToken == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("access_token is empty"))
	}

	ticket, err := c.FetchJsapiTicket(ctx, accessToken)
	if err != nil {
		return nil, c.wrapError(ctx, fmt.Errorf("fetch jsapi ticket failed: %w", err))
	}

	nonce, err := generateNonce(16)
	if err != nil {
		return nil, c.wrapError(ctx, fmt.Errorf("generate nonce failed: %w", err))
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	raw := "jsapi_ticket=" + ticket + "&noncestr=" + nonce + "&timestamp=" + timestamp + "&url=" + targetURL

	return &WxSignResult{
		AppID:     appID,
		Timestamp: timestamp,
		NonceStr:  nonce,
		URL:       targetURL,
		Signature: sha1Hex(raw),
	}, nil
}

// FetchWxaCodeUnlimited 生成小程序码（wxacode.getUnlimited）。
// 成功返回图片二进制，失败返回微信 JSON 错误对象。
func (c *Client) FetchWxaCodeUnlimited(ctx context.Context, accessToken string, req WxaCodeUnlimitedRequest) ([]byte, error) {
	scene := strings.TrimSpace(req.Scene)
	if scene == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("scene is required"))
	}
	if len(scene) > 32 {
		return nil, c.wrapError(ctx, fmt.Errorf("scene length must be <= 32"))
	}
	if req.Width != 0 && (req.Width < 280 || req.Width > 1280) {
		return nil, c.wrapError(ctx, fmt.Errorf("width must be between 280 and 1280"))
	}
	req.Scene = scene

	query := url.Values{"access_token": {accessToken}}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, c.wrapError(ctx, fmt.Errorf("marshal wxacode request failed: %w", err))
	}
	return c.doBinary(ctx, http.MethodPost, "/wxa/getwxacodeunlimit", query, body)
}

// generateNonce 生成指定长度随机字符串（字母数字）。
func generateNonce(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid nonce length: %d", length)
	}
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	max := big.NewInt(int64(len(chars)))
	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[i] = chars[n.Int64()]
	}
	return string(buf), nil
}

// sha1Hex 计算字符串的 SHA1 十六进制摘要。
func sha1Hex(s string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// 默认全局客户端，便于旧项目直接调用函数式 API。
var defaultClient = NewClient()

// WechatFetchUserOpenId 小程序 code 换 openid。
func WechatFetchUserOpenId(ctx context.Context, appID, appSecret, code string) (openid, unionid, sessionKey string, err error) {
	return defaultClient.FetchUserOpenID(ctx, appID, appSecret, code)
}

// WechatCheckSession 校验 session_key 是否有效。
func WechatCheckSession(ctx context.Context, accessToken, openid, signature, sigMethod string) error {
	return defaultClient.CheckSession(ctx, accessToken, openid, signature, sigMethod)
}

// WechatRefundOrder 发起微信虚拟支付退款任务（xpay/refund_order）。
func WechatRefundOrder(ctx context.Context, accessToken, paySig string, req RefundOrderRequest) (*RefundOrderResponse, error) {
	return defaultClient.RefundOrder(ctx, accessToken, paySig, req)
}

// WechatNotifyProvideGoods 通知已经发货完成（xpay/notify_provide_goods）。
func WechatNotifyProvideGoods(ctx context.Context, accessToken, paySig string, req NotifyProvideGoodsRequest) error {
	return defaultClient.NotifyProvideGoods(ctx, accessToken, paySig, req)
}

// WechatFetchAccessTokenTry3Time 获取 access_token（默认重试）。
func WechatFetchAccessTokenTry3Time(ctx context.Context, appID, appSecret string) (token string, expiresIn int64, err error) {
	return defaultClient.FetchAccessTokenWithRetry(ctx, appID, appSecret)
}

// WechatFetchAccessToken 获取 access_token（不额外重试）。
func WechatFetchAccessToken(ctx context.Context, appID, appSecret string) (token string, expiresIn int64, err error) {
	return defaultClient.FetchAccessToken(ctx, appID, appSecret)
}

// WechatPostTemplate 发送模板消息。
func WechatPostTemplate(ctx context.Context, accessToken, reqData, jumpURL, templateID, openID string) error {
	return defaultClient.PostTemplate(ctx, accessToken, reqData, jumpURL, templateID, openID)
}

// WechatPostTemplateDirectly 直接发送模板消息原始 JSON。
func WechatPostTemplateDirectly(ctx context.Context, accessToken, dataBody string) error {
	return defaultClient.PostTemplateDirectly(ctx, accessToken, []byte(dataBody))
}

// WechatFetchJsapiTicket 获取 JSAPI ticket。
func WechatFetchJsapiTicket(ctx context.Context, accessToken string) (string, error) {
	return defaultClient.FetchJsapiTicket(ctx, accessToken)
}

// WechatFetchSnsAccessToken SNS 授权换 token。
func WechatFetchSnsAccessToken(ctx context.Context, appID, appSecret, code string) (accessToken, openid, refreshToken, unionid string, err error) {
	return defaultClient.FetchSnsAccessToken(ctx, appID, appSecret, code)
}

// WechatFetchSnsUserInfo SNS 用户信息。
func WechatFetchSnsUserInfo(ctx context.Context, accessToken, openid string) (nickname, headimgurl string, err error) {
	return defaultClient.FetchSnsUserInfo(ctx, accessToken, openid)
}

// WechatFetchSnsRefreshToken 刷新 SNS token。
func WechatFetchSnsRefreshToken(ctx context.Context, appID, refreshToken string) (accessToken string, err error) {
	return defaultClient.FetchSnsRefreshToken(ctx, appID, refreshToken)
}

// WechatFetchWxSign 生成公众号签名参数。
func WechatFetchWxSign(ctx context.Context, appID, appSecret, targetURL string) (*WxSignResult, error) {
	return defaultClient.FetchWxSign(ctx, appID, appSecret, targetURL)
}

// WechatFetchWxaCodeUnlimited 生成小程序码（无限制数量场景）。
func WechatFetchWxaCodeUnlimited(ctx context.Context, accessToken string, req WxaCodeUnlimitedRequest) ([]byte, error) {
	return defaultClient.FetchWxaCodeUnlimited(ctx, accessToken, req)
}

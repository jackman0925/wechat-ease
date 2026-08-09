package wechatease

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// 默认参数提供可直接使用的生产基线，也可通过 Option 覆盖。
	defaultBaseURL       = "https://api.weixin.qq.com"
	defaultTimeout       = 10 * time.Second
	defaultMaxRetries    = 3
	defaultRetryInterval = 300 * time.Millisecond
)

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

// Option 用于配置 Client。
type Option func(*Client)

// WithBaseURL 覆盖默认微信 API 地址（测试或代理场景常用）。
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
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

// requestError 隐藏底层请求错误中的完整 URL，避免查询参数中的凭据泄露。
type requestError struct {
	cause error
}

// Error 返回不包含请求 URL 和敏感查询参数的固定错误信息。
func (e *requestError) Error() string {
	return "wechat request failed"
}

// Unwrap 保留底层错误链，支持 errors.Is/errors.As，同时不暴露敏感 URL。
func (e *requestError) Unwrap() error {
	return e.cause
}

// wrapError 在错误返回调用方前执行统一拦截器。
func (c *Client) wrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if c.errorIntercept != nil {
		return c.errorIntercept(ctx, err)
	}
	return err
}

// safeRequestError 优先保留 context 语义，并隐藏底层错误中的完整请求 URL。
func safeRequestError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return &requestError{cause: err}
}

// redactQueryValues 从错误文本中移除查询参数的原值及 URL 编码值。
func redactQueryValues(text string, query url.Values) string {
	for _, values := range query {
		for _, value := range values {
			if value != "" {
				text = strings.ReplaceAll(text, value, "[REDACTED]")
				text = strings.ReplaceAll(text, url.QueryEscape(value), "[REDACTED]")
			}
		}
	}
	return text
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
		return c.wrapError(ctx, safeRequestError(ctx, err))
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.wrapError(ctx, safeRequestError(ctx, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyPart, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		message := redactQueryValues(strings.TrimSpace(string(bodyPart)), query)
		return c.wrapError(ctx, fmt.Errorf("wechat http status %d: %s", resp.StatusCode, message))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return c.wrapError(ctx, fmt.Errorf("decode wechat response failed: %w", err))
	}
	return nil
}

// doBinary 处理返回二进制内容的接口（如小程序码），并识别 JSON 业务错误。
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
		return nil, c.wrapError(ctx, safeRequestError(ctx, err))
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.wrapError(ctx, safeRequestError(ctx, err))
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, c.wrapError(ctx, err)
	}
	if resp.StatusCode != http.StatusOK {
		message := redactQueryValues(strings.TrimSpace(string(respBytes)), query)
		return nil, c.wrapError(ctx, fmt.Errorf("wechat http status %d: %s", resp.StatusCode, message))
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

// shouldRetry 仅允许系统繁忙和网络超时等可恢复错误进入重试。
func (c *Client) shouldRetry(err error) bool {
	if err == nil || err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	var netErr net.Error
	if ok := errorAs(err, &netErr); ok && netErr.Timeout() {
		return true
	}
	var apiErr *APIError
	return errorAs(err, &apiErr) && apiErr.Code == -1
}

// errorAs 沿 Unwrap 错误链查找目标错误类型。
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
	return ok && errorAs(u.Unwrap(), target)
}

// sleepRetry 按尝试次数线性退避，并响应 context 取消。
func (c *Client) sleepRetry(ctx context.Context, n int) error {
	timer := time.NewTimer(time.Duration(n) * c.retryInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

package wechatease

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// FetchAccessToken 获取微信公众号 access_token。
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
	query := url.Values{"access_token": {accessToken}, "type": {"jsapi"}}
	var resp TicketResponse
	if err := c.doJSON(ctx, http.MethodGet, "/cgi-bin/ticket/getticket", query, nil, &resp); err != nil {
		return "", err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return "", err
	}
	return resp.Ticket, nil
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
		AppID: appID, Timestamp: timestamp, NonceStr: nonce,
		URL: targetURL, Signature: sha1Hex(raw),
	}, nil
}

// generateNonce 使用密码学安全随机源生成指定长度的字母数字串。
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

// sha1Hex 计算字符串的 SHA1 十六进制摘要，用于 JS-SDK 签名。
func sha1Hex(s string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

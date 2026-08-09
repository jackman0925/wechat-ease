package wechatease

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// FetchUserOpenID 小程序 code 换 openid/unionid/session_key。
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

// CheckSession 校验服务器保存的 session_key 是否仍然有效。
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

// SubmitPages 向微信搜索提交已正式发布、可访问的小程序页面。
// 该接口会消耗平台额度，因此默认不进行自动重试。
func (c *Client) SubmitPages(ctx context.Context, accessToken string, req SubmitPagesRequest) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return c.wrapError(ctx, fmt.Errorf("access_token is required"))
	}
	if len(req.Pages) == 0 {
		return c.wrapError(ctx, fmt.Errorf("pages must contain at least one item"))
	}

	for i := range req.Pages {
		path := strings.TrimSpace(req.Pages[i].Path)
		queryString := strings.TrimSpace(req.Pages[i].Query)
		if path == "" {
			return c.wrapError(ctx, fmt.Errorf("pages[%d].path is required", i))
		}
		if strings.HasPrefix(path, "/") {
			return c.wrapError(ctx, fmt.Errorf("pages[%d].path must not start with /", i))
		}
		if strings.ContainsAny(path, "?#") {
			return c.wrapError(ctx, fmt.Errorf("pages[%d].path must not contain ? or #", i))
		}
		if strings.HasPrefix(queryString, "?") {
			return c.wrapError(ctx, fmt.Errorf("pages[%d].query must not start with ?", i))
		}
		if strings.Contains(queryString, "#") {
			return c.wrapError(ctx, fmt.Errorf("pages[%d].query must not contain #", i))
		}
		req.Pages[i].Path = path
		req.Pages[i].Query = queryString
	}

	body, err := json.Marshal(req)
	if err != nil {
		return c.wrapError(ctx, fmt.Errorf("marshal submit_pages request failed: %w", err))
	}
	query := url.Values{"access_token": {accessToken}}
	var resp BaseResponse
	if err := c.doJSON(ctx, http.MethodPost, "/wxa/search/wxaapi_submitpages", query, body, &resp); err != nil {
		return err
	}
	err = resp.Check()
	if apiErr, ok := err.(*APIError); ok {
		err = &APIError{Code: apiErr.Code, Msg: redactQueryValues(apiErr.Msg, query)}
	}
	return c.wrapError(ctx, err)
}

// FetchWxaCodeUnlimited 生成小程序码（wxacode.getUnlimited）。
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

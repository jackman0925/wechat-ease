package wechatease

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// FetchSnsAccessToken 网页授权场景 code 换 access_token。
func (c *Client) FetchSnsAccessToken(ctx context.Context, appID, appSecret, code string) (accessToken, openid, refreshToken, unionid string, err error) {
	query := url.Values{
		"appid": {appID}, "secret": {appSecret}, "code": {code},
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
	query := url.Values{"access_token": {accessToken}, "openid": {openid}}
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
		"appid": {appID}, "grant_type": {"refresh_token"}, "refresh_token": {refreshToken},
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

package wechatease

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGlobalWrappers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			_, _ = w.Write([]byte(`{"access_token":"at","expires_in":7200}`))
		case "/cgi-bin/ticket/getticket":
			_, _ = w.Write([]byte(`{"errcode":0,"ticket":"tk"}`))
		case "/sns/jscode2session":
			_, _ = w.Write([]byte(`{"openid":"oid","session_key":"sk"}`))
		case "/cgi-bin/message/template/send", "/wxa/checksession", "/xpay/notify_provide_goods", "/wxa/search/wxaapi_submitpages":
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		case "/sns/oauth2/access_token":
			_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","openid":"oid"}`))
		case "/sns/userinfo":
			_, _ = w.Write([]byte(`{"nickname":"nick","headimgurl":"img"}`))
		case "/sns/oauth2/refresh_token":
			_, _ = w.Write([]byte(`{"access_token":"new-at"}`))
		case "/wxa/getwxacodeunlimit":
			_, _ = w.Write([]byte{0x89, 0x50})
		case "/xpay/refund_order":
			_, _ = w.Write([]byte(`{"errcode":0,"refund_order_id":"refund1234"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	original := defaultClient
	defaultClient = NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	defer func() { defaultClient = original }()
	ctx := context.Background()

	if _, _, _, err := WechatFetchUserOpenId(ctx, "a", "b", "c"); err != nil {
		t.Fatal(err)
	}
	if err := WechatCheckSession(ctx, "at", "oid", "sig", "hmac_sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err := WechatRefundOrder(ctx, "at", "sig", validRefundRequest()); err != nil {
		t.Fatal(err)
	}
	if err := WechatNotifyProvideGoods(ctx, "at", "sig", NotifyProvideGoodsRequest{OrderID: "order"}); err != nil {
		t.Fatal(err)
	}
	if err := WechatSubmitPages(ctx, "at", validSubmitPagesRequest()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := WechatFetchAccessTokenTry3Time(ctx, "a", "b"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := WechatFetchAccessToken(ctx, "a", "b"); err != nil {
		t.Fatal(err)
	}
	if err := WechatPostTemplate(ctx, "at", `{}`, "", "t", "u"); err != nil {
		t.Fatal(err)
	}
	if err := WechatPostTemplateDirectly(ctx, "at", `{}`); err != nil {
		t.Fatal(err)
	}
	if _, err := WechatFetchJsapiTicket(ctx, "at"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := WechatFetchSnsAccessToken(ctx, "a", "b", "c"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := WechatFetchSnsUserInfo(ctx, "at", "oid"); err != nil {
		t.Fatal(err)
	}
	if _, err := WechatFetchSnsRefreshToken(ctx, "a", "rt"); err != nil {
		t.Fatal(err)
	}
	if _, err := WechatFetchWxSign(ctx, "a", "b", "https://x"); err != nil {
		t.Fatal(err)
	}
	if _, err := WechatFetchWxaCodeUnlimited(ctx, "at", WxaCodeUnlimitedRequest{Scene: "id=1"}); err != nil {
		t.Fatal(err)
	}
}

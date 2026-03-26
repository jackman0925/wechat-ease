package wechatease

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClientOptions(t *testing.T) {
	hc := &http.Client{Timeout: 2 * time.Second}
	c := NewClient(
		WithBaseURL("https://api.test.com/"),
		WithHTTPClient(hc),
		WithMaxRetries(5),
		WithRetryInterval(20*time.Millisecond),
	)
	if c.baseURL != "https://api.test.com" {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
	if c.maxRetries != 5 {
		t.Fatalf("unexpected retries: %d", c.maxRetries)
	}
	if c.httpClient.Timeout != 2*time.Second {
		t.Fatalf("unexpected timeout: %s", c.httpClient.Timeout)
	}
	if c.retryInterval != 20*time.Millisecond {
		t.Fatalf("unexpected retry interval: %s", c.retryInterval)
	}
}

func TestAPIError(t *testing.T) {
	err := (&BaseResponse{ErrCode: 40029, ErrMsg: "invalid code"}).Check()
	if err == nil || err.Error() != "wechat error: errcode=40029, errmsg=invalid code" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestFetchUserOpenID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sns/jscode2session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"openid":"oid","unionid":"uid","session_key":"sk"}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	oid, uid, sk, err := c.FetchUserOpenID(context.Background(), "appid", "sec", "code")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if oid != "oid" || uid != "uid" || sk != "sk" {
		t.Fatalf("unexpected values: %s %s %s", oid, uid, sk)
	}
}

func TestFetchAccessTokenWithRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			_, _ = w.Write([]byte(`{"errcode":-1,"errmsg":"system busy"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"ok-token","expires_in":7200}`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithMaxRetries(3),
		WithRetryInterval(5*time.Millisecond),
	)
	token, exp, err := c.FetchAccessTokenWithRetry(context.Background(), "app", "sec")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if token != "ok-token" || exp != 7200 {
		t.Fatalf("unexpected result: %s %d", token, exp)
	}
	if calls != 3 {
		t.Fatalf("unexpected call count: %d", calls)
	}
}

func TestCheckSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wxa/checksession" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			q := r.URL.Query()
			if q.Get("access_token") != "at" || q.Get("openid") != "oid" || q.Get("signature") != "sig" || q.Get("sig_method") != "hmac_sha256" {
				t.Fatalf("unexpected query: %v", q)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
		if err := c.CheckSession(context.Background(), "at", "oid", "sig", "hmac_sha256"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("wechat_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":87009,"errmsg":"invalid signature"}`))
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
		if err := c.CheckSession(context.Background(), "at", "oid", "sig", "hmac_sha256"); err == nil || !strings.Contains(err.Error(), "errcode=87009") {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("invalid_params", func(t *testing.T) {
		c := NewClient()
		if err := c.CheckSession(context.Background(), "", "oid", "sig", "hmac_sha256"); err == nil {
			t.Fatalf("expected error for empty access_token")
		}
		if err := c.CheckSession(context.Background(), "at", "", "sig", "hmac_sha256"); err == nil {
			t.Fatalf("expected error for empty openid")
		}
		if err := c.CheckSession(context.Background(), "at", "oid", "", "hmac_sha256"); err == nil {
			t.Fatalf("expected error for empty signature")
		}
		if err := c.CheckSession(context.Background(), "at", "oid", "sig", ""); err == nil {
			t.Fatalf("expected error for empty sig_method")
		}
		if err := c.CheckSession(context.Background(), "at", "oid", "sig", "md5"); err == nil {
			t.Fatalf("expected error for unsupported sig_method")
		}
	})
}

func TestFetchAccessTokenWithRetryFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":-1,"errmsg":"busy"}`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithMaxRetries(2),
		WithRetryInterval(5*time.Millisecond),
	)
	_, _, err := c.FetchAccessTokenWithRetry(context.Background(), "app", "sec")
	if err == nil || !strings.Contains(err.Error(), "after 2 attempts") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestPostTemplate(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/message/template/send" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		got = string(body)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	err := c.PostTemplate(context.Background(), "at", `{"k":{"value":"v"}}`, "https://x", "tmpl", "u1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if m["touser"] != "u1" || m["template_id"] != "tmpl" {
		t.Fatalf("unexpected request body: %s", got)
	}
}

func TestPostTemplateInvalidJSON(t *testing.T) {
	c := NewClient()
	err := c.PostTemplate(context.Background(), "at", `{"k":`, "", "t", "u")
	if err == nil || !strings.Contains(err.Error(), "not valid json") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestFetchJsapiTicket(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0,"ticket":"ticket123","expires_in":7200}`))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	ticket, err := c.FetchJsapiTicket(context.Background(), "at")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ticket != "ticket123" {
		t.Fatalf("unexpected ticket: %s", ticket)
	}
}

func TestFetchSnsFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sns/oauth2/access_token":
			_, _ = w.Write([]byte(`{"access_token":"at","expires_in":7200,"refresh_token":"rt","openid":"oid","unionid":"uid"}`))
		case "/sns/userinfo":
			_, _ = w.Write([]byte(`{"openid":"oid","nickname":"nick","headimgurl":"img"}`))
		case "/sns/oauth2/refresh_token":
			_, _ = w.Write([]byte(`{"access_token":"new-at","expires_in":7200}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	at, oid, rt, uid, err := c.FetchSnsAccessToken(context.Background(), "app", "sec", "code")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if at != "at" || oid != "oid" || rt != "rt" || uid != "uid" {
		t.Fatalf("unexpected sns access token result")
	}

	nick, img, err := c.FetchSnsUserInfo(context.Background(), "at", "oid")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if nick != "nick" || img != "img" {
		t.Fatalf("unexpected user info")
	}

	newAT, err := c.FetchSnsRefreshToken(context.Background(), "app", "rt")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if newAT != "new-at" {
		t.Fatalf("unexpected refresh token result: %s", newAT)
	}
}

func TestFetchWxSign(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			_, _ = w.Write([]byte(`{"access_token":"at","expires_in":7200}`))
		case "/cgi-bin/ticket/getticket":
			_, _ = w.Write([]byte(`{"errcode":0,"ticket":"tk","expires_in":7200}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	sign, err := c.FetchWxSign(context.Background(), "appid", "secret", "https://example.com/page")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sign.AppID != "appid" || sign.URL != "https://example.com/page" || sign.Signature == "" || sign.NonceStr == "" || sign.Timestamp == "" {
		t.Fatalf("unexpected sign result: %+v", sign)
	}
}

func TestFetchWxaCodeUnlimited(t *testing.T) {
	t.Run("success_binary", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wxa/getwxacodeunlimit" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("access_token") != "at" {
				t.Fatalf("unexpected access token: %s", r.URL.Query().Get("access_token"))
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"scene":"order=123"`) {
				t.Fatalf("unexpected request body: %s", string(body))
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x01, 0x02})
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
		img, err := c.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{
			Scene:  "order=123",
			Page:   "pages/index/index",
			Width:  430,
			EnvVer: "release",
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(img) == 0 || img[0] != 0x89 || img[1] != 0x50 {
			t.Fatalf("unexpected image bytes: %v", img)
		}
	})

	t.Run("wechat_error_json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":41030,"errmsg":"invalid page hint"}`))
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
		_, err := c.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{
			Scene: "order=123",
			Page:  "bad-page",
		})
		if err == nil || !strings.Contains(err.Error(), "errcode=41030") {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("scene_required", func(t *testing.T) {
		c := NewClient()
		_, err := c.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{})
		if err == nil || !strings.Contains(err.Error(), "scene is required") {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("scene_too_long", func(t *testing.T) {
		c := NewClient()
		_, err := c.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{
			Scene: strings.Repeat("a", 33),
		})
		if err == nil || !strings.Contains(err.Error(), "scene length must be <= 32") {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("width_out_of_range", func(t *testing.T) {
		c := NewClient()
		_, err := c.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{
			Scene: "ok=1",
			Width: 200,
		})
		if err == nil || !strings.Contains(err.Error(), "width must be between 280 and 1280") {
			t.Fatalf("unexpected err: %v", err)
		}
	})
}

func TestHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, _, err := c.FetchAccessToken(context.Background(), "a", "b")
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := c.FetchAccessToken(ctx, "a", "b")
	if err == nil {
		t.Fatalf("expect timeout err")
	}
}

func TestErrorInterceptor(t *testing.T) {
	intercepted := errors.New("intercepted")
	c := NewClient(WithErrorInterceptor(func(ctx context.Context, err error) error {
		if err == nil {
			return nil
		}
		return intercepted
	}))
	_, _, err := c.FetchAccessToken(context.Background(), "a", "b")
	if !errors.Is(err, intercepted) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHelpers(t *testing.T) {
	s, err := generateNonce(12)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(s) != 12 {
		t.Fatalf("unexpected nonce length: %d", len(s))
	}
	if _, err := generateNonce(0); err == nil {
		t.Fatalf("expect error for zero length")
	}
	if got := sha1Hex("hello"); got != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("unexpected sha1: %s", got)
	}
}

func TestGlobalWrappers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			_, _ = w.Write([]byte(`{"access_token":"at","expires_in":7200}`))
		case "/cgi-bin/ticket/getticket":
			_, _ = w.Write([]byte(`{"errcode":0,"ticket":"tk","expires_in":7200}`))
		case "/sns/jscode2session":
			_, _ = w.Write([]byte(`{"errcode":0,"openid":"oid","unionid":"uid","session_key":"sk"}`))
		case "/cgi-bin/message/template/send":
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		case "/sns/oauth2/access_token":
			_, _ = w.Write([]byte(`{"access_token":"at","expires_in":7200,"refresh_token":"rt","openid":"oid","unionid":"uid"}`))
		case "/sns/userinfo":
			_, _ = w.Write([]byte(`{"openid":"oid","nickname":"nick","headimgurl":"img"}`))
		case "/sns/oauth2/refresh_token":
			_, _ = w.Write([]byte(`{"access_token":"new-at","expires_in":7200}`))
		case "/wxa/getwxacodeunlimit":
			_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47})
		case "/wxa/checksession":
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	old := defaultClient
	defaultClient = NewClient(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithRetryInterval(1*time.Millisecond),
	)
	defer func() { defaultClient = old }()

	if _, _, _, err := WechatFetchUserOpenId(context.Background(), "a", "b", "c"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, _, err := WechatFetchAccessTokenTry3Time(context.Background(), "a", "b"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, _, err := WechatFetchAccessToken(context.Background(), "a", "b"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := WechatPostTemplate(context.Background(), "at", `{"k":{"value":"v"}}`, "https://x", "t", "u"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := WechatPostTemplateDirectly(context.Background(), "at", `{"touser":"u","template_id":"t","data":{"k":{"value":"v"}}}`); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := WechatFetchJsapiTicket(context.Background(), "at"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, _, _, _, err := WechatFetchSnsAccessToken(context.Background(), "a", "b", "c"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, _, err := WechatFetchSnsUserInfo(context.Background(), "at", "oid"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := WechatFetchSnsRefreshToken(context.Background(), "a", "rt"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := WechatFetchWxSign(context.Background(), "a", "b", "https://x"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := WechatFetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{Scene: "id=1"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := WechatCheckSession(context.Background(), "at", "oid", "sig", "hmac_sha256"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

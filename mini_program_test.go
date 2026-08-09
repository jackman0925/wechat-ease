package wechatease

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchUserOpenID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/sns/jscode2session" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"openid":"oid","unionid":"uid","session_key":"sk"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		oid, uid, key, err := client.FetchUserOpenID(context.Background(), "app", "secret", "code")
		if err != nil || oid != "oid" || uid != "uid" || key != "sk" {
			t.Fatalf("unexpected result: %s, %s, %s, %v", oid, uid, key, err)
		}
	})

	t.Run("wechat error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		_, _, _, err := client.FetchUserOpenID(context.Background(), "app", "secret", "code")
		if err == nil || !strings.Contains(err.Error(), "errcode=40029") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestFetchWxaCodeUnlimited(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wxa/getwxacodeunlimit" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47})
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		image, err := client.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{Scene: "order=123", Width: 430})
		if err != nil || len(image) != 4 {
			t.Fatalf("unexpected image response: %v, %v", image, err)
		}
	})

	t.Run("wechat error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":41030,"errmsg":"invalid page"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		_, err := client.FetchWxaCodeUnlimited(context.Background(), "at", WxaCodeUnlimitedRequest{Scene: "order=123"})
		if err == nil || !strings.Contains(err.Error(), "errcode=41030") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validation", func(t *testing.T) {
		client := NewClient()
		cases := []WxaCodeUnlimitedRequest{
			{},
			{Scene: strings.Repeat("a", 33)},
			{Scene: "ok", Width: 200},
		}
		for _, req := range cases {
			if _, err := client.FetchWxaCodeUnlimited(context.Background(), "at", req); err == nil {
				t.Fatalf("expected validation error for %+v", req)
			}
		}
	})
}

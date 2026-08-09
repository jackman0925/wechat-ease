package wechatease

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wxa/checksession" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			q := r.URL.Query()
			if q.Get("access_token") != "at" || q.Get("openid") != "oid" || q.Get("signature") != "sig" || q.Get("sig_method") != "hmac_sha256" {
				t.Fatalf("unexpected query: %v", q)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		if err := client.CheckSession(context.Background(), "at", "oid", "sig", "hmac_sha256"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wechat error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":87009,"errmsg":"invalid signature"}`))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		err := client.CheckSession(context.Background(), "at", "oid", "sig", "hmac_sha256")
		if err == nil || !strings.Contains(err.Error(), "errcode=87009") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		client := NewClient()
		cases := [][]string{
			{"", "oid", "sig", "hmac_sha256"},
			{"at", "", "sig", "hmac_sha256"},
			{"at", "oid", "", "hmac_sha256"},
			{"at", "oid", "sig", ""},
			{"at", "oid", "sig", "md5"},
		}
		for _, args := range cases {
			if err := client.CheckSession(context.Background(), args[0], args[1], args[2], args[3]); err == nil {
				t.Fatalf("expected error for args: %v", args)
			}
		}
	})
}

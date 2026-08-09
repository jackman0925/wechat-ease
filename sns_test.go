package wechatease

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchSnsAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","openid":"oid","unionid":"uid"}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	at, oid, rt, uid, err := client.FetchSnsAccessToken(context.Background(), "app", "secret", "code")
	if err != nil || at != "at" || oid != "oid" || rt != "rt" || uid != "uid" {
		t.Fatalf("unexpected result: %s, %s, %s, %s, %v", at, oid, rt, uid, err)
	}
}

func TestFetchSnsUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"openid":"oid","nickname":"nick","headimgurl":"img"}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	nick, image, err := client.FetchSnsUserInfo(context.Background(), "at", "oid")
	if err != nil || nick != "nick" || image != "img" {
		t.Fatalf("unexpected result: %s, %s, %v", nick, image, err)
	}
}

func TestFetchSnsRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"access_token":"new-at"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		token, err := client.FetchSnsRefreshToken(context.Background(), "app", "rt")
		if err != nil || token != "new-at" {
			t.Fatalf("unexpected result: %s, %v", token, err)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"expires_in":7200}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		_, err := client.FetchSnsRefreshToken(context.Background(), "app", "rt")
		if err == nil || !strings.Contains(err.Error(), "access_token empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

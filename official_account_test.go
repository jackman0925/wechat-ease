package wechatease

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":7200}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	token, expires, err := client.FetchAccessToken(context.Background(), "app", "secret")
	if err != nil || token != "token" || expires != 7200 {
		t.Fatalf("unexpected result: %s, %d, %v", token, expires, err)
	}
}

func TestFetchAccessTokenWithRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			_, _ = w.Write([]byte(`{"errcode":-1,"errmsg":"busy"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":7200}`))
	}))
	defer server.Close()
	client := NewClient(
		WithBaseURL(server.URL), WithHTTPClient(server.Client()),
		WithMaxRetries(3), WithRetryInterval(time.Millisecond),
	)
	token, _, err := client.FetchAccessTokenWithRetry(context.Background(), "app", "secret")
	if err != nil || token != "token" || calls.Load() != 3 {
		t.Fatalf("unexpected retry result: %s, %d, %v", token, calls.Load(), err)
	}
}

func TestPostTemplate(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err := client.PostTemplate(context.Background(), "at", `{"k":{"value":"v"}}`, "https://x", "tmpl", "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil || payload["touser"] != "u1" {
		t.Fatalf("unexpected body: %s, %v", body, err)
	}
	if err := client.PostTemplate(context.Background(), "at", `{"broken":`, "", "t", "u"); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestPostTemplateDirectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/message/template/send" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err := client.PostTemplateDirectly(context.Background(), "at", []byte(`{"data":{}}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchJsapiTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0,"ticket":"ticket123","expires_in":7200}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	ticket, err := client.FetchJsapiTicket(context.Background(), "at")
	if err != nil || ticket != "ticket123" {
		t.Fatalf("unexpected result: %s, %v", ticket, err)
	}
}

func TestFetchWxSign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			_, _ = w.Write([]byte(`{"access_token":"at","expires_in":7200}`))
		case "/cgi-bin/ticket/getticket":
			_, _ = w.Write([]byte(`{"errcode":0,"ticket":"tk","expires_in":7200}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	sign, err := client.FetchWxSign(context.Background(), "appid", "secret", "https://example.com")
	if err != nil || sign.Signature == "" || sign.NonceStr == "" {
		t.Fatalf("unexpected sign result: %+v, %v", sign, err)
	}
}

func TestOfficialAccountHelpers(t *testing.T) {
	nonce, err := generateNonce(12)
	if err != nil || len(nonce) != 12 {
		t.Fatalf("unexpected nonce: %q, %v", nonce, err)
	}
	if _, err := generateNonce(0); err == nil {
		t.Fatal("expected nonce length error")
	}
	if got := sha1Hex("hello"); got != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("unexpected SHA1: %s", got)
	}
}

func TestFetchAccessTokenWithRetryFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":-1,"errmsg":"busy"}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithMaxRetries(2), WithRetryInterval(time.Millisecond))
	_, _, err := client.FetchAccessTokenWithRetry(context.Background(), "app", "secret")
	if err == nil || !strings.Contains(err.Error(), "after 2 attempts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

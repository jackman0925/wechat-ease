package wechatease

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClientOptions(t *testing.T) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	client := NewClient(
		WithBaseURL("https://api.test.com/"),
		WithHTTPClient(httpClient),
		WithMaxRetries(5),
		WithRetryInterval(20*time.Millisecond),
		WithTimeout(3*time.Second),
	)
	if client.baseURL != "https://api.test.com" || client.maxRetries != 5 {
		t.Fatalf("unexpected client options: %+v", client)
	}
	if client.httpClient.Timeout != 3*time.Second || client.retryInterval != 20*time.Millisecond {
		t.Fatalf("unexpected timing options: %+v", client)
	}
}

func TestAPIErrorError(t *testing.T) {
	err := &APIError{Code: 40029, Msg: "invalid code"}
	if got := err.Error(); got != "wechat error: errcode=40029, errmsg=invalid code" {
		t.Fatalf("unexpected error text: %s", got)
	}
}

func TestBaseResponseCheck(t *testing.T) {
	if err := (BaseResponse{}).Check(); err != nil {
		t.Fatalf("unexpected success error: %v", err)
	}
	var apiErr *APIError
	err := (BaseResponse{ErrCode: 1, ErrMsg: "failed"}).Check()
	if !errors.As(err, &apiErr) || apiErr.Code != 1 {
		t.Fatalf("unexpected API error: %v", err)
	}
}

func TestRequestErrorError(t *testing.T) {
	err := &requestError{cause: errors.New("secret URL")}
	if err.Error() != "wechat request failed" {
		t.Fatalf("unexpected request error: %v", err)
	}
}

func TestRequestErrorUnwrap(t *testing.T) {
	cause := errors.New("cause")
	err := &requestError{cause: cause}
	if !errors.Is(err, cause) {
		t.Fatal("request error did not unwrap its cause")
	}
}

func TestClientWrapError(t *testing.T) {
	want := errors.New("wrapped")
	client := NewClient(WithErrorInterceptor(func(ctx context.Context, err error) error {
		return want
	}))
	if got := client.wrapError(context.Background(), errors.New("source")); !errors.Is(got, want) {
		t.Fatalf("unexpected intercepted error: %v", got)
	}
	if client.wrapError(context.Background(), nil) != nil {
		t.Fatal("nil error should remain nil")
	}
}

func TestClientDoJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	var resp BaseResponse
	if err := client.doJSON(context.Background(), http.MethodGet, "/test", nil, nil, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoJSONHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	var resp BaseResponse
	err := client.doJSON(context.Background(), http.MethodGet, "/test", nil, nil, &resp)
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoBinary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte{1, 2, 3})
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	body, err := client.doBinary(context.Background(), http.MethodGet, "/binary", nil, nil)
	if err != nil || len(body) != 3 {
		t.Fatalf("unexpected binary response: %v, %v", body, err)
	}
}

func TestClientShouldRetry(t *testing.T) {
	client := NewClient()
	if !client.shouldRetry(&APIError{Code: -1, Msg: "busy"}) {
		t.Fatal("system busy should be retryable")
	}
	if client.shouldRetry(&APIError{Code: 40001, Msg: "invalid"}) {
		t.Fatal("credential error should not be retryable")
	}
}

func TestClientSleepRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := NewClient(WithRetryInterval(time.Millisecond))
	if err := client.sleepRetry(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestSafeRequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := safeRequestError(ctx, errors.New("source")); !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected context error: %v", err)
	}
}

func TestRedactQueryValues(t *testing.T) {
	query := url.Values{"access_token": {"token +"}}
	got := redactQueryValues("token + token+%2B", query)
	if strings.Contains(got, "token") {
		t.Fatalf("secret was not redacted: %s", got)
	}
}

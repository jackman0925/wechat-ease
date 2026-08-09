package wechatease

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSubmitPages(t *testing.T) {
	const token = "token +/&?"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/wxa/search/wxaapi_submitpages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "access_token=token+%2B%2F%26%3F" {
			t.Error("access_token was not encoded correctly")
		}
		if r.URL.Query().Get("access_token") != token {
			t.Error("unexpected access_token")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected Content-Type: %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}
		if strings.Contains(string(body), `"query":""`) {
			t.Errorf("empty query should be omitted: %s", body)
		}
		var req SubmitPagesRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if len(req.Pages) != 2 {
			t.Errorf("unexpected pages count: %d", len(req.Pages))
			return
		}
		if req.Pages[0].Path != "pages/content/detail/index" || req.Pages[0].Query != "slug=birthday-wishes-for-friend" {
			t.Errorf("unexpected first page: %+v", req.Pages[0])
		}
		if req.Pages[1].Path != "pages/index/index" || req.Pages[1].Query != "" {
			t.Errorf("unexpected second page: %+v", req.Pages[1])
		}

		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	err := client.SubmitPages(context.Background(), "  "+token+"  ", SubmitPagesRequest{
		Pages: []SubmitPage{
			{Path: "  pages/content/detail/index  ", Query: "  slug=birthday-wishes-for-friend  "},
			{Path: " pages/index/index "},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubmitPagesWechatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	err := client.SubmitPages(context.Background(), "token", validSubmitPagesRequest())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.Code != 40013 || apiErr.Msg != "invalid appid" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func TestSubmitPagesValidation(t *testing.T) {
	tests := []struct {
		name  string
		token string
		req   SubmitPagesRequest
		want  string
	}{
		{name: "empty token", token: "  ", req: validSubmitPagesRequest(), want: "access_token is required"},
		{name: "empty pages", token: "token", req: SubmitPagesRequest{}, want: "pages must contain at least one item"},
		{name: "empty path", token: "token", req: SubmitPagesRequest{Pages: []SubmitPage{{Path: "  "}}}, want: "pages[0].path is required"},
		{name: "leading slash", token: "token", req: SubmitPagesRequest{Pages: []SubmitPage{{Path: "/pages/index/index"}}}, want: "must not start with /"},
		{name: "path query", token: "token", req: SubmitPagesRequest{Pages: []SubmitPage{{Path: "pages/index/index?a=1"}}}, want: "must not contain ? or #"},
		{name: "path fragment", token: "token", req: SubmitPagesRequest{Pages: []SubmitPage{{Path: "pages/index/index#top"}}}, want: "must not contain ? or #"},
		{name: "query prefix", token: "token", req: SubmitPagesRequest{Pages: []SubmitPage{{Path: "pages/index/index", Query: "?a=1"}}}, want: "query must not start with ?"},
		{name: "query fragment", token: "token", req: SubmitPagesRequest{Pages: []SubmitPage{{Path: "pages/index/index", Query: "a=1#top"}}}, want: "query must not contain #"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			client := NewClient(WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("network request should not run")
			})}))

			err := client.SubmitPages(context.Background(), tt.token, tt.req)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if calls.Load() != 0 {
				t.Fatalf("validation made %d network requests", calls.Load())
			}
		})
	}
}

func TestSubmitPagesRemoteFailures(t *testing.T) {
	const token = "sensitive-access-token"

	t.Run("http status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("request failed for " + token))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		err := client.SubmitPages(context.Background(), token, validSubmitPagesRequest())
		assertSubmitPagesFailure(t, err, token)
		if !strings.Contains(err.Error(), "status 502") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":`))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		err := client.SubmitPages(context.Background(), token, validSubmitPagesRequest())
		assertSubmitPagesFailure(t, err, token)
		if !strings.Contains(err.Error(), "decode wechat response failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(30 * time.Millisecond)
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		err := client.SubmitPages(ctx, token, validSubmitPagesRequest())
		assertSubmitPagesFailure(t, err, token)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		client := NewClient()
		err := client.SubmitPages(ctx, token, validSubmitPagesRequest())
		assertSubmitPagesFailure(t, err, token)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})

	t.Run("transport error", func(t *testing.T) {
		client := NewClient(WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("transport failed for %s", req.URL.String())
		})}))
		err := client.SubmitPages(context.Background(), token, validSubmitPagesRequest())
		assertSubmitPagesFailure(t, err, token)
		if err.Error() != "wechat request failed" {
			t.Fatalf("unexpected transport error: %v", err)
		}
	})
}

func TestSubmitPagesErrorInterceptor(t *testing.T) {
	const token = "interceptor-sensitive-token"
	var messages []string
	interceptor := func(ctx context.Context, err error) error {
		messages = append(messages, err.Error())
		return err
	}

	client := NewClient(WithErrorInterceptor(interceptor))
	if err := client.SubmitPages(context.Background(), token, SubmitPagesRequest{}); err == nil {
		t.Fatal("expected local validation error")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40001,"errmsg":"invalid ` + token + `"}`))
	}))
	defer server.Close()
	client = NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithErrorInterceptor(interceptor),
	)
	if err := client.SubmitPages(context.Background(), token, validSubmitPagesRequest()); err == nil {
		t.Fatal("expected remote error")
	}

	if len(messages) != 2 {
		t.Fatalf("interceptor called %d times, want 2", len(messages))
	}
	for _, message := range messages {
		if strings.Contains(message, token) {
			t.Fatalf("interceptor received access_token: %s", message)
		}
	}
}

func TestWechatSubmitPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wxa/search/wxaapi_submitpages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	original := defaultClient
	defaultClient = NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	defer func() { defaultClient = original }()

	if err := WechatSubmitPages(context.Background(), "token", validSubmitPagesRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func validSubmitPagesRequest() SubmitPagesRequest {
	return SubmitPagesRequest{Pages: []SubmitPage{{Path: "pages/index/index", Query: "from=search"}}}
}

func assertSubmitPagesFailure(t *testing.T, err error, token string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked access_token: %v", err)
	}
}

package wechatease

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRefundOrder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/xpay/refund_order" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			q := r.URL.Query()
			if q.Get("access_token") != "at" || q.Get("pay_sig") != "psig" {
				t.Fatalf("unexpected query: %v", q)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"refund_order_id":"refund1234"`) {
				t.Fatalf("unexpected body: %s", body)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok","refund_order_id":"refund1234","refund_wx_order_id":"rwx","pay_order_id":"pay1","pay_wx_order_id":"pwx"}`))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		resp, err := client.RefundOrder(context.Background(), "at", "psig", validRefundRequest())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.RefundOrderID != "refund1234" || resp.PayOrderID != "pay1" {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("wechat error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":40001,"errmsg":"invalid credential"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		_, err := client.RefundOrder(context.Background(), "at", "psig", validRefundRequest())
		if err == nil || !strings.Contains(err.Error(), "errcode=40001") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		client := NewClient()
		cases := []struct {
			token, sig string
			req        RefundOrderRequest
		}{
			{"", "psig", validRefundRequest()},
			{"at", "", validRefundRequest()},
			{"at", "psig", RefundOrderRequest{}},
			{"at", "psig", RefundOrderRequest{OpenID: "oid"}},
			{"at", "psig", RefundOrderRequest{OpenID: "oid", RefundOrderID: "short"}},
			{"at", "psig", RefundOrderRequest{OpenID: "oid", RefundOrderID: "refund1234"}},
			{"at", "psig", RefundOrderRequest{OpenID: "oid", OrderID: "order", RefundOrderID: "refund1234", RefundFee: 1}},
			{"at", "psig", RefundOrderRequest{OpenID: "oid", OrderID: "order", RefundOrderID: "refund1234", LeftFee: 10}},
			{"at", "psig", RefundOrderRequest{OpenID: "oid", OrderID: "order", RefundOrderID: "refund1234", LeftFee: 10, RefundFee: 20}},
		}
		for _, tt := range cases {
			if _, err := client.RefundOrder(context.Background(), tt.token, tt.sig, tt.req); err == nil {
				t.Fatalf("expected validation error for %+v", tt)
			}
		}
	})
}

func TestNotifyProvideGoods(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/xpay/notify_provide_goods" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		if err := client.NotifyProvideGoods(context.Background(), "at", "psig", NotifyProvideGoodsRequest{OrderID: "order123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wechat error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":40001,"errmsg":"invalid credential"}`))
		}))
		defer server.Close()
		client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
		err := client.NotifyProvideGoods(context.Background(), "at", "psig", NotifyProvideGoodsRequest{OrderID: "order123"})
		if err == nil || !strings.Contains(err.Error(), "errcode=40001") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		client := NewClient()
		if err := client.NotifyProvideGoods(context.Background(), "", "psig", NotifyProvideGoodsRequest{OrderID: "order"}); err == nil {
			t.Fatal("expected access_token validation error")
		}
		if err := client.NotifyProvideGoods(context.Background(), "at", "", NotifyProvideGoodsRequest{OrderID: "order"}); err == nil {
			t.Fatal("expected pay_sig validation error")
		}
		if err := client.NotifyProvideGoods(context.Background(), "at", "psig", NotifyProvideGoodsRequest{}); err == nil {
			t.Fatal("expected order validation error")
		}
	})
}

func validRefundRequest() RefundOrderRequest {
	return RefundOrderRequest{
		OpenID: "oid", OrderID: "order123", RefundOrderID: "refund1234",
		LeftFee: 100, RefundFee: 50, RefundReason: 1, ReqFrom: 2,
	}
}

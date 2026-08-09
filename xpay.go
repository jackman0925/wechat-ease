package wechatease

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RefundOrder 发起微信虚拟支付退款任务。
// 接口仅启动退款任务，需后续通过 query_order 查询退款状态。
func (c *Client) RefundOrder(ctx context.Context, accessToken, paySig string, req RefundOrderRequest) (*RefundOrderResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("access_token is required"))
	}
	if strings.TrimSpace(paySig) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("pay_sig is required"))
	}
	if strings.TrimSpace(req.OpenID) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("openid is required"))
	}
	if strings.TrimSpace(req.RefundOrderID) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_order_id is required"))
	}
	if l := len(req.RefundOrderID); l < 8 || l > 32 {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_order_id length must be between 8 and 32"))
	}
	if strings.TrimSpace(req.OrderID) == "" && strings.TrimSpace(req.WxOrderID) == "" {
		return nil, c.wrapError(ctx, fmt.Errorf("order_id or wx_order_id is required"))
	}
	if req.LeftFee <= 0 {
		return nil, c.wrapError(ctx, fmt.Errorf("left_fee must be greater than 0"))
	}
	if req.RefundFee <= 0 {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_fee must be greater than 0"))
	}
	if req.RefundFee > req.LeftFee {
		return nil, c.wrapError(ctx, fmt.Errorf("refund_fee must be <= left_fee"))
	}

	query := url.Values{"access_token": {accessToken}, "pay_sig": {paySig}}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, c.wrapError(ctx, fmt.Errorf("marshal refund_order request failed: %w", err))
	}
	var resp RefundOrderResponse
	if err := c.doJSON(ctx, http.MethodPost, "/xpay/refund_order", query, body, &resp); err != nil {
		return nil, err
	}
	if err := c.wrapError(ctx, resp.Check()); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NotifyProvideGoods 通知已经发货完成。
// 仅在 xpay_goods_deliver_notify 推送失败的异常场景下使用。
func (c *Client) NotifyProvideGoods(ctx context.Context, accessToken, paySig string, req NotifyProvideGoodsRequest) error {
	if strings.TrimSpace(accessToken) == "" {
		return c.wrapError(ctx, fmt.Errorf("access_token is required"))
	}
	if strings.TrimSpace(paySig) == "" {
		return c.wrapError(ctx, fmt.Errorf("pay_sig is required"))
	}
	if strings.TrimSpace(req.OrderID) == "" && strings.TrimSpace(req.WxOrderID) == "" {
		return c.wrapError(ctx, fmt.Errorf("order_id or wx_order_id is required"))
	}

	query := url.Values{"access_token": {accessToken}, "pay_sig": {paySig}}
	body, err := json.Marshal(req)
	if err != nil {
		return c.wrapError(ctx, fmt.Errorf("marshal notify_provide_goods request failed: %w", err))
	}
	var resp BaseResponse
	if err := c.doJSON(ctx, http.MethodPost, "/xpay/notify_provide_goods", query, body, &resp); err != nil {
		return err
	}
	return c.wrapError(ctx, resp.Check())
}

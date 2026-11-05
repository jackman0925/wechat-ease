package event

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// PenaltyViolationEvent 是 Event 为 wxa_punish_event 的小程序违规处罚通知。
// Detail 保留事件详情的原始 JSON，具体结构由 EventType 决定。
type PenaltyViolationEvent struct {
	BasePlatformEvent
	EventType      int             `json:"event_type"`
	PunishID       string          `json:"punish_id"`
	AppID          string          `json:"appid"`
	PunishTime     int64           `json:"punish_time"`
	IllegalReason  string          `json:"illegal_reason"`
	IllegalContent []string        `json:"illegal_content"`
	RuleName       string          `json:"rule_name"`
	RuleURL        string          `json:"rule_url"`
	AdjustGuideURL string          `json:"adjust_guide_url"`
	Detail         json.RawMessage `json:"detail"`
}

// ParsePenaltyViolationJSON 解析小程序违规处罚信息通知的 JSON 正文。
// 该函数仅处理内容结构，不负责 HTTP、验签或消息解密。
func ParsePenaltyViolationJSON(body []byte) (*PenaltyViolationEvent, error) {
	base, err := parseBasePlatformEvent(body)
	if err != nil {
		return nil, err
	}
	if base.Event != EventPenaltyViolation {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrUnexpectedEvent, base.Event, EventPenaltyViolation)
	}

	var payload struct {
		EventType      int             `json:"event_type"`
		PunishID       string          `json:"punish_id"`
		AppID          string          `json:"appid"`
		PunishTime     int64           `json:"punish_time"`
		IllegalReason  string          `json:"illegal_reason"`
		IllegalContent json.RawMessage `json:"illegal_content"`
		RuleName       string          `json:"rule_name"`
		RuleURL        string          `json:"rule_url"`
		AdjustGuideURL string          `json:"adjust_guide_url"`
		Detail         json.RawMessage `json:"detail"`
	}
	if err := unmarshalPlatformEventJSON(body, &payload); err != nil {
		return nil, err
	}
	illegalContent, err := parseStringList(payload.IllegalContent)
	if err != nil {
		return nil, fmt.Errorf("%w: illegal_content: %v", ErrInvalidPlatformEvent, err)
	}
	detail, err := parseDetailJSON(payload.Detail)
	if err != nil {
		return nil, fmt.Errorf("%w: detail: %v", ErrInvalidPlatformEvent, err)
	}

	return &PenaltyViolationEvent{
		BasePlatformEvent: base,
		EventType:         payload.EventType,
		PunishID:          payload.PunishID,
		AppID:             payload.AppID,
		PunishTime:        payload.PunishTime,
		IllegalReason:     payload.IllegalReason,
		IllegalContent:    illegalContent,
		RuleName:          payload.RuleName,
		RuleURL:           payload.RuleURL,
		AdjustGuideURL:    payload.AdjustGuideURL,
		Detail:            detail,
	}, nil
}

func parseStringList(data json.RawMessage) ([]string, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		return list, nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("must be a string or string array")
	}
	return []string{value}, nil
}

func parseDetailJSON(data json.RawMessage) (json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	var detail string
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("must be a JSON string")
	}
	if !json.Valid([]byte(detail)) {
		return nil, fmt.Errorf("contains invalid JSON")
	}
	return append(json.RawMessage(nil), detail...), nil
}

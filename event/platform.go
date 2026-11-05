// Package event 提供已验证通知正文的小程序平台事件解析能力。
// 调用方负责 HTTP 接收、验签和解密，本包不依赖任何 Web 框架。
package event

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrInvalidPlatformEvent 表示通知正文不是合法的平台事件 JSON。
	ErrInvalidPlatformEvent = errors.New("invalid platform event")
	// ErrMissingEventName 表示通知正文缺少用于路由的 Event 字段。
	ErrMissingEventName = errors.New("missing platform event name")
	// ErrUnexpectedEvent 表示专用解析函数收到了不匹配的事件类型。
	ErrUnexpectedEvent = errors.New("unexpected platform event")
)

// EventPenaltyViolation 是小程序违规处罚信息通知的事件名称。
const EventPenaltyViolation = "wxa_punish_event"

const (
	// EventUserInfoModified 表示平台已修改授权用户资料。
	EventUserInfoModified = "user_info_modified"
	// EventUserAuthorizationRevoke 表示用户撤回已授权的信息。
	EventUserAuthorizationRevoke = "user_authorization_revoke"
	// EventUserAuthorizationCancellation 表示授权用户已完成注销。
	EventUserAuthorizationCancellation = "user_authorization_cancellation"
)

// PlatformEvent 是所有已解析平台事件的公共接口。
type PlatformEvent interface {
	EventName() string
	RawJSON() json.RawMessage
}

// BasePlatformEvent 包含所有平台事件共享的通知字段。
type BasePlatformEvent struct {
	ToUserName   string `json:"ToUserName"`
	FromUserName string `json:"FromUserName"`
	CreateTime   int64  `json:"CreateTime"`
	MsgType      string `json:"MsgType"`
	Event        string `json:"Event"`

	raw json.RawMessage
}

// EventName 返回通知的 Event 字段，用于识别具体的平台事件类型。
func (event BasePlatformEvent) EventName() string {
	return event.Event
}

// RawJSON 返回通知正文的副本，避免调用方意外修改解析结果。
func (event BasePlatformEvent) RawJSON() json.RawMessage {
	return append(json.RawMessage(nil), event.raw...)
}

// UnknownPlatformEvent 表示库尚未提供强类型支持的平台事件。
// 该类型保留公共字段和原始正文，便于调用方自行处理新事件。
type UnknownPlatformEvent struct {
	BasePlatformEvent
}

// ParsePlatformEventJSON 解析已验签、解密后的小程序平台通知 JSON。
// 已支持的事件会返回对应强类型；未知事件返回 UnknownPlatformEvent。
func ParsePlatformEventJSON(body []byte) (PlatformEvent, error) {
	base, err := parseBasePlatformEvent(body)
	if err != nil {
		return nil, err
	}
	switch base.Event {
	case EventPenaltyViolation:
		return ParsePenaltyViolationJSON(body)
	case EventUserInfoModified, EventUserAuthorizationRevoke, EventUserAuthorizationCancellation:
		return ParseAuthorizedUserInformationChangeJSON(body)
	default:
		return &UnknownPlatformEvent{BasePlatformEvent: base}, nil
	}
}

func parseBasePlatformEvent(body []byte) (BasePlatformEvent, error) {
	if !json.Valid(body) {
		return BasePlatformEvent{}, ErrInvalidPlatformEvent
	}
	var event BasePlatformEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return BasePlatformEvent{}, fmt.Errorf("%w: decode common fields: %v", ErrInvalidPlatformEvent, err)
	}
	if event.Event == "" {
		return BasePlatformEvent{}, ErrMissingEventName
	}
	event.raw = append(json.RawMessage(nil), body...)
	return event, nil
}

func unmarshalPlatformEventJSON(body []byte, target any) error {
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("%w: decode event fields: %v", ErrInvalidPlatformEvent, err)
	}
	return nil
}

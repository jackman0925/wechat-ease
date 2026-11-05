package event

import "fmt"

// AuthorizedUserInformationChangeEvent 是授权用户资料变更、撤回授权或完成注销通知。
// Event 的合法值为 user_info_modified、user_authorization_revoke 和 user_authorization_cancellation。
type AuthorizedUserInformationChangeEvent struct {
	BasePlatformEvent
	OpenID     string `json:"OpenID"`
	AppID      string `json:"AppID"`
	RevokeInfo string `json:"RevokeInfo"`
	PluginID   string `json:"PluginID"`
	OpenPID    string `json:"OpenPID"`
}

// ParseAuthorizedUserInformationChangeJSON 解析授权用户信息变更通知的 JSON 正文。
// 该函数仅处理内容结构，不负责 HTTP、验签或消息解密。
func ParseAuthorizedUserInformationChangeJSON(body []byte) (*AuthorizedUserInformationChangeEvent, error) {
	base, err := parseBasePlatformEvent(body)
	if err != nil {
		return nil, err
	}
	if !isAuthorizedUserInformationChangeEvent(base.Event) {
		return nil, fmt.Errorf("%w: got %q", ErrUnexpectedEvent, base.Event)
	}

	var payload struct {
		OpenID     string `json:"OpenID"`
		AppID      string `json:"AppID"`
		RevokeInfo string `json:"RevokeInfo"`
		PluginID   string `json:"PluginID"`
		OpenPID    string `json:"OpenPID"`
	}
	if err := unmarshalPlatformEventJSON(body, &payload); err != nil {
		return nil, err
	}
	return &AuthorizedUserInformationChangeEvent{
		BasePlatformEvent: base,
		OpenID:            payload.OpenID,
		AppID:             payload.AppID,
		RevokeInfo:        payload.RevokeInfo,
		PluginID:          payload.PluginID,
		OpenPID:           payload.OpenPID,
	}, nil
}

func isAuthorizedUserInformationChangeEvent(event string) bool {
	switch event {
	case EventUserInfoModified, EventUserAuthorizationRevoke, EventUserAuthorizationCancellation:
		return true
	default:
		return false
	}
}

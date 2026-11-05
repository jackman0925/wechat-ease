package event

import (
	"errors"
	"testing"
)

const authorizedUserInformationChangeFixture = `{
  "ToUserName":"gh_870882ca4b1",
  "FromUserName":"oaKk346BaWE-eIn4oSRWbaM9vR7s",
  "CreateTime":1627359464,
  "MsgType":"event",
  "Event":"user_authorization_revoke",
  "OpenID":"oaKk343WOktAaT2ygsX138BGblrg",
  "AppID":"wx13974bf780d3dc89",
  "RevokeInfo":"1",
  "PluginID":"wx13974bf780d3dc89",
  "OpenPID":"G7esq5NVzP76HIHoB95t4CVBP6to"
}`

func TestParseAuthorizedUserInformationChangeJSON(t *testing.T) {
	event, err := ParseAuthorizedUserInformationChangeJSON([]byte(authorizedUserInformationChangeFixture))
	if err != nil {
		t.Fatalf("ParseAuthorizedUserInformationChangeJSON() error = %v", err)
	}
	if event.Event != EventUserAuthorizationRevoke || event.OpenID == "" || event.AppID == "" {
		t.Fatalf("ParseAuthorizedUserInformationChangeJSON() event = %+v", event)
	}
	if event.RevokeInfo != "1" || event.PluginID == "" || event.OpenPID == "" {
		t.Fatalf("ParseAuthorizedUserInformationChangeJSON() revoke fields = %+v", event)
	}
}

func TestParseAuthorizedUserInformationChangeJSONSupportedEvents(t *testing.T) {
	for _, eventName := range []string{
		EventUserInfoModified,
		EventUserAuthorizationRevoke,
		EventUserAuthorizationCancellation,
	} {
		t.Run(eventName, func(t *testing.T) {
			body := []byte(`{"Event":"` + eventName + `"}`)
			event, err := ParseAuthorizedUserInformationChangeJSON(body)
			if err != nil {
				t.Fatalf("ParseAuthorizedUserInformationChangeJSON() error = %v", err)
			}
			if event.Event != eventName {
				t.Fatalf("Event = %q, want %q", event.Event, eventName)
			}
		})
	}
}

func TestParseAuthorizedUserInformationChangeJSONUnexpectedEvent(t *testing.T) {
	if _, err := ParseAuthorizedUserInformationChangeJSON([]byte(`{"Event":"wxa_punish_event"}`)); !errors.Is(err, ErrUnexpectedEvent) {
		t.Fatalf("ParseAuthorizedUserInformationChangeJSON() error = %v", err)
	}
}

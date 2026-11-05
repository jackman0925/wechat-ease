package event

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestParsePlatformEventJSON(t *testing.T) {
	penalty, err := ParsePlatformEventJSON([]byte(penaltyFixture))
	if err != nil {
		t.Fatalf("ParsePlatformEventJSON() error = %v", err)
	}
	if _, ok := penalty.(*PenaltyViolationEvent); !ok {
		t.Fatalf("ParsePlatformEventJSON() type = %T, want *PenaltyViolationEvent", penalty)
	}
	authorizedUser, err := ParsePlatformEventJSON([]byte(authorizedUserInformationChangeFixture))
	if err != nil {
		t.Fatalf("ParsePlatformEventJSON() authorized user event error = %v", err)
	}
	if _, ok := authorizedUser.(*AuthorizedUserInformationChangeEvent); !ok {
		t.Fatalf("ParsePlatformEventJSON() type = %T, want *AuthorizedUserInformationChangeEvent", authorizedUser)
	}

	unknown, err := ParsePlatformEventJSON([]byte(`{"Event":"future_platform_event","MsgType":"event"}`))
	if err != nil {
		t.Fatalf("ParsePlatformEventJSON() unknown event error = %v", err)
	}
	if value, ok := unknown.(*UnknownPlatformEvent); !ok || value.Event != "future_platform_event" {
		t.Fatalf("ParsePlatformEventJSON() unknown event = %#v", unknown)
	}
}

func TestParsePlatformEventJSONInvalidPayload(t *testing.T) {
	if _, err := ParsePlatformEventJSON([]byte(`not-json`)); !errors.Is(err, ErrInvalidPlatformEvent) {
		t.Fatalf("ParsePlatformEventJSON() error = %v", err)
	}
	if _, err := ParsePlatformEventJSON([]byte(`{"MsgType":"event"}`)); !errors.Is(err, ErrMissingEventName) {
		t.Fatalf("ParsePlatformEventJSON() missing Event error = %v", err)
	}
}

func TestParsePlatformEvent(t *testing.T) {
	jsonEvent, err := ParsePlatformEvent([]byte(penaltyFixture))
	if err != nil {
		t.Fatalf("ParsePlatformEvent() JSON error = %v", err)
	}
	if _, ok := jsonEvent.(*PenaltyViolationEvent); !ok {
		t.Fatalf("ParsePlatformEvent() JSON type = %T, want *PenaltyViolationEvent", jsonEvent)
	}

	xmlEvent, err := ParsePlatformEvent([]byte(`
<xml>
  <ToUserName><![CDATA[gh_870882ca4b1]]></ToUserName>
  <FromUserName><![CDATA[oaKk346BaWE-eIn4oSRWbaM9vR7s]]></FromUserName>
  <CreateTime>1627359464</CreateTime>
  <MsgType><![CDATA[event]]></MsgType>
  <Event><![CDATA[user_authorization_revoke]]></Event>
  <OpenID><![CDATA[oaKk343WOktAaT2ygsX138BGblrg]]></OpenID>
  <AppID><![CDATA[wx13974bf780d3dc89]]></AppID>
  <RevokeInfo><![CDATA[1]]></RevokeInfo>
  <PluginID><![CDATA[wx13974bf780d3dc89]]></PluginID>
  <OpenPID><![CDATA[G7esq5NVzP76HIHoB95t4CVBP6to]]></OpenPID>
</xml>`))
	if err != nil {
		t.Fatalf("ParsePlatformEvent() XML error = %v", err)
	}
	authorizedUser, ok := xmlEvent.(*AuthorizedUserInformationChangeEvent)
	if !ok {
		t.Fatalf("ParsePlatformEvent() XML type = %T, want *AuthorizedUserInformationChangeEvent", xmlEvent)
	}
	if authorizedUser.OpenID == "" || authorizedUser.RevokeInfo != "1" {
		t.Fatalf("ParsePlatformEvent() XML event = %+v", authorizedUser)
	}
}

func TestParsePlatformEventXMLPenaltyViolation(t *testing.T) {
	event, err := ParsePlatformEvent([]byte(`
<xml>
  <Event><![CDATA[wxa_punish_event]]></Event>
  <event_type>1</event_type>
  <punish_id><![CDATA[649557]]></punish_id>
  <illegal_content><![CDATA[违规内容测试]]></illegal_content>
  <detail><![CDATA[{"warned_type":1,"warned_ban_days":[3]}]]></detail>
</xml>`))
	if err != nil {
		t.Fatalf("ParsePlatformEvent() XML penalty error = %v", err)
	}
	penalty, ok := event.(*PenaltyViolationEvent)
	if !ok {
		t.Fatalf("ParsePlatformEvent() XML type = %T, want *PenaltyViolationEvent", event)
	}
	if penalty.PunishID != "649557" || len(penalty.IllegalContent) != 1 || penalty.Detail == nil {
		t.Fatalf("ParsePlatformEvent() XML penalty = %+v", penalty)
	}
}

func TestParsePlatformEventInvalidFormat(t *testing.T) {
	if _, err := ParsePlatformEvent([]byte(`invalid`)); !errors.Is(err, ErrInvalidPlatformEvent) {
		t.Fatalf("ParsePlatformEvent() invalid format error = %v", err)
	}
	if _, err := ParsePlatformEvent([]byte(`{invalid`)); !errors.Is(err, ErrInvalidPlatformEvent) {
		t.Fatalf("ParsePlatformEvent() malformed JSON error = %v", err)
	}
	if _, err := ParsePlatformEvent([]byte(`<xml><Event>unknown</Event></xml>trailing`)); !errors.Is(err, ErrInvalidPlatformEvent) {
		t.Fatalf("ParsePlatformEvent() malformed XML error = %v", err)
	}
}

func TestBasePlatformEventEventName(t *testing.T) {
	event := BasePlatformEvent{Event: EventPenaltyViolation}
	if got := event.EventName(); got != EventPenaltyViolation {
		t.Fatalf("EventName() = %q, want %q", got, EventPenaltyViolation)
	}
}

func TestBasePlatformEventRawJSON(t *testing.T) {
	event := BasePlatformEvent{raw: json.RawMessage(`{"Event":"test"}`)}
	raw := event.RawJSON()
	raw[0] = '['
	if bytes.Equal(raw, event.RawJSON()) {
		t.Fatal("RawJSON() returned the internal slice")
	}
}

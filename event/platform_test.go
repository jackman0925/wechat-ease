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

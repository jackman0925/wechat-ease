package event

import (
	"encoding/json"
	"errors"
	"testing"
)

const penaltyFixture = `{
  "ToUserName":"gh_test",
  "FromUserName":"open_id",
  "CreateTime":1699803867,
  "MsgType":"event",
  "Event":"wxa_punish_event",
  "punish_id":"649557",
  "appid":"wx_test",
  "punish_time":1699803865,
  "illegal_reason":"存在诱导分享行为",
  "illegal_content":["违规内容测试"],
  "detail":"{\"warned_type\":1,\"rectify_deadline\":1699796571,\"warned_function_names\":[],\"warned_ban_days\":[3]}",
  "rule_url":"https://developers.weixin.qq.com/example",
  "rule_name":"运营规范",
  "adjust_guide_url":"https://mp.weixin.qq.com/example",
  "event_type":1
}`

func TestParsePenaltyViolationJSON(t *testing.T) {
	event, err := ParsePenaltyViolationJSON([]byte(penaltyFixture))
	if err != nil {
		t.Fatalf("ParsePenaltyViolationJSON() error = %v", err)
	}
	if event.PunishID != "649557" || event.AppID != "wx_test" || event.EventType != 1 {
		t.Fatalf("ParsePenaltyViolationJSON() event = %+v", event)
	}
	if len(event.IllegalContent) != 1 || event.IllegalContent[0] != "违规内容测试" {
		t.Fatalf("ParsePenaltyViolationJSON() illegal_content = %#v", event.IllegalContent)
	}
	var detail struct {
		WarnedType int `json:"warned_type"`
	}
	if err := json.Unmarshal(event.Detail, &detail); err != nil || detail.WarnedType != 1 {
		t.Fatalf("ParsePenaltyViolationJSON() detail = %s, error = %v", event.Detail, err)
	}
}

func TestParsePenaltyViolationJSONStringContent(t *testing.T) {
	body := []byte(`{"Event":"wxa_punish_event","illegal_content":"单条违规内容","detail":"{\"suspended_days\":1}"}`)
	event, err := ParsePenaltyViolationJSON(body)
	if err != nil {
		t.Fatalf("ParsePenaltyViolationJSON() error = %v", err)
	}
	if len(event.IllegalContent) != 1 || event.IllegalContent[0] != "单条违规内容" {
		t.Fatalf("ParsePenaltyViolationJSON() illegal_content = %#v", event.IllegalContent)
	}
}

func TestParsePenaltyViolationJSONInvalidPayload(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{name: "事件不匹配", body: `{"Event":"other_event"}`, want: ErrUnexpectedEvent},
		{name: "违规内容类型错误", body: `{"Event":"wxa_punish_event","illegal_content":1}`, want: ErrInvalidPlatformEvent},
		{name: "详情不是JSON字符串", body: `{"Event":"wxa_punish_event","detail":{}}`, want: ErrInvalidPlatformEvent},
		{name: "详情字符串不是JSON", body: `{"Event":"wxa_punish_event","detail":"not-json"}`, want: ErrInvalidPlatformEvent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParsePenaltyViolationJSON([]byte(tt.body)); !errors.Is(err, tt.want) {
				t.Fatalf("ParsePenaltyViolationJSON() error = %v, want %v", err, tt.want)
			}
		})
	}
}

package main

import (
	"fmt"

	wechatevent "github.com/jackman0925/wechat-ease/event"
)

func main() {
	// body 来自外部通知接收系统，进入此示例前已完成验签和解密。
	body := []byte(`{
"Event":"wxa_punish_event",
"punish_id":"649557",
"appid":"wx_example",
"illegal_content":["违规内容测试"],
"detail":"{\"warned_type\":1,\"warned_ban_days\":[3]}",
"event_type":1
}`)

	notification, err := wechatevent.ParsePlatformEventJSON(body)
	if err != nil {
		panic(err)
	}
	penalty, ok := notification.(*wechatevent.PenaltyViolationEvent)
	if !ok {
		return
	}

	fmt.Printf("处罚编号: %s，事件类型: %d\n", penalty.PunishID, penalty.EventType)
}

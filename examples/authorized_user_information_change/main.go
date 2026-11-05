package main

import (
	"fmt"

	wechatevent "github.com/jackman0925/wechat-ease/event"
)

func main() {
	// body 来自外部通知接收系统，进入此示例前已完成验签和解密。
	body := []byte(`{
"Event":"user_authorization_revoke",
"OpenID":"user-openid",
"AppID":"wx_example",
"RevokeInfo":"6"
}`)

	notification, err := wechatevent.ParsePlatformEventJSON(body)
	if err != nil {
		panic(err)
	}
	event, ok := notification.(*wechatevent.AuthorizedUserInformationChangeEvent)
	if !ok {
		return
	}

	fmt.Printf("用户 %s 撤回的授权信息类型: %s\n", event.OpenID, event.RevokeInfo)
}

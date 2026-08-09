package main

import (
	"context"
	"fmt"
	"time"

	wechatease "github.com/jackman0925/wechat-ease"
)

func main() {
	client := wechatease.NewClient(
		wechatease.WithTimeout(8 * time.Second),
	)

	// 页面必须存在于正式发布版本中，并允许被微信搜索收录。
	// Path 不以 / 开头，Query 只填写查询串本身，不以 ? 开头。
	err := client.SubmitPages(context.Background(), "your-access-token", wechatease.SubmitPagesRequest{
		Pages: []wechatease.SubmitPage{
			{
				Path:  "pages/content/detail/index",
				Query: "slug=birthday-wishes-for-friend",
			},
		},
	})
	if err != nil {
		fmt.Println("submit_pages failed:", err)
		return
	}

	fmt.Println("submit_pages accepted")
}

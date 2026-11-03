package main

import (
	"log"
	"net/http"

	wechatease "github.com/jackman0925/wechat-ease"
)

const wxToken = "your_custom_token_here"

func main() {
	http.HandleFunc("/wechat/callback", callbackHandler)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	signature := r.URL.Query().Get("signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echostr := r.URL.Query().Get("echostr")
	if signature == "" || timestamp == "" || nonce == "" || echostr == "" {
		http.Error(w, "参数缺失", http.StatusBadRequest)
		return
	}

	if !wechatease.VerifyCallbackSignature(wxToken, timestamp, nonce, signature) {
		http.Error(w, "signature error", http.StatusForbidden)
		return
	}

	// 微信要求验证成功后原样返回 echostr 纯文本。
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(echostr))
}

package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"os"

	wechatease "github.com/jackman0925/wechat-ease"
)

type encryptedRequest struct {
	Encrypt string `xml:"Encrypt"`
}

func main() {
	cryptor, err := wechatease.NewMessageCryptor(
		os.Getenv("WECHAT_CALLBACK_TOKEN"),
		os.Getenv("WECHAT_ENCODING_AES_KEY"),
		os.Getenv("WECHAT_APP_ID"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/wechat/encrypted-callback", encryptedCallbackHandler(cryptor))
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func encryptedCallbackHandler(cryptor *wechatease.MessageCryptor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request encryptedRequest
		if err := xml.NewDecoder(r.Body).Decode(&request); err != nil || request.Encrypt == "" {
			http.Error(w, "invalid encrypted message", http.StatusBadRequest)
			return
		}
		message, err := cryptor.DecryptMessage(
			r.URL.Query().Get("timestamp"),
			r.URL.Query().Get("nonce"),
			r.URL.Query().Get("msg_signature"),
			request.Encrypt,
		)
		if err != nil {
			http.Error(w, "signature or encrypted message error", http.StatusForbidden)
			return
		}

		log.Printf("decrypted message: %s", message)
		_, _ = w.Write([]byte("success"))
	}
}

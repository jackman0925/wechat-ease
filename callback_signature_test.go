package wechatease

import "testing"

func TestGenerateCallbackSignature(t *testing.T) {
	got := GenerateCallbackSignature("your_custom_token_here", "1710000000", "abc123")
	want := "5b30cfe5ed90e593daf770d82949fa111ac268c5"
	if got != want {
		t.Fatalf("GenerateCallbackSignature() = %q, want %q", got, want)
	}
}

func TestVerifyCallbackSignature(t *testing.T) {
	const (
		token     = "your_custom_token_here"
		timestamp = "1710000000"
		nonce     = "abc123"
		signature = "5b30cfe5ed90e593daf770d82949fa111ac268c5"
	)

	tests := []struct {
		name      string
		token     string
		timestamp string
		nonce     string
		signature string
		want      bool
	}{
		{name: "签名正确", token: token, timestamp: timestamp, nonce: nonce, signature: signature, want: true},
		{name: "签名错误", token: token, timestamp: timestamp, nonce: nonce, signature: "invalid-signature", want: false},
		{name: "Token为空", timestamp: timestamp, nonce: nonce, signature: signature, want: false},
		{name: "时间戳为空", token: token, nonce: nonce, signature: signature, want: false},
		{name: "随机数为空", token: token, timestamp: timestamp, signature: signature, want: false},
		{name: "签名为空", token: token, timestamp: timestamp, nonce: nonce, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyCallbackSignature(tt.token, tt.timestamp, tt.nonce, tt.signature); got != tt.want {
				t.Fatalf("VerifyCallbackSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

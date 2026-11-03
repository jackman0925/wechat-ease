package wechatease

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

const (
	testEncodingAESKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
	testEncryptedData  = "fb2IJy+yWTdpKjuBAEdBdaelmTk3p2XaE1PFaLFXBduyXAqcElJ93NDuFmzG7kkFPbcJT6DobytGj/U1kpi+QKOlJr+Ml0UaZXU6HhPEup/EwjnuLxGeApBJKwXUhacT"
	testMsgSignature   = "371cdc970ac88ac1ce29a9835381a82c534c8f01"
)

func TestNewMessageCryptor(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		encodingAESKey string
		appID          string
		wantErr        error
	}{
		{name: "创建成功", token: "test-token", encodingAESKey: testEncodingAESKey, appID: "wx-test-appid"},
		{name: "Token为空", encodingAESKey: testEncodingAESKey, appID: "wx-test-appid", wantErr: errors.New("token")},
		{name: "AppID为空", token: "test-token", encodingAESKey: testEncodingAESKey, wantErr: errors.New("appID")},
		{name: "密钥长度错误", token: "test-token", encodingAESKey: "short", appID: "wx-test-appid", wantErr: ErrInvalidEncodingAESKey},
		{name: "密钥编码错误", token: "test-token", encodingAESKey: "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", appID: "wx-test-appid", wantErr: ErrInvalidEncodingAESKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cryptor, err := NewMessageCryptor(tt.token, tt.encodingAESKey, tt.appID)
			if tt.wantErr == nil {
				if err != nil || cryptor == nil {
					t.Fatalf("NewMessageCryptor() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("NewMessageCryptor() expected error")
			}
			if errors.Is(tt.wantErr, ErrInvalidEncodingAESKey) && !errors.Is(err, ErrInvalidEncodingAESKey) {
				t.Fatalf("NewMessageCryptor() error = %v, want %v", err, ErrInvalidEncodingAESKey)
			}
		})
	}
}

func TestGenerateEncryptedCallbackSignature(t *testing.T) {
	got := GenerateEncryptedCallbackSignature("test-token", "1710000000", "abc123", "encrypted-data")
	want := "3786cd61ebf152e53cda7c427cc8f76988605438"
	if got != want {
		t.Fatalf("GenerateEncryptedCallbackSignature() = %q, want %q", got, want)
	}
}

func TestVerifyEncryptedCallbackSignature(t *testing.T) {
	if !VerifyEncryptedCallbackSignature("test-token", "1710000000", "abc123", testEncryptedData, testMsgSignature) {
		t.Fatal("VerifyEncryptedCallbackSignature() rejected valid signature")
	}
	if VerifyEncryptedCallbackSignature("test-token", "1710000000", "abc123", testEncryptedData, "invalid") {
		t.Fatal("VerifyEncryptedCallbackSignature() accepted invalid signature")
	}
	if VerifyEncryptedCallbackSignature("", "1710000000", "abc123", testEncryptedData, testMsgSignature) {
		t.Fatal("VerifyEncryptedCallbackSignature() accepted missing token")
	}
}

func TestMessageCryptorDecryptMessage(t *testing.T) {
	cryptor := newTestMessageCryptor(t, "wx-test-appid")
	message, err := cryptor.DecryptMessage("1710000000", "abc123", testMsgSignature, testEncryptedData)
	if err != nil {
		t.Fatalf("DecryptMessage() error = %v", err)
	}
	if want := "<xml><Content>hello</Content></xml>"; string(message) != want {
		t.Fatalf("DecryptMessage() = %q, want %q", message, want)
	}

	if _, err := cryptor.DecryptMessage("1710000000", "abc123", "invalid", testEncryptedData); !errors.Is(err, ErrInvalidMessageSignature) {
		t.Fatalf("DecryptMessage() signature error = %v", err)
	}
	invalidBase64 := "not-base64"
	invalidBase64Signature := GenerateEncryptedCallbackSignature("test-token", "1710000000", "abc123", invalidBase64)
	if _, err := cryptor.DecryptMessage("1710000000", "abc123", invalidBase64Signature, invalidBase64); !errors.Is(err, ErrInvalidEncryptedMessage) {
		t.Fatalf("DecryptMessage() ciphertext error = %v", err)
	}

	wrongAppIDCryptor := newTestMessageCryptor(t, "wx-wrong-appid")
	if _, err := wrongAppIDCryptor.DecryptMessage("1710000000", "abc123", testMsgSignature, testEncryptedData); !errors.Is(err, ErrAppIDMismatch) {
		t.Fatalf("DecryptMessage() AppID error = %v", err)
	}
}

func TestMessageCryptorEncryptMessage(t *testing.T) {
	cryptor := newTestMessageCryptor(t, "wx-test-appid")
	cryptor.random = bytes.NewReader([]byte("0123456789abcdef"))
	message := []byte("<xml><Content>hello</Content></xml>")

	result, err := cryptor.EncryptMessage(message, "1710000000", "abc123")
	if err != nil {
		t.Fatalf("EncryptMessage() error = %v", err)
	}
	if result.Encrypt != testEncryptedData {
		t.Fatalf("EncryptMessage().Encrypt = %q, want %q", result.Encrypt, testEncryptedData)
	}
	if result.MsgSignature != testMsgSignature || result.Timestamp != "1710000000" || result.Nonce != "abc123" {
		t.Fatalf("EncryptMessage() metadata = %+v", result)
	}

	if _, err := cryptor.EncryptMessage(message, "", "abc123"); err == nil {
		t.Fatal("EncryptMessage() accepted missing timestamp")
	}
	cryptor.random = failingReader{}
	if _, err := cryptor.EncryptMessage(message, "1710000000", "abc123"); err == nil {
		t.Fatal("EncryptMessage() expected random source error")
	}
}

func TestPKCS7Padding(t *testing.T) {
	for _, size := range []int{0, 1, 31, 32, 63, 64} {
		original := bytes.Repeat([]byte{'x'}, size)
		padded := pkcs7Pad(append([]byte(nil), original...))
		got, err := pkcs7Unpad(padded)
		if err != nil {
			t.Fatalf("size %d: pkcs7Unpad() error = %v", size, err)
		}
		if !bytes.Equal(got, original) {
			t.Fatalf("size %d: padding round trip mismatch", size)
		}
	}
	if _, err := pkcs7Unpad([]byte("invalid")); !errors.Is(err, ErrInvalidEncryptedMessage) {
		t.Fatalf("pkcs7Unpad() error = %v", err)
	}

	invalidPaddingLength := bytes.Repeat([]byte{'x'}, pkcs7BlockSize)
	invalidPaddingLength[len(invalidPaddingLength)-1] = 0
	if _, err := pkcs7Unpad(invalidPaddingLength); !errors.Is(err, ErrInvalidEncryptedMessage) {
		t.Fatalf("pkcs7Unpad() padding length error = %v", err)
	}

	invalidPadding := bytes.Repeat([]byte{'x'}, pkcs7BlockSize)
	invalidPadding[len(invalidPadding)-2] = 1
	invalidPadding[len(invalidPadding)-1] = 2
	if _, err := pkcs7Unpad(invalidPadding); !errors.Is(err, ErrInvalidEncryptedMessage) {
		t.Fatalf("pkcs7Unpad() padding content error = %v", err)
	}
}

func newTestMessageCryptor(t *testing.T, appID string) *MessageCryptor {
	t.Helper()
	cryptor, err := NewMessageCryptor("test-token", testEncodingAESKey, appID)
	if err != nil {
		t.Fatalf("NewMessageCryptor() error = %v", err)
	}
	return cryptor
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("random source failed")
}

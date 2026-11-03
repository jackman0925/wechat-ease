package wechatease

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	encodingAESKeyLength = 43
	randomPrefixLength   = 16
	messageLengthSize    = 4
	pkcs7BlockSize       = 32
)

var (
	// ErrInvalidEncodingAESKey 表示 EncodingAESKey 格式或长度不符合微信协议。
	ErrInvalidEncodingAESKey = errors.New("invalid EncodingAESKey")
	// ErrInvalidMessageSignature 表示加密消息签名不正确或签名参数不完整。
	ErrInvalidMessageSignature = errors.New("invalid message signature")
	// ErrInvalidEncryptedMessage 表示密文、填充或消息结构不符合微信协议。
	ErrInvalidEncryptedMessage = errors.New("invalid encrypted message")
	// ErrAppIDMismatch 表示解密消息中的 AppID 与本地配置不一致。
	ErrAppIDMismatch = errors.New("appID mismatch")
)

// EncryptedMessage 包含安全模式回复微信时需要返回的加密结果和签名参数。
type EncryptedMessage struct {
	// Encrypt 是经过 AES 加密并进行 Base64 编码后的消息正文。
	Encrypt string
	// MsgSignature 是由 Token、时间戳、随机数和 Encrypt 共同生成的签名。
	MsgSignature string
	// Timestamp 是参与消息签名的时间戳。
	Timestamp string
	// Nonce 是参与消息签名的随机字符串。
	Nonce string
}

// MessageCryptor 提供微信安全模式消息的签名、加密和解密能力。
// 实例创建后可安全地被多个 goroutine 并发用于正常请求处理。
type MessageCryptor struct {
	token  string
	appID  string
	aesKey []byte
	random io.Reader
}

// NewMessageCryptor 根据微信后台配置创建消息加解密工具。
func NewMessageCryptor(token, encodingAESKey, appID string) (*MessageCryptor, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	if appID == "" {
		return nil, fmt.Errorf("appID is required")
	}
	if len(encodingAESKey) != encodingAESKeyLength {
		return nil, fmt.Errorf("%w: expected %d characters", ErrInvalidEncodingAESKey, encodingAESKeyLength)
	}
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil || len(aesKey) != 32 {
		return nil, fmt.Errorf("%w: must decode to 32 bytes", ErrInvalidEncodingAESKey)
	}
	return &MessageCryptor{
		token:  token,
		appID:  appID,
		aesKey: append([]byte(nil), aesKey...),
		random: rand.Reader,
	}, nil
}

// GenerateEncryptedCallbackSignature 生成微信加密回调的消息签名。
func GenerateEncryptedCallbackSignature(token, timestamp, nonce, encrypted string) string {
	params := []string{token, timestamp, nonce, encrypted}
	sort.Strings(params)
	return sha1Hex(strings.Join(params, ""))
}

// VerifyEncryptedCallbackSignature 校验微信加密回调的消息签名。
func VerifyEncryptedCallbackSignature(token, timestamp, nonce, encrypted, signature string) bool {
	if token == "" || timestamp == "" || nonce == "" || encrypted == "" || signature == "" {
		return false
	}
	calculated := GenerateEncryptedCallbackSignature(token, timestamp, nonce, encrypted)
	return subtle.ConstantTimeCompare([]byte(calculated), []byte(signature)) == 1
}

// DecryptMessage 验证 msg_signature 后解密微信回调消息，并校验消息中的 AppID。
func (c *MessageCryptor) DecryptMessage(timestamp, nonce, signature, encrypted string) ([]byte, error) {
	if !VerifyEncryptedCallbackSignature(c.token, timestamp, nonce, encrypted, signature) {
		return nil, ErrInvalidMessageSignature
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%pkcs7BlockSize != 0 {
		return nil, fmt.Errorf("%w: invalid base64 ciphertext", ErrInvalidEncryptedMessage)
	}
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, c.aesKey[:aes.BlockSize]).CryptBlocks(plaintext, ciphertext)
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return nil, err
	}
	if len(plaintext) < randomPrefixLength+messageLengthSize {
		return nil, fmt.Errorf("%w: plaintext is too short", ErrInvalidEncryptedMessage)
	}

	messageLength := binary.BigEndian.Uint32(plaintext[randomPrefixLength : randomPrefixLength+messageLengthSize])
	remainingLength := len(plaintext) - randomPrefixLength - messageLengthSize
	if uint64(messageLength) > uint64(remainingLength) {
		return nil, fmt.Errorf("%w: message length exceeds plaintext", ErrInvalidEncryptedMessage)
	}
	messageStart := randomPrefixLength + messageLengthSize
	messageEnd := messageStart + int(messageLength)
	messageAppID := plaintext[messageEnd:]
	if subtle.ConstantTimeCompare(messageAppID, []byte(c.appID)) != 1 {
		return nil, ErrAppIDMismatch
	}
	return append([]byte(nil), plaintext[messageStart:messageEnd]...), nil
}

// EncryptMessage 加密回复消息，并生成微信安全模式所需的 msg_signature。
func (c *MessageCryptor) EncryptMessage(message []byte, timestamp, nonce string) (*EncryptedMessage, error) {
	if timestamp == "" || nonce == "" {
		return nil, fmt.Errorf("timestamp and nonce are required")
	}
	if uint64(len(message)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("message is too large")
	}

	randomPrefix := make([]byte, randomPrefixLength)
	if _, err := io.ReadFull(c.random, randomPrefix); err != nil {
		return nil, fmt.Errorf("generate random prefix: %w", err)
	}
	plaintext := make([]byte, randomPrefixLength+messageLengthSize+len(message)+len(c.appID))
	copy(plaintext, randomPrefix)
	binary.BigEndian.PutUint32(plaintext[randomPrefixLength:], uint32(len(message)))
	messageStart := randomPrefixLength + messageLengthSize
	copy(plaintext[messageStart:], message)
	copy(plaintext[messageStart+len(message):], c.appID)
	plaintext = pkcs7Pad(plaintext)

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, c.aesKey[:aes.BlockSize]).CryptBlocks(ciphertext, plaintext)
	encrypted := base64.StdEncoding.EncodeToString(ciphertext)
	return &EncryptedMessage{
		Encrypt:      encrypted,
		MsgSignature: GenerateEncryptedCallbackSignature(c.token, timestamp, nonce, encrypted),
		Timestamp:    timestamp,
		Nonce:        nonce,
	}, nil
}

func pkcs7Pad(data []byte) []byte {
	paddingLength := pkcs7BlockSize - len(data)%pkcs7BlockSize
	return append(data, bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%pkcs7BlockSize != 0 {
		return nil, fmt.Errorf("%w: invalid PKCS#7 block length", ErrInvalidEncryptedMessage)
	}
	paddingLength := int(data[len(data)-1])
	if paddingLength < 1 || paddingLength > pkcs7BlockSize || paddingLength > len(data) {
		return nil, fmt.Errorf("%w: invalid PKCS#7 padding length", ErrInvalidEncryptedMessage)
	}
	for _, value := range data[len(data)-paddingLength:] {
		if int(value) != paddingLength {
			return nil, fmt.Errorf("%w: invalid PKCS#7 padding", ErrInvalidEncryptedMessage)
		}
	}
	return data[:len(data)-paddingLength], nil
}

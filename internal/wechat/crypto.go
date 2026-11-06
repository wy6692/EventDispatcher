package wechat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCipher = errors.New("invalid ciphertext")
	ErrAppIDMismatch = errors.New("appid mismatch")
)

// DecodeAESKey 把 43 位 EncodingAESKey 解码成 32 字节 AES-256 密钥。
func DecodeAESKey(encodingAESKey string) ([]byte, error) {
	if len(encodingAESKey) != 43 {
		return nil, fmt.Errorf("EncodingAESKey 必须是 43 位")
	}
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("EncodingAESKey 无法解码")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("EncodingAESKey 解码后必须是 32 字节")
	}
	return key, nil
}

// Encrypt 按微信规则加密：16 字节随机数 + 4 字节消息长度 + 消息 + AppID，PKCS#7 块长 32。
func Encrypt(encodingAESKey, appID string, msg []byte) (string, error) {
	key, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		return "", err
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	buf := make([]byte, 0, 20+len(msg)+len(appID))
	buf = append(buf, random...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(msg)))
	buf = append(buf, msg...)
	buf = append(buf, appID...)
	padded := pkcs7Pad(buf, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	dst := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(dst, padded)
	return base64.StdEncoding.EncodeToString(dst), nil
}

// Decrypt 解密并核对明文尾部的 AppID。
func Decrypt(encodingAESKey, appID, encrypted string) ([]byte, error) {
	key, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(CleanBase64(encrypted))
	if err != nil || len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return nil, ErrInvalidCipher
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	dst := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(dst, raw)
	dst, err = pkcs7Unpad(dst, 32)
	if err != nil || len(dst) < 20 {
		return nil, ErrInvalidCipher
	}
	msgLen := int(binary.BigEndian.Uint32(dst[16:20]))
	if msgLen < 0 || 20+msgLen > len(dst) {
		return nil, ErrInvalidCipher
	}
	msg := dst[20 : 20+msgLen]
	gotAppID := string(dst[20+msgLen:])
	if gotAppID != appID {
		return nil, ErrAppIDMismatch
	}
	out := make([]byte, len(msg))
	copy(out, msg)
	return out, nil
}

// CleanBase64 修正 query string 把 '+' 解析成空格的情况。
func CleanBase64(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "+")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// 微信的 PKCS#7 块长是 32，不是 AES 的 16。改成 16 会解不开官方密文。
func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	if pad == 0 {
		pad = blockSize
	}
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, ErrInvalidCipher
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > blockSize || pad > len(data) {
		return nil, ErrInvalidCipher
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, ErrInvalidCipher
		}
	}
	return data[:len(data)-pad], nil
}

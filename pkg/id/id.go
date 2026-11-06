package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// New 生成具有业务语义前缀的防碰撞随机主键 ID。
// 示例：New("acc_") -> "acc_a8b9c0d1e2f3..."
func New(prefix string) (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("生成唯一 ID 失败: %w", err)
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}

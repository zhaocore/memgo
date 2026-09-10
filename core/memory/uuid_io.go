package memory

import "crypto/rand"

// randRead 转发 crypto/rand.Read (保持 memory.go 简洁)。
func randRead(b []byte) (int, error) {
	return rand.Read(b)
}

// Package auth: 密码/API Key 哈希、JWT 签发校验、三层鉴权依赖 (对齐 server/auth.py)。
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword bcrypt (passlib 默认 rounds=12, Go 默认 DefaultCost=12 一致)。
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// VerifyPassword 恒时比对。
func VerifyPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// DummyVerify 对齐 dummy_verify_password: 空烧同量 bcrypt (时序防探测)。
func DummyVerify() {
	_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$K0D.HfgkhgZ0ZXaA5k8ZWu8h6xS0PQyOcrRCF3C0hCPp7cIhU0mMi"), []byte("dummy"))
}

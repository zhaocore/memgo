package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token 时效对齐 server/auth.py (access 30min / refresh 30d)。
const (
	AccessTokenTTL  = 30 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

// Claims 自定义声明 (sub/role/exp/jti/type)。
type Claims struct {
	Role string `json:"role,omitempty"`
	Type string `json:"type"`
	JTI  string `json:"jti,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager 签发/校验 (HS256 + JWT_SECRET)。
type TokenManager struct {
	secret []byte
}

// NewTokenManager 构造。
func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{secret: []byte(secret)}
}

// CreateAccessToken 对齐 create_access_token。
func (tm *TokenManager) CreateAccessToken(userID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tm.secret)
}

// CreateRefreshToken 对齐 create_refresh_token (jti 由调用方落库)。
// 返回 (token, jti, expiresAt)。
func (tm *TokenManager) CreateRefreshToken(userID string) (string, string, time.Time, error) {
	now := time.Now()
	expires := now.Add(RefreshTokenTTL)
	jti := newUUID()
	claims := Claims{
		Type: "refresh",
		JTI:  jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tm.secret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("auth: 签发 refresh 失败: %w", err)
	}
	return token, jti, expires, nil
}

// Decode 校验签名并返回 claims (过期自动拒)。
func (tm *TokenManager) Decode(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return tm.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

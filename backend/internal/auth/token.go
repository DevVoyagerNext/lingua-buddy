package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken 令牌无效或过期。
var ErrInvalidToken = errors.New("invalid token")

// JWTManager 负责签发与解析登录 JWT。
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager 构造 JWTManager。ttl<=0 时默认 7 天。
func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Issue 为指定用户签发访问令牌。
func (m *JWTManager) Issue(userID uint64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 校验令牌并返回用户 ID。
func (m *JWTManager) Parse(tokenString string) (uint64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}
	return id, nil
}

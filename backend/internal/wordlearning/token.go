package wordlearning

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// 令牌相关错误。
var (
	ErrTokenInvalid = errors.New("question token invalid")
	ErrTokenExpired = errors.New("question token expired")
)

// tokenTTL 题目令牌有效期。
const tokenTTL = 24 * time.Hour

// QuestionToken 是签发给前端的题目凭据，提交时回传校验。不含正确答案。
type QuestionToken struct {
	UserID       uint64   `json:"uid"`
	PlanID       *uint64  `json:"pid,omitempty"`
	PlanItemID   *uint64  `json:"pii,omitempty"`
	UserWordID   uint64   `json:"uwid"`
	Stage        string   `json:"stage"`
	QuestionType string   `json:"qt"`
	QuestionKey  string   `json:"qk"`
	Options      []string `json:"opts,omitempty"`
	GenVersion   string   `json:"gv"`
	IssuedAt     int64    `json:"iat"`
	Nonce        string   `json:"nonce"`
}

// TokenManager 负责题目令牌的签名与校验。
type TokenManager struct {
	secret []byte
}

// NewTokenManager 构造令牌管理器。
func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{secret: []byte(secret)}
}

// Sign 生成 payload.signature 形式的令牌。
func (m *TokenManager) Sign(tok *QuestionToken) (string, error) {
	if tok.Nonce == "" {
		tok.Nonce = randomNonce()
	}
	if tok.IssuedAt == 0 {
		tok.IssuedAt = time.Now().Unix()
	}
	payload, err := json.Marshal(tok)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + m.sign(encoded), nil
}

// Parse 校验签名与过期，返回令牌内容。
func (m *TokenManager) Parse(s string) (*QuestionToken, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return nil, ErrTokenInvalid
	}
	if !hmac.Equal([]byte(m.sign(parts[0])), []byte(parts[1])) {
		return nil, ErrTokenInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var tok QuestionToken
	if err := json.Unmarshal(payload, &tok); err != nil {
		return nil, ErrTokenInvalid
	}
	if time.Since(time.Unix(tok.IssuedAt, 0)) > tokenTTL {
		return nil, ErrTokenExpired
	}
	return &tok, nil
}

func (m *TokenManager) sign(encoded string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(encoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomNonce() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

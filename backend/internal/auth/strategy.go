package auth

import (
	"context"
	"errors"
	"strings"

	"lingua-buddy/internal/models"
	"lingua-buddy/internal/user"
)

// RegistrationMethod 注册方式。
type RegistrationMethod string

const (
	// MethodUsernamePassword 普通账号密码注册（首版唯一实现）。
	MethodUsernamePassword RegistrationMethod = "username_password"
)

// ErrMethodUnsupported 不支持的注册方式。
var ErrMethodUnsupported = errors.New("registration method unsupported")

// RegistrationRequest 注册输入。
type RegistrationRequest struct {
	Method   RegistrationMethod
	Username string
	Password string
	Email    *string
}

// RegistrationStrategy 单一注册方式的策略接口。
type RegistrationStrategy interface {
	Method() RegistrationMethod
	Register(ctx context.Context, req RegistrationRequest) (*models.User, error)
}

// Registry 按 method 分发注册策略。
type Registry struct {
	strategies map[RegistrationMethod]RegistrationStrategy
}

// NewRegistry 构造注册表并注册给定策略。
func NewRegistry(strategies ...RegistrationStrategy) *Registry {
	m := make(map[RegistrationMethod]RegistrationStrategy, len(strategies))
	for _, s := range strategies {
		m[s.Method()] = s
	}
	return &Registry{strategies: m}
}

// Get 返回对应方式的策略，未知方式返回 ErrMethodUnsupported。
func (r *Registry) Get(method RegistrationMethod) (RegistrationStrategy, error) {
	s, ok := r.strategies[method]
	if !ok {
		return nil, ErrMethodUnsupported
	}
	return s, nil
}

// UsernamePasswordStrategy 普通账号密码注册。
type UsernamePasswordStrategy struct {
	users *user.Repository
}

// NewUsernamePasswordStrategy 构造策略。
func NewUsernamePasswordStrategy(users *user.Repository) *UsernamePasswordStrategy {
	return &UsernamePasswordStrategy{users: users}
}

// Method 返回 username_password。
func (s *UsernamePasswordStrategy) Method() RegistrationMethod { return MethodUsernamePassword }

// Register 直接创建账号：不验证邮箱、不限制注册。
func (s *UsernamePasswordStrategy) Register(ctx context.Context, req RegistrationRequest) (*models.User, error) {
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	var email *string
	if req.Email != nil {
		e := strings.TrimSpace(*req.Email)
		if e != "" {
			email = &e
		}
	}
	u := &models.User{
		Username:           req.Username,
		Email:              email,
		PasswordHash:       &hash,
		RegistrationMethod: string(MethodUsernamePassword),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

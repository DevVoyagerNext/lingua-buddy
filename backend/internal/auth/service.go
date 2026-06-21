package auth

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/user"
)

var (
	usernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,30}$`)
	emailRe    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// Service 处理注册与登录。
type Service struct {
	users    *user.Repository
	registry *Registry
	jwt      *JWTManager
}

// NewService 构造认证服务。
func NewService(users *user.Repository, registry *Registry, jwt *JWTManager) *Service {
	return &Service{users: users, registry: registry, jwt: jwt}
}

// Result 注册/登录结果。
type Result struct {
	User  *models.User
	Token string
}

// Register 校验输入、分发策略、签发 JWT。
func (s *Service) Register(ctx context.Context, req RegistrationRequest) (*Result, error) {
	if req.Method == "" {
		req.Method = MethodUsernamePassword
	}
	strategy, err := s.registry.Get(req.Method)
	if err != nil {
		return nil, httpx.NewError(http.StatusBadRequest, "REGISTRATION_METHOD_UNSUPPORTED", "不支持的注册方式")
	}

	req.Username = strings.TrimSpace(req.Username)
	if !usernameRe.MatchString(req.Username) {
		return nil, httpx.ErrValidation("用户名需为 3-30 位字母、数字或下划线")
	}
	if len(req.Password) < 8 {
		return nil, httpx.ErrValidation("密码至少 8 位")
	}
	if req.Email != nil {
		e := strings.TrimSpace(*req.Email)
		if e != "" && !emailRe.MatchString(e) {
			return nil, httpx.ErrValidation("邮箱格式不正确")
		}
	}

	// 预检查以返回清晰冲突信息；数据库唯一索引为最终保证。
	if _, err := s.users.FindByUsername(ctx, req.Username); err == nil {
		return nil, httpx.ErrConflict("用户名已被占用")
	} else if !errors.Is(err, user.ErrNotFound) {
		return nil, err
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		if _, err := s.users.FindByEmail(ctx, strings.TrimSpace(*req.Email)); err == nil {
			return nil, httpx.ErrConflict("邮箱已被占用")
		} else if !errors.Is(err, user.ErrNotFound) {
			return nil, err
		}
	}

	u, err := strategy.Register(ctx, req)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.ErrConflict("用户名或邮箱已被占用")
		}
		return nil, err
	}

	token, err := s.jwt.Issue(u.ID)
	if err != nil {
		return nil, err
	}
	return &Result{User: u, Token: token}, nil
}

// Login 用户名或邮箱 + 密码登录。
func (s *Service) Login(ctx context.Context, login, password string) (*Result, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return nil, httpx.ErrValidation("账号和密码不能为空")
	}
	u, err := s.users.FindByLogin(ctx, login)
	if errors.Is(err, user.ErrNotFound) {
		return nil, httpx.ErrUnauthorized("账号或密码错误")
	}
	if err != nil {
		return nil, err
	}
	if u.PasswordHash == nil || !CheckPassword(*u.PasswordHash, password) {
		return nil, httpx.ErrUnauthorized("账号或密码错误")
	}
	token, err := s.jwt.Issue(u.ID)
	if err != nil {
		return nil, err
	}
	return &Result{User: u, Token: token}, nil
}

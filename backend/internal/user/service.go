package user

import (
	"context"
	"net/http"
	"strings"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
)

// ValidEnglishLevels 允许的英语水平取值。
var ValidEnglishLevels = map[string]bool{
	"beginner":     true, // 初级
	"intermediate": true, // 中级
	"advanced":     true, // 高级
	"cet4":         true,
	"cet6":         true,
}

// DefaultEnglishLevel 用户未设置时 AI 使用的默认档。
const DefaultEnglishLevel = "intermediate"

// Service 个人资料服务。
type Service struct {
	repo *Repository
}

// NewService 构造用户服务。
func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// Me 返回当前用户。
func (s *Service) Me(ctx context.Context, id uint64) (*models.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err == ErrNotFound {
		return nil, httpx.ErrNotFound("用户不存在")
	}
	return u, err
}

// UpdateInput 个人资料更新输入（字段为 nil 表示不修改）。
type UpdateInput struct {
	Email        *string
	EnglishLevel *string
}

// Update 更新邮箱或英语水平。
func (s *Service) Update(ctx context.Context, id uint64, in UpdateInput) (*models.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err == ErrNotFound {
		return nil, httpx.ErrNotFound("用户不存在")
	}
	if err != nil {
		return nil, err
	}

	if in.EnglishLevel != nil {
		lv := strings.ToLower(strings.TrimSpace(*in.EnglishLevel))
		if !ValidEnglishLevels[lv] {
			return nil, httpx.ErrValidation("英语水平取值不合法")
		}
		u.EnglishLevel = lv
	}
	if in.Email != nil {
		e := strings.TrimSpace(*in.Email)
		if e == "" {
			u.Email = nil
		} else {
			// 邮箱唯一性：若被他人占用则冲突。
			if existing, err := s.repo.FindByEmail(ctx, e); err == nil && existing.ID != id {
				return nil, httpx.NewError(http.StatusConflict, "CONFLICT", "邮箱已被占用")
			}
			u.Email = &e
		}
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// EffectiveLevel 返回用于 AI 的有效英语水平（空值兜底为默认档）。
func EffectiveLevel(level string) string {
	if level == "" {
		return DefaultEnglishLevel
	}
	return level
}

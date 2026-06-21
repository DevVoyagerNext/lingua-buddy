// Package user 负责用户账号的持久化与个人资料维护。
package user

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"lingua-buddy/internal/models"
)

// ErrNotFound 用户不存在。
var ErrNotFound = errors.New("user not found")

// Repository 用户持久化接口。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造用户仓库。
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create 新增用户。用户名或邮箱重复时返回 gorm.ErrDuplicatedKey。
func (r *Repository) Create(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// FindByID 按 ID 查询。
func (r *Repository) FindByID(ctx context.Context, id uint64) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// FindByUsername 按用户名查询。
func (r *Repository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// FindByEmail 按邮箱查询。
func (r *Repository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// FindByLogin 用户名或邮箱登录查询：先按用户名，再按邮箱。
func (r *Repository) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	u, err := r.FindByUsername(ctx, login)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return r.FindByEmail(ctx, login)
}

// Update 保存用户字段（邮箱、英语水平等）。
func (r *Repository) Update(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

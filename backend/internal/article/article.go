// Package article 提供外刊文章列表、阅读与阅读记录。
package article

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
)

// Repository 文章持久化。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造。
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Upsert 按 source_url 幂等导入/更新文章（供同步命令使用）。
func (r *Repository) Upsert(ctx context.Context, a *models.Article) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source_url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "summary", "difficulty", "source_name", "attribution", "published_at", "updated_at",
		}),
	}).Create(a).Error
}

// List 分页查询文章，可按难度过滤、标题搜索。
func (r *Repository) List(ctx context.Context, difficulty, keyword string, page, size int) ([]models.Article, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Article{})
	if difficulty != "" {
		tx = tx.Where("difficulty = ?", difficulty)
	}
	if keyword != "" {
		tx = tx.Where("title LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Article
	err := tx.Order("published_at DESC, created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// Get 查询文章详情。
func (r *Repository) Get(ctx context.Context, id uint64) (*models.Article, error) {
	var a models.Article
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// MarkRead 新增或更新阅读记录。
func (r *Repository) MarkRead(ctx context.Context, userID, articleID uint64, finished bool) error {
	rec := models.UserArticleRead{UserID: userID, ArticleID: articleID, IsFinished: finished, LastReadAt: time.Now()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "article_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_finished", "last_read_at"}),
	}).Create(&rec).Error
}

// History 阅读历史（连带文章标题）。
type HistoryRow struct {
	ID         uint64    `json:"id"`
	ArticleID  uint64    `json:"article_id"`
	Title      string    `json:"title"`
	IsFinished bool      `json:"is_finished"`
	LastReadAt time.Time `json:"last_read_at"`
}

// ListHistory 分页查询阅读历史。
func (r *Repository) ListHistory(ctx context.Context, userID uint64, page, size int) ([]HistoryRow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.UserArticleRead{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []HistoryRow
	err := r.db.WithContext(ctx).
		Table("user_article_reads AS ar").
		Select("ar.id, ar.article_id, a.title, ar.is_finished, ar.last_read_at").
		Joins("JOIN articles a ON a.id = ar.article_id").
		Where("ar.user_id = ?", userID).
		Order("ar.last_read_at DESC").
		Offset((page - 1) * size).Limit(size).
		Scan(&rows).Error
	return rows, total, err
}

// DeleteHistory 删除阅读记录。
func (r *Repository) DeleteHistory(ctx context.Context, userID, id uint64) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.UserArticleRead{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Handler 暴露外刊接口。
type Handler struct{ repo *Repository }

// NewHandler 构造。
func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/articles", h.list)
	rg.GET("/articles/history", h.history)
	rg.DELETE("/articles/history/:id", h.deleteHistory)
	rg.GET("/articles/:id", h.get)
	rg.POST("/articles/:id/read", h.read)
}

func (h *Handler) list(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.repo.List(c.Request.Context(), c.Query("difficulty"), c.Query("keyword"), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	a, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, httpx.ErrNotFound("文章不存在"))
		return
	}
	httpx.OK(c, a)
}

type readReq struct {
	Finished bool `json:"finished"`
}

func (h *Handler) read(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req readReq
	_ = c.ShouldBindJSON(&req)
	if err := h.repo.MarkRead(c.Request.Context(), httpx.MustUserID(c), id, req.Finished); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) history(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	rows, total, err := h.repo.ListHistory(c.Request.Context(), httpx.MustUserID(c), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: rows, Page: page, PageSize: size, Total: total})
}

func (h *Handler) deleteHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.repo.DeleteHistory(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, httpx.ErrNotFound("阅读记录不存在"))
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

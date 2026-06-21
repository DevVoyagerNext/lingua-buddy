// Package sentence 提供收藏句子的增删改查。
package sentence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
)

// Service 句子收藏服务。
type Service struct {
	db *gorm.DB
}

// NewService 构造。
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func hashSentence(s string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(s))))
	return hex.EncodeToString(sum[:])
}

// Create 收藏句子，按 (user_id, sentence_hash) 去重。
func (s *Service) Create(ctx context.Context, userID uint64, sentence string, translation, analysis, note *string) (*models.UserSentence, error) {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return nil, httpx.ErrValidation("句子不能为空")
	}
	rec := &models.UserSentence{
		UserID: userID, Sentence: sentence, SentenceHash: hashSentence(sentence),
		Translation: translation, Analysis: analysis, Note: note,
	}
	if err := s.db.WithContext(ctx).Create(rec).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.ErrConflict("该句子已收藏")
		}
		return nil, err
	}
	return rec, nil
}

// List 分页查询收藏句子，可按内容搜索。
func (s *Service) List(ctx context.Context, userID uint64, keyword string, page, size int) ([]models.UserSentence, int64, error) {
	tx := s.db.WithContext(ctx).Model(&models.UserSentence{}).Where("user_id = ?", userID)
	if keyword != "" {
		tx = tx.Where("sentence LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.UserSentence
	err := tx.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// Update 修改翻译、句子分析或备注。
func (s *Service) Update(ctx context.Context, userID, id uint64, translation, analysis, note *string) error {
	fields := map[string]any{}
	if translation != nil {
		fields["translation"] = *translation
	}
	if analysis != nil {
		fields["analysis"] = *analysis
	}
	if note != nil {
		fields["note"] = *note
	}
	if len(fields) == 0 {
		return nil
	}
	res := s.db.WithContext(ctx).Model(&models.UserSentence{}).
		Where("id = ? AND user_id = ?", id, userID).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.ErrNotFound("句子不存在")
	}
	return nil
}

// Delete 取消收藏。
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.UserSentence{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.ErrNotFound("句子不存在")
	}
	return nil
}

// Handler 暴露句子接口。
type Handler struct{ svc *Service }

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/sentences", h.create)
	rg.GET("/sentences", h.list)
	rg.PATCH("/sentences/:id", h.update)
	rg.DELETE("/sentences/:id", h.delete)
}

type createReq struct {
	Sentence    string  `json:"sentence"`
	Translation *string `json:"translation"`
	Analysis    *string `json:"analysis"`
	Note        *string `json:"note"`
}

func (h *Handler) create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	rec, err := h.svc.Create(c.Request.Context(), httpx.MustUserID(c), req.Sentence, req.Translation, req.Analysis, req.Note)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rec)
}

func (h *Handler) list(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.svc.List(c.Request.Context(), httpx.MustUserID(c), c.Query("keyword"), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: items, Page: page, PageSize: size, Total: total})
}

type updateReq struct {
	Translation *string `json:"translation"`
	Analysis    *string `json:"analysis"`
	Note        *string `json:"note"`
}

func (h *Handler) update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	if err := h.svc.Update(c.Request.Context(), httpx.MustUserID(c), id, req.Translation, req.Analysis, req.Note); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"updated": true})
}

func (h *Handler) delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

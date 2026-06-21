// Package wordnote 提供单词笔记的增删改查（与生词收藏相互独立）。
package wordnote

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
)

// Service 单词笔记服务。
type Service struct {
	db *gorm.DB
}

// NewService 构造。
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// Create 为单词新增一条笔记。
func (s *Service) Create(ctx context.Context, userID uint64, word, content string) (*models.UserWordNote, error) {
	word = strings.TrimSpace(word)
	content = strings.TrimSpace(content)
	if word == "" || content == "" {
		return nil, httpx.ErrValidation("单词和笔记内容不能为空")
	}
	note := &models.UserWordNote{UserID: userID, Word: word, Content: content}
	if err := s.db.WithContext(ctx).Create(note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

// ListByWord 查询某个单词的全部笔记。
func (s *Service) ListByWord(ctx context.Context, userID uint64, word string) ([]models.UserWordNote, error) {
	var notes []models.UserWordNote
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND word = ?", userID, strings.TrimSpace(word)).
		Order("created_at DESC").Find(&notes).Error
	return notes, err
}

// Update 修改笔记内容。
func (s *Service) Update(ctx context.Context, userID, id uint64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return httpx.ErrValidation("笔记内容不能为空")
	}
	res := s.db.WithContext(ctx).Model(&models.UserWordNote{}).
		Where("id = ? AND user_id = ?", id, userID).Update("content", content)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.ErrNotFound("笔记不存在")
	}
	return nil
}

// Delete 删除笔记。
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.UserWordNote{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.ErrNotFound("笔记不存在")
	}
	return nil
}

// Handler 暴露单词笔记接口。
type Handler struct{ svc *Service }

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/word-notes", h.create)
	rg.GET("/word-notes", h.listByWord)
	rg.PATCH("/word-notes/:id", h.update)
	rg.DELETE("/word-notes/:id", h.delete)
}

type createReq struct {
	Word    string `json:"word"`
	Content string `json:"content"`
}

func (h *Handler) create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	note, err := h.svc.Create(c.Request.Context(), httpx.MustUserID(c), req.Word, req.Content)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, note)
}

func (h *Handler) listByWord(c *gin.Context) {
	word := c.Query("word")
	if strings.TrimSpace(word) == "" {
		httpx.Fail(c, httpx.ErrValidation("缺少 word 参数"))
		return
	}
	notes, err := h.svc.ListByWord(c.Request.Context(), httpx.MustUserID(c), word)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, notes)
}

type updateReq struct {
	Content string `json:"content"`
}

func (h *Handler) update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	if err := h.svc.Update(c.Request.Context(), httpx.MustUserID(c), id, req.Content); err != nil {
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

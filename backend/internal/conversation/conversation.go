// Package conversation 提供 AI 情景对话会话与多轮消息。
package conversation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/user"
)

// Service 对话服务。
type Service struct {
	db    *gorm.DB
	ai    ai.Provider
	level user.LevelLookup
}

// NewService 构造。
func NewService(db *gorm.DB, provider ai.Provider, level user.LevelLookup) *Service {
	return &Service{db: db, ai: provider, level: level}
}

// sceneLabels 场景中文名。
var sceneLabels = map[string]string{
	"travel":     "旅行对话",
	"restaurant": "餐厅点餐",
	"campus":     "校园交流",
	"interview":  "求职面试",
	"cet":        "CET口语",
	"free":       "自由对话",
}

func sceneLabel(scene string) string {
	if l, ok := sceneLabels[scene]; ok {
		return l
	}
	return scene
}

// uniqueTitle 为会话生成同一用户内不重复的名称，如「餐厅点餐 1」。
func (s *Service) uniqueTitle(ctx context.Context, userID uint64, scene string) string {
	base := sceneLabel(scene)
	var count int64
	s.db.WithContext(ctx).Model(&models.Conversation{}).
		Where("user_id = ? AND scene = ?", userID, scene).Count(&count)
	n := int(count) + 1
	for {
		candidate := fmt.Sprintf("%s %d", base, n)
		var exists int64
		s.db.WithContext(ctx).Model(&models.Conversation{}).
			Where("user_id = ? AND title = ?", userID, candidate).Count(&exists)
		if exists == 0 {
			return candidate
		}
		n++
	}
}

// Create 新建会话。
func (s *Service) Create(ctx context.Context, userID uint64, scene, difficulty, role, title string) (*models.Conversation, error) {
	if scene == "" {
		scene = "free"
	}
	if title == "" {
		title = s.uniqueTitle(ctx, userID, scene)
	}
	conv := &models.Conversation{
		UserID: userID, Title: title, Scene: scene, Difficulty: difficulty, Status: "active",
	}
	if err := s.db.WithContext(ctx).Create(conv).Error; err != nil {
		return nil, err
	}
	return conv, nil
}

// ConversationView 会话列表项，附带最近一条用户消息预览。
type ConversationView struct {
	models.Conversation
	LastMessage string `json:"last_message"`
}

// List 会话列表（按创建顺序正序，新会话排在最后）。
func (s *Service) List(ctx context.Context, userID uint64) ([]ConversationView, error) {
	var items []models.Conversation
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	views := make([]ConversationView, 0, len(items))
	for _, c := range items {
		var msg models.ConversationMessage
		s.db.WithContext(ctx).
			Where("conversation_id = ? AND role = ?", c.ID, "user").
			Order("created_at DESC").Limit(1).Take(&msg)
		views = append(views, ConversationView{Conversation: c, LastMessage: msg.Content})
	}
	return views, nil
}

// Messages 会话消息（按时间正序）。
func (s *Service) Messages(ctx context.Context, userID, convID uint64) ([]models.ConversationMessage, error) {
	if _, err := s.getConv(ctx, userID, convID); err != nil {
		return nil, err
	}
	var msgs []models.ConversationMessage
	err := s.db.WithContext(ctx).Where("conversation_id = ?", convID).Order("created_at ASC").Find(&msgs).Error
	return msgs, err
}

// SendResult 发送消息结果。
type SendResult struct {
	UserMessage models.ConversationMessage `json:"user_message"`
	AIMessage   models.ConversationMessage `json:"ai_message"`
}

// Send 发送用户消息，调用 AI，保存两条消息。
func (s *Service) Send(ctx context.Context, userID, convID uint64, content string) (*SendResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, httpx.ErrValidation("消息不能为空")
	}
	conv, err := s.getConv(ctx, userID, convID)
	if err != nil {
		return nil, err
	}
	if conv.Status == "finished" {
		return nil, httpx.ErrValidation("会话已结束")
	}

	// 取历史消息作为上下文。
	var history []models.ConversationMessage
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", convID).Order("created_at ASC").Find(&history).Error; err != nil {
		return nil, err
	}
	turns := make([]ai.ChatTurn, 0, len(history))
	for _, m := range history {
		turns = append(turns, ai.ChatTurn{Role: m.Role, Content: m.Content})
	}

	out, err := s.ai.Chat(ctx, ai.ChatInput{
		Scene: conv.Scene, Difficulty: conv.Difficulty, Role: conv.Scene,
		Level: s.level.Level(ctx, userID), History: turns, UserMessage: content,
	})
	if err != nil {
		status, code, msg := ai.ErrorCode(err)
		return nil, httpx.NewError(status, code, msg)
	}

	userMsg := models.ConversationMessage{ConversationID: convID, Role: "user", Content: content}
	feedback := out.Feedback
	aiMsg := models.ConversationMessage{ConversationID: convID, Role: "assistant", Content: out.Reply}
	if feedback != "" {
		aiMsg.Feedback = &feedback
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&userMsg).Error; err != nil {
			return err
		}
		if err := tx.Create(&aiMsg).Error; err != nil {
			return err
		}
		return tx.Model(&models.Conversation{}).Where("id = ?", convID).Update("updated_at", aiMsg.CreatedAt).Error
	})
	if err != nil {
		return nil, err
	}
	return &SendResult{UserMessage: userMsg, AIMessage: aiMsg}, nil
}

// Finish 结束会话。
func (s *Service) Finish(ctx context.Context, userID, convID uint64) error {
	if _, err := s.getConv(ctx, userID, convID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&models.Conversation{}).Where("id = ?", convID).Update("status", "finished").Error
}

// Delete 删除会话及其消息。
func (s *Service) Delete(ctx context.Context, userID, convID uint64) error {
	if _, err := s.getConv(ctx, userID, convID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", convID).Delete(&models.ConversationMessage{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", convID, userID).Delete(&models.Conversation{}).Error
	})
}

func (s *Service) getConv(ctx context.Context, userID, convID uint64) (*models.Conversation, error) {
	var conv models.Conversation
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", convID, userID).Take(&conv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.ErrNotFound("会话不存在")
	}
	return &conv, err
}

// Handler 暴露对话接口。
type Handler struct{ svc *Service }

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/conversations", h.create)
	rg.GET("/conversations", h.list)
	rg.POST("/conversations/:id/messages", h.send)
	rg.GET("/conversations/:id/messages", h.messages)
	rg.POST("/conversations/:id/finish", h.finish)
	rg.DELETE("/conversations/:id", h.delete)
}

type createReq struct {
	Scene      string `json:"scene"`
	Difficulty string `json:"difficulty"`
	Role       string `json:"role"`
	Title      string `json:"title"`
}

func (h *Handler) create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	conv, err := h.svc.Create(c.Request.Context(), httpx.MustUserID(c), req.Scene, req.Difficulty, req.Role, req.Title)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, conv)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context(), httpx.MustUserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, items)
}

type sendReq struct {
	Content string `json:"content"`
}

func (h *Handler) send(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req sendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.Send(c.Request.Context(), httpx.MustUserID(c), id, req.Content)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) messages(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	msgs, err := h.svc.Messages(c.Request.Context(), httpx.MustUserID(c), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, msgs)
}

func (h *Handler) finish(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Finish(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"status": "finished"})
}

func (h *Handler) delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

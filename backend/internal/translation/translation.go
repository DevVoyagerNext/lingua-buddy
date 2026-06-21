// Package translation 提供英汉互译与用户译文对比。
package translation

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/history"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/user"
)

const maxTextLen = 5000

// Service 翻译服务。
type Service struct {
	ai      ai.Provider
	history *history.Repository
	level   user.LevelLookup
}

// NewService 构造翻译服务。
func NewService(provider ai.Provider, hist *history.Repository, level user.LevelLookup) *Service {
	return &Service{ai: provider, history: hist, level: level}
}

// Result 翻译结果（含历史 ID）。
type Result struct {
	Output    ai.TranslationOutput `json:"output"`
	HistoryID uint64               `json:"history_id"`
}

// Translate 翻译并保存历史。
func (s *Service) Translate(ctx context.Context, userID uint64, text, source, target, tone string) (*Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, httpx.ErrValidation("待翻译文本不能为空")
	}
	if utf8.RuneCountInString(text) > maxTextLen {
		return nil, httpx.ErrValidation("文本超过 5000 字限制")
	}
	if source == "" {
		source = "auto"
	}
	if target == "" {
		target = inferTarget(text)
	}
	out, err := s.ai.Translate(ctx, ai.TranslationInput{
		Text: text, SourceLang: source, TargetLang: target, Tone: tone,
		Level: s.level.Level(ctx, userID),
	})
	if err != nil {
		return nil, aiError(err)
	}
	rec := &models.HistoryRecord{
		UserID: userID, RecordType: history.TypeTranslation,
		InputText: text, ResultText: out.TranslatedText,
	}
	if err := s.history.Create(ctx, rec); err != nil {
		return nil, err
	}
	return &Result{Output: out, HistoryID: rec.ID}, nil
}

// Compare 用户译文对比（不入历史）。
func (s *Service) Compare(ctx context.Context, userID uint64, sourceText, userText string) (ai.TranslationCompareOutput, error) {
	sourceText = strings.TrimSpace(sourceText)
	userText = strings.TrimSpace(userText)
	if sourceText == "" || userText == "" {
		return ai.TranslationCompareOutput{}, httpx.ErrValidation("原文和译文都不能为空")
	}
	out, err := s.ai.CompareTranslation(ctx, ai.TranslationCompareInput{
		SourceText: sourceText, UserText: userText, Level: s.level.Level(ctx, userID),
	})
	if err != nil {
		return ai.TranslationCompareOutput{}, aiError(err)
	}
	return out, nil
}

func inferTarget(text string) string {
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return "en" // 含中文 → 译成英文
		}
	}
	return "zh"
}

func aiError(err error) error {
	status, code, msg := ai.ErrorCode(err)
	return httpx.NewError(status, code, msg)
}

// Handler 暴露翻译接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/translations", h.translate)
	rg.POST("/translations/compare", h.compare)
}

type translateReq struct {
	Text       string `json:"text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
	Tone       string `json:"tone"`
}

func (h *Handler) translate(c *gin.Context) {
	var req translateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.Translate(c.Request.Context(), httpx.MustUserID(c), req.Text, req.SourceLang, req.TargetLang, req.Tone)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

type compareReq struct {
	SourceText string `json:"source_text"`
	UserText   string `json:"user_text"`
}

func (h *Handler) compare(c *gin.Context) {
	var req compareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	out, err := h.svc.Compare(c.Request.Context(), httpx.MustUserID(c), req.SourceText, req.UserText)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, out)
}

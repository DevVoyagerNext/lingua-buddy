// Package grammar 提供语法分析、纠错与润色。
package grammar

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/history"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/user"
)

// Service 语法工具服务。
type Service struct {
	ai      ai.Provider
	history *history.Repository
	level   user.LevelLookup
}

// NewService 构造。
func NewService(provider ai.Provider, hist *history.Repository, level user.LevelLookup) *Service {
	return &Service{ai: provider, history: hist, level: level}
}

func aiError(err error) error {
	status, code, msg := ai.ErrorCode(err)
	return httpx.NewError(status, code, msg)
}

// Analyze 语法分析（结果以 JSON 存入历史 result_text）。
func (s *Service) Analyze(ctx context.Context, userID uint64, text string) (*ai.GrammarAnalysisOutput, uint64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, 0, httpx.ErrValidation("待分析文本不能为空")
	}
	out, err := s.ai.AnalyzeGrammar(ctx, ai.GrammarInput{Text: text, Level: s.level.Level(ctx, userID)})
	if err != nil {
		return nil, 0, aiError(err)
	}
	resultJSON, _ := json.Marshal(out)
	rec := &models.HistoryRecord{
		UserID: userID, RecordType: history.TypeGrammarAnalysis,
		InputText: text, ResultText: string(resultJSON),
	}
	if err := s.history.Create(ctx, rec); err != nil {
		return nil, 0, err
	}
	return &out, rec.ID, nil
}

// Correct 纠错或润色（结果文本存入历史）。
func (s *Service) Correct(ctx context.Context, userID uint64, text, mode, style string) (*ai.CorrectionOutput, uint64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, 0, httpx.ErrValidation("待处理文本不能为空")
	}
	if mode != "polish" {
		mode = "correct"
	}
	out, err := s.ai.Correct(ctx, ai.CorrectionInput{Text: text, Mode: mode, Style: style, Level: s.level.Level(ctx, userID)})
	if err != nil {
		return nil, 0, aiError(err)
	}
	rec := &models.HistoryRecord{
		UserID: userID, RecordType: history.TypeCorrection,
		InputText: text, ResultText: out.CorrectedText,
	}
	if err := s.history.Create(ctx, rec); err != nil {
		return nil, 0, err
	}
	return &out, rec.ID, nil
}

// Handler 暴露语法工具接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/grammar/analysis", h.analyze)
	rg.POST("/corrections", h.correct)
}

type analyzeReq struct {
	Text string `json:"text"`
}

func (h *Handler) analyze(c *gin.Context) {
	var req analyzeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	out, hid, err := h.svc.Analyze(c.Request.Context(), httpx.MustUserID(c), req.Text)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"analysis": out, "history_id": hid})
}

type correctReq struct {
	Text  string `json:"text"`
	Mode  string `json:"mode"`  // correct / polish
	Style string `json:"style"` // 润色风格
}

func (h *Handler) correct(c *gin.Context) {
	var req correctReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	out, hid, err := h.svc.Correct(c.Request.Context(), httpx.MustUserID(c), req.Text, req.Mode, req.Style)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"result": out, "history_id": hid})
}

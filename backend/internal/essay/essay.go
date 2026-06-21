// Package essay 提供作文批改与批改历史（含版本分组）。
package essay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/history"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/trainrec"
	"lingua-buddy/internal/user"
)

// Service 作文服务。
type Service struct {
	ai      ai.Provider
	records *trainrec.Repository
	history *history.Repository
	level   user.LevelLookup
}

// NewService 构造。
func NewService(provider ai.Provider, records *trainrec.Repository, hist *history.Repository, level user.LevelLookup) *Service {
	return &Service{ai: provider, records: records, history: hist, level: level}
}

func questionKey(userID uint64, title, exam string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(title)) + "|" + exam))
	return hex.EncodeToString(sum[:])[:32]
}

// ReviewInput 批改输入。
type ReviewInput struct {
	SubmissionID string
	Title        string
	Body         string
	EssayType    string
	Requirement  string
	TargetExam   string
}

// ReviewResult 批改结果。
type ReviewResult struct {
	RecordID uint64               `json:"record_id"`
	Review   ai.EssayReviewOutput `json:"review"`
}

// Review 先保存作文原文，再调用 AI 批改并写历史（同一作文按 question_key 分组）。
func (s *Service) Review(ctx context.Context, userID uint64, in ReviewInput) (*ReviewResult, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, httpx.ErrValidation("作文正文不能为空")
	}
	if in.SubmissionID == "" {
		return nil, httpx.ErrValidation("缺少 submission_id")
	}
	qkey := questionKey(userID, in.Title, in.TargetExam)
	rec := &models.TrainingAnswerRecord{
		UserID: userID, SubmissionID: in.SubmissionID,
		TrainingType: "essay", QuestionType: "essay_review", QuestionKey: qkey,
		AnswerSource: "essay_standalone", QuestionText: in.Title,
		UserAnswer: in.Body, EvaluationStatus: "pending",
	}
	saved, dup, err := s.records.CreateOrGet(ctx, rec)
	if err != nil {
		return nil, err
	}
	if dup && saved.EvaluationStatus == "completed" {
		var out ai.EssayReviewOutput
		_ = json.Unmarshal(saved.EvaluationResult, &out)
		return &ReviewResult{RecordID: saved.ID, Review: out}, nil
	}

	out, aiErr := s.ai.ReviewEssay(ctx, ai.EssayInput{
		Title: in.Title, Body: in.Body, EssayType: in.EssayType,
		Requirement: in.Requirement, TargetExam: in.TargetExam, Level: s.level.Level(ctx, userID),
	})
	if aiErr != nil {
		_ = s.records.UpdateEvaluation(ctx, saved.ID, trainrec.EvaluationUpdate{Status: "failed"})
		status, code, msg := ai.ErrorCode(aiErr)
		return nil, httpx.NewError(status, code, msg)
	}

	// 写统一历史。
	histRec := &models.HistoryRecord{
		UserID: userID, RecordType: history.TypeEssay,
		InputText: in.Title + "\n" + in.Body, ResultText: out.OverallComment + "\n\n" + out.RevisedText,
	}
	if err := s.history.Create(ctx, histRec); err != nil {
		return nil, err
	}
	evalJSON, _ := json.Marshal(out)
	if err := s.records.UpdateEvaluation(ctx, saved.ID, trainrec.EvaluationUpdate{
		EvaluationResult: evalJSON, Status: "completed", HistoryRecordID: &histRec.ID,
	}); err != nil {
		return nil, err
	}
	return &ReviewResult{RecordID: saved.ID, Review: out}, nil
}

// History 批改历史（training_type=essay）。
func (s *Service) History(ctx context.Context, userID uint64, questionKey string, page, size int) ([]models.TrainingAnswerRecord, int64, error) {
	return s.records.List(ctx, userID, "essay", questionKey, page, size)
}

// Handler 暴露作文接口。
type Handler struct{ svc *Service }

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/essays/review", h.review)
	rg.GET("/essays/history", h.history)
}

type reviewReq struct {
	SubmissionID string `json:"submission_id"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	EssayType    string `json:"essay_type"`
	Requirement  string `json:"requirement"`
	TargetExam   string `json:"target_exam"`
}

func (h *Handler) review(c *gin.Context) {
	var req reviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.Review(c.Request.Context(), httpx.MustUserID(c), ReviewInput{
		SubmissionID: req.SubmissionID, Title: req.Title, Body: req.Body,
		EssayType: req.EssayType, Requirement: req.Requirement, TargetExam: req.TargetExam,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) history(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	rows, total, err := h.svc.History(c.Request.Context(), httpx.MustUserID(c), c.Query("question_key"), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: rows, Page: page, PageSize: size, Total: total})
}

// Package training 提供翻译/作文专项训练：出题、提交评价、答题记录、翻译错题。
package training

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/trainrec"
	"lingua-buddy/internal/user"
)

// Service 训练服务。
type Service struct {
	db      *gorm.DB
	ai      ai.Provider
	records *trainrec.Repository
	level   user.LevelLookup
}

// NewService 构造。
func NewService(db *gorm.DB, provider ai.Provider, records *trainrec.Repository, level user.LevelLookup) *Service {
	return &Service{db: db, ai: provider, records: records, level: level}
}

func aiErr(err error) error {
	status, code, msg := ai.ErrorCode(err)
	return httpx.NewError(status, code, msg)
}

func transKey(direction, source string) string {
	sum := sha256.Sum256([]byte(direction + "|" + strings.ToLower(strings.TrimSpace(source))))
	return hex.EncodeToString(sum[:])[:32]
}

// ===== 翻译训练 =====

// NextTranslation 生成一句待翻译文本（不入库）。
func (s *Service) NextTranslation(ctx context.Context, userID uint64, direction, difficulty string) (string, error) {
	if direction != "en_to_zh" {
		direction = "zh_to_en"
	}
	ex, err := s.ai.GenerateTranslationExercise(ctx, ai.TranslationExerciseInput{
		Direction: direction, Difficulty: difficulty, Level: s.level.Level(ctx, userID),
	})
	if err != nil {
		return "", aiErr(err)
	}
	return ex.Text, nil
}

// EvaluateInput 翻译训练评价输入。
type EvaluateInput struct {
	SubmissionID string
	Direction    string
	SourceText   string
	UserAnswer   string
}

// EvaluateResult 翻译训练评价结果。
type EvaluateResult struct {
	RecordID   uint64                   `json:"record_id"`
	Evaluation ai.TranslationEvaluation `json:"evaluation"`
}

// EvaluateTranslation 先保存译文，再调用 AI 评价。
func (s *Service) EvaluateTranslation(ctx context.Context, userID uint64, in EvaluateInput) (*EvaluateResult, error) {
	if in.SubmissionID == "" {
		return nil, httpx.ErrValidation("缺少 submission_id")
	}
	if strings.TrimSpace(in.UserAnswer) == "" || strings.TrimSpace(in.SourceText) == "" {
		return nil, httpx.ErrValidation("原文和译文不能为空")
	}
	if in.Direction != "en_to_zh" {
		in.Direction = "zh_to_en"
	}
	rec := &models.TrainingAnswerRecord{
		UserID: userID, SubmissionID: in.SubmissionID,
		TrainingType: "translation", QuestionType: in.Direction, QuestionKey: transKey(in.Direction, in.SourceText),
		AnswerSource: "translation_training", QuestionText: in.SourceText,
		UserAnswer: in.UserAnswer, EvaluationStatus: "pending",
	}
	saved, dup, err := s.records.CreateOrGet(ctx, rec)
	if err != nil {
		return nil, err
	}
	if dup && saved.EvaluationStatus == "completed" {
		var out ai.TranslationEvaluation
		_ = json.Unmarshal(saved.EvaluationResult, &out)
		return &EvaluateResult{RecordID: saved.ID, Evaluation: out}, nil
	}

	out, aerr := s.ai.EvaluateTranslation(ctx, ai.TranslationEvaluationInput{
		Direction: in.Direction, SourceText: in.SourceText, UserText: in.UserAnswer, Level: s.level.Level(ctx, userID),
	})
	if aerr != nil {
		_ = s.records.UpdateEvaluation(ctx, saved.ID, trainrec.EvaluationUpdate{Status: "failed"})
		return nil, aiErr(aerr)
	}
	evalJSON, _ := json.Marshal(out)
	ref := out.ReferenceText
	if err := s.records.UpdateEvaluation(ctx, saved.ID, trainrec.EvaluationUpdate{
		ReferenceAnswer: &ref, EvaluationResult: evalJSON, Status: "completed",
	}); err != nil {
		return nil, err
	}
	return &EvaluateResult{RecordID: saved.ID, Evaluation: out}, nil
}

// RetryEvaluation 重试 failed 或超时 pending 的翻译评价（不改写用户答案）。
func (s *Service) RetryEvaluation(ctx context.Context, userID, recordID uint64) (*EvaluateResult, error) {
	rec, err := s.records.GetByID(ctx, userID, recordID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.ErrNotFound("答题记录不存在")
	}
	if err != nil {
		return nil, err
	}
	if rec.TrainingType != "translation" {
		return nil, httpx.ErrValidation("仅翻译训练支持此重试")
	}
	out, aerr := s.ai.EvaluateTranslation(ctx, ai.TranslationEvaluationInput{
		Direction: rec.QuestionType, SourceText: rec.QuestionText, UserText: rec.UserAnswer, Level: s.level.Level(ctx, userID),
	})
	if aerr != nil {
		_ = s.records.UpdateEvaluation(ctx, rec.ID, trainrec.EvaluationUpdate{Status: "failed"})
		return nil, aiErr(aerr)
	}
	evalJSON, _ := json.Marshal(out)
	ref := out.ReferenceText
	if err := s.records.UpdateEvaluation(ctx, rec.ID, trainrec.EvaluationUpdate{
		ReferenceAnswer: &ref, EvaluationResult: evalJSON, Status: "completed",
	}); err != nil {
		return nil, err
	}
	return &EvaluateResult{RecordID: rec.ID, Evaluation: out}, nil
}

// ConfirmWrong 用户确认翻译答案错误，加入翻译错题。
func (s *Service) ConfirmWrong(ctx context.Context, userID, recordID uint64) error {
	rec, err := s.records.GetByID(ctx, userID, recordID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return httpx.ErrNotFound("答题记录不存在")
	}
	if err != nil {
		return err
	}
	if rec.TrainingType != "translation" {
		return httpx.ErrValidation("仅翻译训练可加入翻译错题")
	}
	wrong := models.UserTranslationWrongQuestion{
		UserID: userID, Direction: rec.QuestionType, QuestionKey: rec.QuestionKey,
		QuestionText: rec.QuestionText, ReferenceAnswer: rec.ReferenceAnswer,
		UserAnswer: &rec.UserAnswer, WrongCount: 1, LastAnswerRecordID: &rec.ID,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "question_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"wrong_count":           gorm.Expr("wrong_count + 1"),
			"user_answer":           rec.UserAnswer,
			"reference_answer":      rec.ReferenceAnswer,
			"last_answer_record_id": rec.ID,
			"updated_at":            time.Now(),
		}),
	}).Create(&wrong).Error
}

// ===== 作文训练出题 =====

// EssayTopic 生成作文题目。
func (s *Service) EssayTopic(ctx context.Context, userID uint64, essayType, difficulty string) (ai.EssayTopic, error) {
	out, err := s.ai.GenerateEssayTopic(ctx, ai.EssayTopicInput{EssayType: essayType, Difficulty: difficulty, Level: s.level.Level(ctx, userID)})
	if err != nil {
		return ai.EssayTopic{}, aiErr(err)
	}
	return out, nil
}

// ===== 答题记录与翻译错题查询 =====

// AnswerView 答题记录视图（含有效评价状态）。
type AnswerView struct {
	models.TrainingAnswerRecord
	EffectiveStatus string `json:"effective_status"`
}

// ListAnswers 分页查询答题记录。
func (s *Service) ListAnswers(ctx context.Context, userID uint64, trainingType, questionKey string, page, size int) ([]AnswerView, int64, error) {
	items, total, err := s.records.List(ctx, userID, trainingType, questionKey, page, size)
	if err != nil {
		return nil, 0, err
	}
	views := make([]AnswerView, 0, len(items))
	for i := range items {
		views = append(views, AnswerView{TrainingAnswerRecord: items[i], EffectiveStatus: trainrec.EffectiveStatus(&items[i])})
	}
	return views, total, nil
}

// ListWrongQuestions 查询翻译错题。
func (s *Service) ListWrongQuestions(ctx context.Context, userID uint64, page, size int) ([]models.UserTranslationWrongQuestion, int64, error) {
	tx := s.db.WithContext(ctx).Model(&models.UserTranslationWrongQuestion{}).Where("user_id = ?", userID)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.UserTranslationWrongQuestion
	err := tx.Order("updated_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// DeleteWrongQuestion 确认解决并移除翻译错题。
func (s *Service) DeleteWrongQuestion(ctx context.Context, userID, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.UserTranslationWrongQuestion{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.ErrNotFound("翻译错题不存在")
	}
	return nil
}

// Handler 暴露训练接口。
type Handler struct{ svc *Service }

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/training/translations/next", h.nextTranslation)
	rg.POST("/training/translations/evaluate", h.evaluateTranslation)
	rg.POST("/training/essays/topic", h.essayTopic)
	rg.GET("/training/answers", h.listAnswers)
	rg.GET("/training/answers/:id", h.getAnswer)
	rg.POST("/training/answers/:id/retry-evaluation", h.retry)
	rg.POST("/training/answers/:id/confirm-wrong", h.confirmWrong)
	rg.GET("/translation-wrong-questions", h.listWrong)
	rg.DELETE("/translation-wrong-questions/:id", h.deleteWrong)
}

type nextTransReq struct {
	Direction  string `json:"direction"`
	Difficulty string `json:"difficulty"`
}

func (h *Handler) nextTranslation(c *gin.Context) {
	var req nextTransReq
	_ = c.ShouldBindJSON(&req)
	text, err := h.svc.NextTranslation(c.Request.Context(), httpx.MustUserID(c), req.Direction, req.Difficulty)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"text": text, "direction": req.Direction})
}

type evalReq struct {
	SubmissionID string `json:"submission_id"`
	Direction    string `json:"direction"`
	SourceText   string `json:"source_text"`
	UserAnswer   string `json:"user_answer"`
}

func (h *Handler) evaluateTranslation(c *gin.Context) {
	var req evalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.EvaluateTranslation(c.Request.Context(), httpx.MustUserID(c), EvaluateInput{
		SubmissionID: req.SubmissionID, Direction: req.Direction, SourceText: req.SourceText, UserAnswer: req.UserAnswer,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

type essayTopicReq struct {
	EssayType  string `json:"essay_type"`
	Difficulty string `json:"difficulty"`
}

func (h *Handler) essayTopic(c *gin.Context) {
	var req essayTopicReq
	_ = c.ShouldBindJSON(&req)
	out, err := h.svc.EssayTopic(c.Request.Context(), httpx.MustUserID(c), req.EssayType, req.Difficulty)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, out)
}

func (h *Handler) listAnswers(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.svc.ListAnswers(c.Request.Context(), httpx.MustUserID(c), c.Query("training_type"), c.Query("question_key"), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) getAnswer(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rec, err := h.svc.records.GetByID(c.Request.Context(), httpx.MustUserID(c), id)
	if err != nil {
		httpx.Fail(c, httpx.ErrNotFound("答题记录不存在"))
		return
	}
	httpx.OK(c, AnswerView{TrainingAnswerRecord: *rec, EffectiveStatus: trainrec.EffectiveStatus(rec)})
}

func (h *Handler) retry(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	res, err := h.svc.RetryEvaluation(c.Request.Context(), httpx.MustUserID(c), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) confirmWrong(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.ConfirmWrong(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) listWrong(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.svc.ListWrongQuestions(c.Request.Context(), httpx.MustUserID(c), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) deleteWrong(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.DeleteWrongQuestion(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

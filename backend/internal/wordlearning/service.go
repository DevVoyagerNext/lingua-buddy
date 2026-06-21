package wordlearning

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/lexicon"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/worddistractor"
)

// 自定义错误码。
func errActivePlanConflict() error {
	return httpx.NewError(http.StatusConflict, "ACTIVE_WORD_PLAN_CONFLICT", "已存在一个进行中的词汇计划")
}
func errInsufficientDistractors() error {
	return httpx.NewError(http.StatusUnprocessableEntity, "INSUFFICIENT_DISTRACTORS", "无法生成足够的干扰项")
}
func errTokenInvalid() error {
	return httpx.NewError(http.StatusBadRequest, "QUESTION_TOKEN_INVALID", "题目令牌无效")
}
func errTokenExpired() error {
	return httpx.NewError(http.StatusConflict, "QUESTION_TOKEN_EXPIRED", "题目已过期或阶段已变化，请重新获取")
}

// Service 单词学习与生词本服务。
type Service struct {
	repo       *Repository
	lex        *lexicon.Repository
	distractor *worddistractor.Service
	builder    *QuestionBuilder
	tokens     *TokenManager
}

// NewService 构造服务。
func NewService(repo *Repository, lex *lexicon.Repository, distractor *worddistractor.Service, tokens *TokenManager) *Service {
	return &Service{
		repo:       repo,
		lex:        lex,
		distractor: distractor,
		builder:    NewQuestionBuilder(tokens),
		tokens:     tokens,
	}
}

func normalizeWord(w string) string { return strings.ToLower(strings.TrimSpace(w)) }

// ===== 生词本 =====

// CollectWord 收藏生词，按熟悉度设定初始阶段与首次复习时间。
func (s *Service) CollectWord(ctx context.Context, userID uint64, word, familiarity string) (*models.UserWord, error) {
	norm := strings.TrimSpace(word)
	if norm == "" {
		return nil, httpx.ErrValidation("单词不能为空")
	}
	entry, err := s.lex.FindExact(ctx, norm)
	if errors.Is(err, lexicon.ErrNotFound) {
		return nil, httpx.ErrNotFound("词典中没有该单词，无法收藏")
	}
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.FindUserWord(ctx, userID, entry.Word)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, httpx.ErrConflict("该单词已在生词本")
	}

	now := time.Now()
	stage, delay := InitialStageFor(familiarity)
	uw := &models.UserWord{
		UserID:         userID,
		ECDICTEntryID:  &entry.ID,
		Word:           entry.Word,
		LearningStage:  stage,
		NextReviewAt:   now.Add(delay),
		StageChangedAt: now,
	}
	if stage == StageMastered {
		uw.FirstMasteredAt = &now
	}
	if err := s.repo.CreateUserWord(ctx, uw); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.ErrConflict("该单词已在生词本")
		}
		return nil, err
	}
	return uw, nil
}

// RemoveWord 移出生词本。
func (s *Service) RemoveWord(ctx context.Context, userID, id uint64) error {
	if err := s.repo.DeleteUserWord(ctx, userID, id); err != nil {
		return httpx.ErrNotFound("生词不存在")
	}
	return nil
}

// ListWords 生词列表。
func (s *Service) ListWords(ctx context.Context, userID uint64, stage, keyword string, page, size int) ([]models.UserWord, int64, error) {
	return s.repo.ListUserWords(ctx, userID, stage, keyword, page, size)
}

// ===== 计划 =====

// CreatePlan 创建词汇计划：拉取标签词、过滤词组、频率分桶打乱、批量入队。
func (s *Service) CreatePlan(ctx context.Context, userID uint64, name, sourceValue string) (*models.WordLearningPlan, PlanCounts, error) {
	if _, ok := map[string]bool{"cet4": true, "cet6": true, "zk": true, "gk": true, "ky": true, "toefl": true, "ielts": true, "gre": true}[sourceValue]; !ok {
		return nil, PlanCounts{}, httpx.ErrValidation("不支持的词表标签")
	}
	active, err := s.repo.GetActivePlan(ctx, userID)
	if err != nil {
		return nil, PlanCounts{}, err
	}
	if active != nil {
		return nil, PlanCounts{}, errActivePlanConflict()
	}

	entries, err := s.lex.ListByExamTag(ctx, sourceValue)
	if err != nil {
		return nil, PlanCounts{}, err
	}
	if len(entries) == 0 {
		return nil, PlanCounts{}, httpx.ErrValidation("该词表没有可用单词")
	}

	seed := rand.Int63()
	ordered := frequencyShuffle(entries, seed)

	if name == "" {
		name = defaultPlanName(sourceValue)
	}
	now := time.Now()
	plan := &models.WordLearningPlan{
		UserID:              userID,
		Name:                name,
		SourceType:          "ecdict_tag",
		SourceValue:         sourceValue,
		OrderingMode:        "frequency_shuffled",
		ShuffleSeed:         seed,
		SourceSnapshotCount: len(ordered),
		DailyNewWordLimit:   10,
		ActiveWordLimit:     20,
		Status:              "active",
		StartedAt:           &now,
	}
	items := make([]models.WordLearningPlanItem, 0, len(ordered))
	for i, e := range ordered {
		items = append(items, models.WordLearningPlanItem{
			ECDICTEntryID: e.ID,
			Word:          e.Word,
			QueuePosition: i + 1,
		})
	}
	if err := s.repo.CreatePlanWithItems(ctx, plan, items); err != nil {
		return nil, PlanCounts{}, err
	}
	counts, err := s.repo.CountPlanItems(ctx, plan.ID)
	if err != nil {
		return nil, PlanCounts{}, err
	}
	return plan, counts, nil
}

func defaultPlanName(tag string) string {
	switch tag {
	case "cet4":
		return "四级词汇"
	case "cet6":
		return "六级词汇"
	default:
		return tag + " 词汇"
	}
}

// frequencyShuffle 频率分桶 + 桶内固定种子打乱。
func frequencyShuffle(entries []lexicon.Entry, seed int64) []lexicon.Entry {
	sorted := make([]lexicon.Entry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		fi, fj := freqVal(sorted[i]), freqVal(sorted[j])
		return fi < fj
	})
	rng := rand.New(rand.NewSource(seed))
	const bucket = 50
	for start := 0; start < len(sorted); start += bucket {
		end := start + bucket
		if end > len(sorted) {
			end = len(sorted)
		}
		b := sorted[start:end]
		for i := len(b) - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			b[i], b[j] = b[j], b[i]
		}
	}
	return sorted
}

func freqVal(e lexicon.Entry) int {
	if e.FrequencyRank != nil && *e.FrequencyRank > 0 {
		return *e.FrequencyRank
	}
	return 1 << 30
}

// ListPlans 列出计划。
func (s *Service) ListPlans(ctx context.Context, userID uint64) ([]models.WordLearningPlan, error) {
	return s.repo.ListPlans(ctx, userID)
}

// PlanDetail 计划及进度。
type PlanDetail struct {
	Plan   *models.WordLearningPlan `json:"plan"`
	Counts PlanCounts               `json:"counts"`
}

// GetPlan 计划详情。
func (s *Service) GetPlan(ctx context.Context, userID, id uint64) (*PlanDetail, error) {
	p, err := s.repo.GetPlan(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.ErrNotFound("计划不存在")
	}
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountPlanItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return &PlanDetail{Plan: p, Counts: counts}, nil
}

// ActivatePlan 激活/切换计划：要求当前无其他 active 计划。
func (s *Service) ActivatePlan(ctx context.Context, userID, id uint64) error {
	active, err := s.repo.GetActivePlan(ctx, userID)
	if err != nil {
		return err
	}
	if active != nil && active.ID != id {
		return errActivePlanConflict()
	}
	if err := s.repo.SetPlanStatus(ctx, userID, id, "active"); err != nil {
		return httpx.ErrNotFound("计划不存在")
	}
	return nil
}

// PausePlan 暂停计划。
func (s *Service) PausePlan(ctx context.Context, userID, id uint64) error {
	if err := s.repo.SetPlanStatus(ctx, userID, id, "paused"); err != nil {
		return httpx.ErrNotFound("计划不存在")
	}
	return nil
}

// SkipItem 跳过计划词。
func (s *Service) SkipItem(ctx context.Context, userID, itemID uint64) error {
	if err := s.repo.SkipPlanItem(ctx, userID, itemID, time.Now()); err != nil {
		return httpx.ErrNotFound("计划词不存在")
	}
	return nil
}

// ListPlanWords 计划词条及阶段。
func (s *Service) ListPlanWords(ctx context.Context, userID, planID uint64, page, size int) ([]PlanWordRow, int64, error) {
	if _, err := s.repo.GetPlan(ctx, userID, planID); err != nil {
		return nil, 0, httpx.ErrNotFound("计划不存在")
	}
	return s.repo.ListPlanWords(ctx, planID, page, size)
}

// ListWrongAnswers 历史单词错误答案。
func (s *Service) ListWrongAnswers(ctx context.Context, userID uint64, page, size int) ([]models.TrainingAnswerRecord, int64, error) {
	return s.repo.ListWrongWordAnswers(ctx, userID, page, size)
}

// ===== 取题 =====

// NextResult 取题结果：题目或状态。
type NextResult struct {
	Question  *QuestionView
	Status    string // "" / NO_DUE_WORDS / NO_ACTIVE_PLAN
	NextDueAt *time.Time
}

// Next 获取下一道到期题，必要时激活新词。
func (s *Service) Next(ctx context.Context, userID uint64) (*NextResult, error) {
	now := time.Now()

	due, err := s.repo.FindDueWords(ctx, userID, now, 1)
	if err != nil {
		return nil, err
	}
	if len(due) > 0 {
		q, err := s.buildForUserWord(ctx, &due[0])
		if err != nil {
			return nil, err
		}
		return &NextResult{Question: q}, nil
	}

	plan, err := s.repo.GetActivePlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if plan != nil {
		picked, err := s.repo.ActivateAndPickDue(ctx, userID, plan, now)
		if err != nil {
			return nil, err
		}
		if picked != nil {
			q, err := s.buildForUserWord(ctx, picked)
			if err != nil {
				return nil, err
			}
			return &NextResult{Question: q}, nil
		}
	}

	// 没有到期题，也没有可激活的新词。
	total, err := s.repo.CountUserWords(ctx, userID)
	if err != nil {
		return nil, err
	}
	if total == 0 && plan == nil {
		return &NextResult{Status: "NO_ACTIVE_PLAN"}, nil
	}
	return &NextResult{Status: "NO_DUE_WORDS"}, nil
}

func (s *Service) buildForUserWord(ctx context.Context, uw *models.UserWord) (*QuestionView, error) {
	entry, err := s.loadEntry(ctx, uw)
	if err != nil {
		return nil, err
	}
	var planID, planItemID *uint64 // 计划字段由令牌可选携带；散收词为空（出题不强依赖）
	qtype := StageForQuestionType(uw.LearningStage)
	var distractors []string
	switch qtype {
	case QTypeWordToMeaningChoice:
		d, _, err := s.distractor.FindMeaningDistractors(ctx, entry, 3)
		if err != nil {
			return nil, errInsufficientDistractors()
		}
		distractors = d
	case QTypeMeaningToWordChoice:
		d, _, err := s.distractor.FindWordDistractors(ctx, entry, 3)
		if err != nil {
			return nil, errInsufficientDistractors()
		}
		distractors = d
	}
	return s.builder.Build(entry, uw, planID, planItemID, distractors)
}

func (s *Service) loadEntry(ctx context.Context, uw *models.UserWord) (*lexicon.Entry, error) {
	if uw.ECDICTEntryID != nil {
		if e, err := s.lex.GetByID(ctx, *uw.ECDICTEntryID); err == nil {
			return e, nil
		}
	}
	e, err := s.lex.FindExact(ctx, uw.Word)
	if errors.Is(err, lexicon.ErrNotFound) {
		return nil, httpx.NewError(http.StatusUnprocessableEntity, "DICTIONARY_UNAVAILABLE", "该单词的词典数据不可用")
	}
	return e, err
}

// ===== 提交答案 =====

// SubmitResult 提交结果。
type SubmitResult struct {
	Correct       bool      `json:"correct"`
	CorrectAnswer string    `json:"correct_answer"`
	StageBefore   string    `json:"stage_before"`
	StageAfter    string    `json:"stage_after"`
	NextReviewAt  time.Time `json:"next_review_at"`
	Duplicate     bool      `json:"duplicate"`
}

// SubmitAnswer 校验令牌、重判分、按规则推进阶段并落库。
func (s *Service) SubmitAnswer(ctx context.Context, userID uint64, submissionID, tokenStr, answer string, usedHint bool) (*SubmitResult, error) {
	if submissionID == "" {
		return nil, httpx.ErrValidation("缺少 submission_id")
	}
	if strings.TrimSpace(answer) == "" {
		return nil, httpx.ErrValidation("答案不能为空")
	}
	tok, err := s.tokens.Parse(tokenStr)
	if errors.Is(err, ErrTokenExpired) {
		return nil, errTokenExpired()
	}
	if err != nil {
		return nil, errTokenInvalid()
	}
	if tok.UserID != userID {
		return nil, errTokenInvalid()
	}

	uw, err := s.repo.GetUserWordByID(ctx, userID, tok.UserWordID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errTokenInvalid()
	}
	if err != nil {
		return nil, err
	}
	if uw.LearningStage != tok.Stage {
		return nil, errTokenExpired()
	}

	entry, err := s.loadEntry(ctx, uw)
	if err != nil {
		return nil, err
	}

	correct, correctAnswer := judge(tok.QuestionType, answer, entry)
	now := time.Now()
	decision := Apply(tok.Stage, uw.StageCorrectStreak, correct, usedHint, now)
	becameFirstMastered := decision.BecameMastered && uw.FirstMasteredAt == nil

	// 组装更新后的用户单词状态。
	newUW := *uw
	newUW.LearningStage = decision.Stage
	newUW.StageCorrectStreak = decision.Streak
	newUW.NextReviewAt = decision.NextReviewAt
	newUW.LastTrainedAt = &now
	if decision.Stage != uw.LearningStage {
		newUW.StageChangedAt = now
	}
	if correct {
		newUW.TotalCorrectCount++
	} else {
		newUW.TotalWrongCount++
	}
	newUW.LastAnswerCorrect = &correct
	if becameFirstMastered {
		newUW.FirstMasteredAt = &now
	}

	optionsJSON, _ := json.Marshal(tok.Options)
	stageBefore := tok.Stage
	stageAfter := decision.Stage
	genVersion := tok.GenVersion
	rec := &models.TrainingAnswerRecord{
		UserID:                 userID,
		SubmissionID:           submissionID,
		TrainingType:           "word",
		QuestionType:           tok.QuestionType,
		QuestionKey:            tok.QuestionKey,
		AnswerSource:           "word_learning",
		UserWordID:             &uw.ID,
		WordLearningPlanID:     tok.PlanID,
		WordLearningPlanItemID: tok.PlanItemID,
		QuestionText:           questionTextFor(tok.QuestionType, entry),
		Options:                optionsJSON,
		UserAnswer:             strings.TrimSpace(answer),
		ReferenceAnswer:        &correctAnswer,
		IsCorrect:              &correct,
		UsedHint:               usedHint,
		LearningStageBefore:    &stageBefore,
		LearningStageAfter:     &stageAfter,
		GeneratorVersion:       &genVersion,
		EvaluationStatus:       "completed",
		SubmittedAt:            now,
	}

	saved, duplicate, err := s.repo.SubmitAnswer(ctx, SubmitInput{
		Record:              rec,
		UserWord:            &newUW,
		ExpectedStage:       tok.Stage,
		PlanID:              tok.PlanID,
		BecameFirstMastered: becameFirstMastered,
		Now:                 now,
	})
	if errors.Is(err, ErrStageChanged) {
		return nil, errTokenExpired()
	}
	if err != nil {
		return nil, err
	}

	if duplicate {
		return &SubmitResult{
			Correct:       saved.IsCorrect != nil && *saved.IsCorrect,
			CorrectAnswer: derefStr(saved.ReferenceAnswer),
			StageBefore:   derefStr(saved.LearningStageBefore),
			StageAfter:    derefStr(saved.LearningStageAfter),
			NextReviewAt:  newUW.NextReviewAt,
			Duplicate:     true,
		}, nil
	}

	return &SubmitResult{
		Correct:       correct,
		CorrectAnswer: correctAnswer,
		StageBefore:   stageBefore,
		StageAfter:    stageAfter,
		NextReviewAt:  decision.NextReviewAt,
	}, nil
}

// judge 重判分：从 ecdict 重新得到正确答案，不信任客户端。
func judge(qtype, answer string, entry *lexicon.Entry) (correct bool, correctAnswer string) {
	switch qtype {
	case QTypeWordToMeaningChoice:
		gloss := entry.CanonicalGlossOf()
		return strings.TrimSpace(answer) == gloss, gloss
	default: // meaning_to_word_choice / spelling：答案是英文单词
		return normalizeWord(answer) == normalizeWord(entry.Word), entry.Word
	}
}

func questionTextFor(qtype string, entry *lexicon.Entry) string {
	if qtype == QTypeWordToMeaningChoice {
		return entry.Word
	}
	return entry.CanonicalGlossOf()
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

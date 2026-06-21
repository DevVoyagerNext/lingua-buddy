package wordlearning

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
)

// ===== 仓库：组进度与按位置取词 =====

// ListPlanItemsByPosition 取计划中 [from, to] 队列位置的词条（按位置升序）。
func (r *Repository) ListPlanItemsByPosition(ctx context.Context, planID uint64, from, to int) ([]models.WordLearningPlanItem, error) {
	var items []models.WordLearningPlanItem
	err := r.db.WithContext(ctx).
		Where("plan_id = ? AND queue_position BETWEEN ? AND ?", planID, from, to).
		Order("queue_position ASC").Find(&items).Error
	return items, err
}

// ListGroupProgress 取某计划全部组进度。
func (r *Repository) ListGroupProgress(ctx context.Context, userID, planID uint64) ([]models.WordGroupProgress, error) {
	var rows []models.WordGroupProgress
	err := r.db.WithContext(ctx).Where("user_id = ? AND plan_id = ?", userID, planID).Find(&rows).Error
	return rows, err
}

// GetGroupProgress 取某一组进度；不存在返回 nil,nil。
func (r *Repository) GetGroupProgress(ctx context.Context, userID, planID uint64, index int) (*models.WordGroupProgress, error) {
	var p models.WordGroupProgress
	err := r.db.WithContext(ctx).Where("user_id = ? AND plan_id = ? AND group_index = ?", userID, planID, index).Take(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertGroupProgress 新增或更新组进度。
func (r *Repository) UpsertGroupProgress(ctx context.Context, p *models.WordGroupProgress) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "plan_id"}, {Name: "group_index"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"review_count", "first_studied_at", "last_studied_at", "next_review_at", "updated_at",
		}),
	}).Create(p).Error
}

// ===== 服务 =====

// GroupSummary 组概览。
type GroupSummary struct {
	Index       int    `json:"index"`
	From        int    `json:"from"`
	To          int    `json:"to"`
	WordCount   int    `json:"word_count"`
	Status      string `json:"status"` // new / learned / due
	ReviewCount int    `json:"review_count"`
}

// GroupListResult 组列表。
type GroupListResult struct {
	PlanID      uint64         `json:"plan_id"`
	PlanName    string         `json:"plan_name"`
	GroupSize   int            `json:"group_size"`
	TotalWords  int            `json:"total_words"`
	TotalGroups int            `json:"total_groups"`
	Groups      []GroupSummary `json:"groups"`
}

// GroupWord 组内单词的学习数据（含正确答案与选项，供前端逐词走流程）。
type GroupWord struct {
	Word           string   `json:"word"`
	Phonetic       string   `json:"phonetic"`
	Definitions    []string `json:"definitions"`
	Translations   []string `json:"translations"`
	Gloss          string   `json:"gloss"`           // 英选汉的正确答案
	MeaningOptions []string `json:"meaning_options"` // 英选汉 4 选项（含正确）
	WordOptions    []string `json:"word_options"`    // 汉选英 4 选项（含正确）
}

// GroupDetail 一组的学习内容。
type GroupDetail struct {
	Index       int         `json:"index"`
	IsFirstTime bool        `json:"is_first_time"` // 首学含详细释义步骤
	GroupSize   int         `json:"group_size"`
	Words       []GroupWord `json:"words"`
}

func (s *Service) requireActivePlan(ctx context.Context, userID uint64) (*models.WordLearningPlan, error) {
	plan, err := s.repo.GetActivePlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, httpx.NewError(http.StatusBadRequest, "NO_ACTIVE_PLAN", "还没有进行中的单词书，请先在「单词书」里创建/进入一本")
	}
	if plan.GroupSize <= 0 {
		plan.GroupSize = 20
	}
	return plan, nil
}

// ListGroups 列出当前单词书的所有组及状态。
func (s *Service) ListGroups(ctx context.Context, userID uint64) (*GroupListResult, error) {
	plan, err := s.requireActivePlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	size := plan.GroupSize
	total := plan.SourceSnapshotCount
	totalGroups := (total + size - 1) / size

	progressRows, err := s.repo.ListGroupProgress(ctx, userID, plan.ID)
	if err != nil {
		return nil, err
	}
	byIndex := make(map[int]models.WordGroupProgress, len(progressRows))
	for _, p := range progressRows {
		byIndex[p.GroupIndex] = p
	}

	now := time.Now()
	groups := make([]GroupSummary, 0, totalGroups)
	for i := 0; i < totalGroups; i++ {
		from := i*size + 1
		to := (i + 1) * size
		if to > total {
			to = total
		}
		g := GroupSummary{Index: i, From: from, To: to, WordCount: to - from + 1, Status: "new"}
		if p, ok := byIndex[i]; ok {
			g.ReviewCount = p.ReviewCount
			if p.NextReviewAt != nil && !p.NextReviewAt.After(now) {
				g.Status = "due"
			} else {
				g.Status = "learned"
			}
		}
		groups = append(groups, g)
	}
	return &GroupListResult{
		PlanID: plan.ID, PlanName: plan.Name, GroupSize: size,
		TotalWords: total, TotalGroups: totalGroups, Groups: groups,
	}, nil
}

// GetGroup 取某一组的学习内容（含选项与正确答案）。
func (s *Service) GetGroup(ctx context.Context, userID uint64, index int) (*GroupDetail, error) {
	plan, err := s.requireActivePlan(ctx, userID)
	if err != nil {
		return nil, err
	}
	if index < 0 {
		return nil, httpx.ErrValidation("无效的组序号")
	}
	size := plan.GroupSize
	from := index*size + 1
	to := (index + 1) * size
	items, err := s.repo.ListPlanItemsByPosition(ctx, plan.ID, from, to)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, httpx.ErrNotFound("该组不存在")
	}

	progress, err := s.repo.GetGroupProgress(ctx, userID, plan.ID, index)
	if err != nil {
		return nil, err
	}
	isFirstTime := progress == nil || progress.ReviewCount == 0

	// 每个词要做 2 次干扰项查询，串行会很慢；用有界并发并行生成，保持原顺序。
	built := make([]GroupWord, len(items))
	okFlags := make([]bool, len(items))
	const workers = 8
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			entry, err := s.lex.GetByID(ctx, items[i].ECDICTEntryID)
			if err != nil {
				return // 词条缺失则跳过
			}
			gloss := entry.CanonicalGlossOf()
			if gloss == "" {
				return // 无有效中文释义无法做选择题，跳过
			}
			gw := GroupWord{
				Word: entry.Word, Phonetic: entry.Phonetic,
				Definitions: entry.Definitions, Translations: entry.Translations, Gloss: gloss,
			}
			if md, _, e := s.distractor.FindMeaningDistractors(ctx, entry, 3); e == nil {
				gw.MeaningOptions = shuffle(append([]string{gloss}, md...))
			} else {
				gw.MeaningOptions = []string{gloss}
			}
			if wd, _, e := s.distractor.FindWordDistractors(ctx, entry, 3); e == nil {
				gw.WordOptions = shuffle(append([]string{entry.Word}, wd...))
			} else {
				gw.WordOptions = []string{entry.Word}
			}
			built[i] = gw
			okFlags[i] = true
		}(i)
	}
	wg.Wait()

	words := make([]GroupWord, 0, len(items))
	for i := range items {
		if okFlags[i] {
			words = append(words, built[i])
		}
	}
	if len(words) == 0 {
		return nil, httpx.ErrNotFound("该组没有可学习的单词")
	}
	return &GroupDetail{Index: index, IsFirstTime: isFirstTime, GroupSize: size, Words: words}, nil
}

// CompleteGroup 标记一组学习/复习完成，推进复习计数与下次复习时间。
func (s *Service) CompleteGroup(ctx context.Context, userID uint64, index int) error {
	plan, err := s.requireActivePlan(ctx, userID)
	if err != nil {
		return err
	}
	progress, err := s.repo.GetGroupProgress(ctx, userID, plan.ID, index)
	if err != nil {
		return err
	}
	now := time.Now()
	if progress == nil {
		progress = &models.WordGroupProgress{UserID: userID, PlanID: plan.ID, GroupIndex: index}
	}
	if progress.FirstStudiedAt == nil {
		progress.FirstStudiedAt = &now
	}
	progress.ReviewCount++
	progress.LastStudiedAt = &now
	next := now.Add(groupReviewInterval(progress.ReviewCount))
	progress.NextReviewAt = &next
	return s.repo.UpsertGroupProgress(ctx, progress)
}

// groupReviewInterval 组复习间隔随复习次数递增。
func groupReviewInterval(reviewCount int) time.Duration {
	switch reviewCount {
	case 1:
		return 24 * time.Hour
	case 2:
		return 2 * 24 * time.Hour
	case 3:
		return 4 * 24 * time.Hour
	case 4:
		return 7 * 24 * time.Hour
	default:
		return 15 * 24 * time.Hour
	}
}

// ===== 处理器 =====

// RegisterGroups 注册按组学习路由（在 Handler.Register 中调用）。
func (h *Handler) RegisterGroups(rg *gin.RouterGroup) {
	rg.GET("/word-learning/groups", h.listGroups)
	rg.GET("/word-learning/groups/:index", h.getGroup)
	rg.POST("/word-learning/groups/:index/complete", h.completeGroup)
}

func (h *Handler) listGroups(c *gin.Context) {
	res, err := h.svc.ListGroups(c.Request.Context(), httpx.MustUserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) getGroup(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		httpx.Fail(c, httpx.ErrValidation("无效的组序号"))
		return
	}
	res, err := h.svc.GetGroup(c.Request.Context(), httpx.MustUserID(c), index)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) completeGroup(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		httpx.Fail(c, httpx.ErrValidation("无效的组序号"))
		return
	}
	if err := h.svc.CompleteGroup(c.Request.Context(), httpx.MustUserID(c), index); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

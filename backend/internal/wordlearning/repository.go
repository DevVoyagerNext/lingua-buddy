package wordlearning

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lingua-buddy/internal/models"
)

// shanghai 用于“今日”自然日边界（与外刊导入一致）。
var shanghai = mustLoadShanghai()

func mustLoadShanghai() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func startOfTodayShanghai(now time.Time) time.Time {
	t := now.In(shanghai)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, shanghai)
}

// Repository 单词学习数据访问。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// ===== 计划 =====

// GetActivePlan 返回用户的 active 计划；无则返回 nil,nil。
func (r *Repository) GetActivePlan(ctx context.Context, userID uint64) (*models.WordLearningPlan, error) {
	var p models.WordLearningPlan
	err := r.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, "active").Take(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePlanWithItems 在一个事务里创建计划与全部计划项。
func (r *Repository) CreatePlanWithItems(ctx context.Context, plan *models.WordLearningPlan, items []models.WordLearningPlanItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].PlanID = plan.ID
		}
		if len(items) > 0 {
			if err := tx.CreateInBatches(items, 500).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListPlans 返回用户全部计划。
func (r *Repository) ListPlans(ctx context.Context, userID uint64) ([]models.WordLearningPlan, error) {
	var plans []models.WordLearningPlan
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&plans).Error
	return plans, err
}

// GetPlan 按 ID 查询计划（限当前用户）。
func (r *Repository) GetPlan(ctx context.Context, userID, id uint64) (*models.WordLearningPlan, error) {
	var p models.WordLearningPlan
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Take(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, err
}

// PlanCounts 计划进度数量。
type PlanCounts struct {
	Total         int64 `json:"total"`
	Waiting       int64 `json:"waiting"`
	Learning      int64 `json:"learning"`
	FirstMastered int64 `json:"first_mastered"`
	Skipped       int64 `json:"skipped"`
}

// CountPlanItems 统计计划队列各状态数量。
func (r *Repository) CountPlanItems(ctx context.Context, planID uint64) (PlanCounts, error) {
	var c PlanCounts
	base := r.db.WithContext(ctx).Model(&models.WordLearningPlanItem{}).Where("plan_id = ?", planID)
	if err := base.Session(&gorm.Session{}).Count(&c.Total).Error; err != nil {
		return c, err
	}
	if err := base.Session(&gorm.Session{}).Where("activated_at IS NULL AND skipped_at IS NULL").Count(&c.Waiting).Error; err != nil {
		return c, err
	}
	if err := base.Session(&gorm.Session{}).Where("activated_at IS NOT NULL AND first_mastered_at IS NULL AND skipped_at IS NULL").Count(&c.Learning).Error; err != nil {
		return c, err
	}
	if err := base.Session(&gorm.Session{}).Where("first_mastered_at IS NOT NULL").Count(&c.FirstMastered).Error; err != nil {
		return c, err
	}
	if err := base.Session(&gorm.Session{}).Where("skipped_at IS NOT NULL").Count(&c.Skipped).Error; err != nil {
		return c, err
	}
	return c, nil
}

// SetPlanStatus 更新计划状态。
func (r *Repository) SetPlanStatus(ctx context.Context, userID, id uint64, status string) error {
	res := r.db.WithContext(ctx).Model(&models.WordLearningPlan{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ===== 用户单词 =====

// FindUserWord 按 (user_id, word) 查询；不存在返回 nil,nil。
func (r *Repository) FindUserWord(ctx context.Context, userID uint64, word string) (*models.UserWord, error) {
	var uw models.UserWord
	err := r.db.WithContext(ctx).Where("user_id = ? AND word = ?", userID, word).Take(&uw).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &uw, nil
}

// GetUserWordByID 按 ID 查询用户单词（限当前用户）。
func (r *Repository) GetUserWordByID(ctx context.Context, userID, id uint64) (*models.UserWord, error) {
	var uw models.UserWord
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Take(&uw).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	return &uw, err
}

// CreateUserWord 新增用户单词。
func (r *Repository) CreateUserWord(ctx context.Context, uw *models.UserWord) error {
	return r.db.WithContext(ctx).Create(uw).Error
}

// DeleteUserWord 移出生词本（限当前用户）。
func (r *Repository) DeleteUserWord(ctx context.Context, userID, id uint64) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.UserWord{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListUserWords 分页查询用户单词，可按阶段过滤、关键词搜索。
func (r *Repository) ListUserWords(ctx context.Context, userID uint64, stage, keyword string, page, size int) ([]models.UserWord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.UserWord{}).Where("user_id = ?", userID)
	if stage != "" {
		tx = tx.Where("learning_stage = ?", stage)
	}
	if keyword != "" {
		tx = tx.Where("word LIKE ?", keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.UserWord
	err := tx.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// FindDueWords 返回到期的活跃单词（排除“仅被跳过计划项引用”的词），按出题优先级排序。
func (r *Repository) FindDueWords(ctx context.Context, userID uint64, now time.Time, limit int) ([]models.UserWord, error) {
	var items []models.UserWord
	stageOrder := "CASE learning_stage WHEN 'recognition' THEN 0 WHEN 'discrimination' THEN 1 WHEN 'spelling' THEN 2 ELSE 3 END"
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND next_review_at <= ?", userID, now).
		Where(`NOT (EXISTS (SELECT 1 FROM word_learning_plan_items i WHERE i.user_word_id = user_words.id AND i.skipped_at IS NOT NULL)
			AND NOT EXISTS (SELECT 1 FROM word_learning_plan_items i2 WHERE i2.user_word_id = user_words.id AND i2.skipped_at IS NULL))`).
		Order("next_review_at ASC").
		Order(stageOrder).
		Order("last_trained_at ASC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

// CountDueWords 统计当前到期单词数（排除仅被跳过引用的词）。
func (r *Repository) CountDueWords(ctx context.Context, userID uint64, now time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.UserWord{}).
		Where("user_id = ? AND next_review_at <= ?", userID, now).
		Where(`NOT (EXISTS (SELECT 1 FROM word_learning_plan_items i WHERE i.user_word_id = user_words.id AND i.skipped_at IS NOT NULL)
			AND NOT EXISTS (SELECT 1 FROM word_learning_plan_items i2 WHERE i2.user_word_id = user_words.id AND i2.skipped_at IS NULL))`).
		Count(&n).Error
	return n, err
}

// CountUserWords 统计用户单词总数（判断是否“完全无词可学”）。
func (r *Repository) CountUserWords(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.UserWord{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// ===== 激活新词（加锁事务）=====

// ActivateResult 激活后选中的用户单词（可能为空）。
type ActivateResult struct {
	Selected *models.UserWord
}

// ActivateAndPickDue 在对计划加锁的事务中按名额激活新词，并返回新激活词中的一道（最小 queue_position）。
func (r *Repository) ActivateAndPickDue(ctx context.Context, userID uint64, plan *models.WordLearningPlan, now time.Time) (*models.UserWord, error) {
	var picked *models.UserWord
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 对计划行加锁，串行结算名额。
		var locked models.WordLearningPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", plan.ID).Take(&locked).Error; err != nil {
			return err
		}

		todayStart := startOfTodayShanghai(now)
		var todayNew, activeCount int64
		if err := tx.Model(&models.WordLearningPlanItem{}).
			Where("plan_id = ? AND activated_at >= ? AND first_mastered_at IS NULL", locked.ID, todayStart).
			Count(&todayNew).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.WordLearningPlanItem{}).
			Where("plan_id = ? AND activated_at IS NOT NULL AND first_mastered_at IS NULL AND skipped_at IS NULL", locked.ID).
			Count(&activeCount).Error; err != nil {
			return err
		}

		slots := min64(int64(locked.DailyNewWordLimit)-todayNew, int64(locked.ActiveWordLimit)-activeCount)
		if slots <= 0 {
			return nil
		}

		// 扫描等待队列（多取一些，已掌握词回填不占名额）。
		var candidates []models.WordLearningPlanItem
		if err := tx.Where("plan_id = ? AND activated_at IS NULL AND skipped_at IS NULL", locked.ID).
			Order("queue_position ASC").
			Limit(int(slots)*3 + 20).
			Find(&candidates).Error; err != nil {
			return err
		}

		consumed := int64(0)
		for i := range candidates {
			if consumed >= slots {
				break
			}
			item := &candidates[i]
			uw, err := findUserWordTx(tx, userID, item.Word)
			if err != nil {
				return err
			}
			nowT := now
			if uw == nil {
				uw = &models.UserWord{
					UserID:         userID,
					ECDICTEntryID:  &item.ECDICTEntryID,
					Word:           item.Word,
					LearningStage:  StageRecognition,
					NextReviewAt:   nowT,
					StageChangedAt: nowT,
				}
				if err := tx.Create(uw).Error; err != nil {
					return err
				}
				item.UserWordID = &uw.ID
				item.ActivatedAt = &nowT
				consumed++
			} else if uw.LearningStage == StageMastered {
				// 别处已掌握：回填，不占名额。
				item.UserWordID = &uw.ID
				item.ActivatedAt = &nowT
				item.FirstMasteredAt = uw.FirstMasteredAt
				if item.FirstMasteredAt == nil {
					item.FirstMasteredAt = &nowT
				}
			} else {
				item.UserWordID = &uw.ID
				item.ActivatedAt = &nowT
				consumed++
			}
			if err := tx.Model(&models.WordLearningPlanItem{}).Where("id = ?", item.ID).
				Updates(map[string]any{
					"user_word_id":      item.UserWordID,
					"activated_at":      item.ActivatedAt,
					"first_mastered_at": item.FirstMasteredAt,
				}).Error; err != nil {
				return err
			}
			if picked == nil && item.FirstMasteredAt == nil {
				picked = uw
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return picked, nil
}

func findUserWordTx(tx *gorm.DB, userID uint64, word string) (*models.UserWord, error) {
	var uw models.UserWord
	err := tx.Where("user_id = ? AND word = ?", userID, word).Take(&uw).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &uw, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

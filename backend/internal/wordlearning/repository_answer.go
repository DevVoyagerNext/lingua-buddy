package wordlearning

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lingua-buddy/internal/models"
)

// ErrStageChanged 题目令牌阶段与数据库当前阶段不一致。
var ErrStageChanged = errors.New("stage changed")

// SubmitInput 提交答案事务输入。
type SubmitInput struct {
	Record              *models.TrainingAnswerRecord
	UserWord            *models.UserWord // 已由 StagePolicy 计算好的新状态
	ExpectedStage       string           // 令牌中的作答前阶段
	PlanID              *uint64
	BecameFirstMastered bool
	Now                 time.Time
}

// SubmitAnswer 在单事务内完成幂等校验、阶段校验、记录写入、单词更新与计划进度更新。
// 返回记录与是否为重复提交（幂等命中）。
func (r *Repository) SubmitAnswer(ctx context.Context, in SubmitInput) (*models.TrainingAnswerRecord, bool, error) {
	var (
		result    *models.TrainingAnswerRecord
		duplicate bool
	)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 幂等：同 submission_id 已存在则直接返回原记录。
		var existing models.TrainingAnswerRecord
		err := tx.Where("user_id = ? AND submission_id = ?", in.Record.UserID, in.Record.SubmissionID).
			Take(&existing).Error
		if err == nil {
			result = &existing
			duplicate = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 2. 加锁读取用户单词，校验阶段是否仍与令牌一致。
		var locked models.UserWord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", in.UserWord.ID).Take(&locked).Error; err != nil {
			return err
		}
		if locked.LearningStage != in.ExpectedStage {
			return ErrStageChanged
		}

		// 3. 写入答题记录。
		if err := tx.Create(in.Record).Error; err != nil {
			return err
		}

		// 4. 更新用户单词。
		if err := tx.Model(&models.UserWord{}).Where("id = ?", in.UserWord.ID).
			Updates(map[string]any{
				"learning_stage":       in.UserWord.LearningStage,
				"stage_correct_streak": in.UserWord.StageCorrectStreak,
				"next_review_at":       in.UserWord.NextReviewAt,
				"last_trained_at":      in.UserWord.LastTrainedAt,
				"stage_changed_at":     in.UserWord.StageChangedAt,
				"first_mastered_at":    in.UserWord.FirstMasteredAt,
				"total_correct_count":  in.UserWord.TotalCorrectCount,
				"total_wrong_count":    in.UserWord.TotalWrongCount,
				"last_answer_correct":  in.UserWord.LastAnswerCorrect,
			}).Error; err != nil {
			return err
		}

		// 5. 首次掌握：回填关联计划项，并判断计划是否完成。
		if in.BecameFirstMastered {
			if err := tx.Model(&models.WordLearningPlanItem{}).
				Where("user_word_id = ? AND first_mastered_at IS NULL", in.UserWord.ID).
				Update("first_mastered_at", in.Now).Error; err != nil {
				return err
			}
			if in.PlanID != nil {
				var remaining int64
				if err := tx.Model(&models.WordLearningPlanItem{}).
					Where("plan_id = ? AND first_mastered_at IS NULL AND skipped_at IS NULL", *in.PlanID).
					Count(&remaining).Error; err != nil {
					return err
				}
				if remaining == 0 {
					if err := tx.Model(&models.WordLearningPlan{}).
						Where("id = ? AND completed_at IS NULL", *in.PlanID).
						Update("completed_at", in.Now).Error; err != nil {
						return err
					}
				}
			}
		}

		result = in.Record
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return result, duplicate, nil
}

// SkipPlanItem 跳过计划项（限当前用户 active 计划），写入 skipped_at。
func (r *Repository) SkipPlanItem(ctx context.Context, userID, itemID uint64, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.WordLearningPlanItem
		if err := tx.Where("id = ?", itemID).Take(&item).Error; err != nil {
			return err
		}
		var plan models.WordLearningPlan
		if err := tx.Where("id = ? AND user_id = ?", item.PlanID, userID).Take(&plan).Error; err != nil {
			return err
		}
		if item.SkippedAt != nil {
			return nil
		}
		return tx.Model(&models.WordLearningPlanItem{}).Where("id = ?", itemID).
			Update("skipped_at", now).Error
	})
}

// PlanWordRow 计划词条及其当前阶段。
type PlanWordRow struct {
	ItemID        uint64     `json:"item_id"`
	Word          string     `json:"word"`
	QueuePosition int        `json:"queue_position"`
	LearningStage *string    `json:"learning_stage"`
	ActivatedAt   *time.Time `json:"activated_at"`
	SkippedAt     *time.Time `json:"skipped_at"`
}

// ListPlanWords 分页查询计划词条及当前阶段。
func (r *Repository) ListPlanWords(ctx context.Context, planID uint64, page, size int) ([]PlanWordRow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.WordLearningPlanItem{}).
		Where("plan_id = ?", planID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PlanWordRow
	err := r.db.WithContext(ctx).
		Table("word_learning_plan_items AS i").
		Select("i.id AS item_id, i.word, i.queue_position, uw.learning_stage, i.activated_at, i.skipped_at").
		Joins("LEFT JOIN user_words uw ON uw.id = i.user_word_id").
		Where("i.plan_id = ?", planID).
		Order("i.queue_position ASC").
		Offset((page - 1) * size).Limit(size).
		Scan(&rows).Error
	return rows, total, err
}

// ListWrongWordAnswers 分页查询历史单词错误答案。
func (r *Repository) ListWrongWordAnswers(ctx context.Context, userID uint64, page, size int) ([]models.TrainingAnswerRecord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.TrainingAnswerRecord{}).
		Where("user_id = ? AND training_type = ? AND is_correct = ?", userID, "word", false)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.TrainingAnswerRecord
	err := tx.Order("submitted_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

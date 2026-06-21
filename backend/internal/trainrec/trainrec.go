// Package trainrec 提供专项训练答题记录的共享持久化，供作文与训练模块复用。
package trainrec

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"lingua-buddy/internal/models"
)

// EvaluationPendingTimeout pending 软超时阈值。
const EvaluationPendingTimeout = 60 * time.Second

// Repository 答题记录持久化。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造。
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// CreateOrGet 按 (user_id, submission_id) 幂等创建；已存在则返回原记录与 true。
func (r *Repository) CreateOrGet(ctx context.Context, rec *models.TrainingAnswerRecord) (*models.TrainingAnswerRecord, bool, error) {
	var existing models.TrainingAnswerRecord
	err := r.db.WithContext(ctx).Where("user_id = ? AND submission_id = ?", rec.UserID, rec.SubmissionID).Take(&existing).Error
	if err == nil {
		return &existing, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	if rec.SubmittedAt.IsZero() {
		rec.SubmittedAt = time.Now()
	}
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		// 并发下唯一索引冲突：返回已存在记录。
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if e2 := r.db.WithContext(ctx).Where("user_id = ? AND submission_id = ?", rec.UserID, rec.SubmissionID).Take(&existing).Error; e2 == nil {
				return &existing, true, nil
			}
		}
		return nil, false, err
	}
	return rec, false, nil
}

// GetByID 查询答题记录（限当前用户）。
func (r *Repository) GetByID(ctx context.Context, userID, id uint64) (*models.TrainingAnswerRecord, error) {
	var rec models.TrainingAnswerRecord
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	return &rec, err
}

// EvaluationUpdate 评价更新字段。
type EvaluationUpdate struct {
	ReferenceAnswer  *string
	EvaluationResult []byte
	Status           string
	HistoryRecordID  *uint64
}

// UpdateEvaluation 仅更新评价相关字段，不改写用户答案。
func (r *Repository) UpdateEvaluation(ctx context.Context, id uint64, up EvaluationUpdate) error {
	now := time.Now()
	fields := map[string]any{
		"evaluation_status": up.Status,
		"evaluated_at":      now,
		"updated_at":        now,
	}
	if up.ReferenceAnswer != nil {
		fields["reference_answer"] = *up.ReferenceAnswer
	}
	if up.EvaluationResult != nil {
		fields["evaluation_result"] = up.EvaluationResult
	}
	if up.HistoryRecordID != nil {
		fields["history_record_id"] = *up.HistoryRecordID
	}
	return r.db.WithContext(ctx).Model(&models.TrainingAnswerRecord{}).Where("id = ?", id).Updates(fields).Error
}

// List 分页查询答题记录，可按训练类型与题目键过滤。
func (r *Repository) List(ctx context.Context, userID uint64, trainingType, questionKey string, page, size int) ([]models.TrainingAnswerRecord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.TrainingAnswerRecord{}).Where("user_id = ?", userID)
	if trainingType != "" {
		tx = tx.Where("training_type = ?", trainingType)
	}
	if questionKey != "" {
		tx = tx.Where("question_key = ?", questionKey)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.TrainingAnswerRecord
	err := tx.Order("submitted_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// EffectiveStatus 计算评价状态：pending 且超过软超时按 timeout 处理。
func EffectiveStatus(rec *models.TrainingAnswerRecord) string {
	if rec.EvaluationStatus == "pending" && time.Since(rec.SubmittedAt) > EvaluationPendingTimeout {
		return "timeout"
	}
	return rec.EvaluationStatus
}

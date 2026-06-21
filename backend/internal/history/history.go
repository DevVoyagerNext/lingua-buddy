// Package history 提供统一历史记录的写入、查询与删除。
package history

import (
	"context"
	"time"

	"gorm.io/gorm"

	"lingua-buddy/internal/models"
)

// 历史类型。
const (
	TypeTranslation     = "translation"
	TypeSpeech          = "speech"
	TypeGrammarAnalysis = "grammar_analysis"
	TypeCorrection      = "correction"
	TypeEssay           = "essay"
)

// Repository 历史持久化。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造历史仓库。
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create 写入一条历史记录并回填 ID。
func (r *Repository) Create(ctx context.Context, rec *models.HistoryRecord) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(rec).Error
}

// List 分页查询历史，可按类型过滤。
func (r *Repository) List(ctx context.Context, userID uint64, recordType string, page, size int) ([]models.HistoryRecord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.HistoryRecord{}).Where("user_id = ?", userID)
	if recordType != "" {
		tx = tx.Where("record_type = ?", recordType)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.HistoryRecord
	err := tx.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// Delete 删除历史；speech 记录连带删除关联音频。
func (r *Repository) Delete(ctx context.Context, userID, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rec models.HistoryRecord
		if err := tx.Where("id = ? AND user_id = ?", id, userID).Take(&rec).Error; err != nil {
			return err
		}
		if rec.AudioFileID != nil {
			if err := tx.Where("id = ? AND user_id = ?", *rec.AudioFileID, userID).
				Delete(&models.AudioFile{}).Error; err != nil {
				return err
			}
			// 注：OSS 对象删除由存储层异步清理（首版记录待清理日志）。
		}
		return tx.Delete(&rec).Error
	})
}

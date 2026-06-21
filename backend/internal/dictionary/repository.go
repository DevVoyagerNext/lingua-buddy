// Package dictionary 提供查词与查词历史接口。
package dictionary

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lingua-buddy/internal/models"
)

// HistoryRepository 管理查词历史。
type HistoryRepository struct {
	db *gorm.DB
}

// NewHistoryRepository 构造查词历史仓库。
func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// Record 记录一次查词：同 (user_id, word) 累加次数并更新时间。
func (r *HistoryRepository) Record(ctx context.Context, userID uint64, word string) error {
	rec := models.DictionaryQueryRecord{
		UserID:        userID,
		Word:          word,
		QueryCount:    1,
		LastQueriedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "word"}},
		DoUpdates: clause.Assignments(map[string]any{
			"query_count":     gorm.Expr("query_count + 1"),
			"last_queried_at": rec.LastQueriedAt,
		}),
	}).Create(&rec).Error
}

// List 分页返回查词历史，按最近查询时间倒序。
func (r *HistoryRepository) List(ctx context.Context, userID uint64, page, size int) ([]models.DictionaryQueryRecord, int64, error) {
	var (
		items []models.DictionaryQueryRecord
		total int64
	)
	tx := r.db.WithContext(ctx).Model(&models.DictionaryQueryRecord{}).Where("user_id = ?", userID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := tx.Order("last_queried_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&items).Error
	return items, total, err
}

// Delete 删除一条查词历史（限当前用户）。
func (r *HistoryRepository) Delete(ctx context.Context, userID, id uint64) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.DictionaryQueryRecord{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

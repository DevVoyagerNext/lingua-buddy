package dictionary

import (
	"context"
	"log"
	"strings"

	"lingua-buddy/internal/lexicon"
	"lingua-buddy/internal/models"
)

// Service 查词服务。
type Service struct {
	lex     *lexicon.Repository
	history *HistoryRepository
}

// NewService 构造查词服务。
func NewService(lex *lexicon.Repository, history *HistoryRepository) *Service {
	return &Service{lex: lex, history: history}
}

// Lookup 精确查词；命中后记录登录用户的查词历史（失败不阻断查询）。
func (s *Service) Lookup(ctx context.Context, userID uint64, word string) (*lexicon.Entry, error) {
	entry, err := s.lex.FindExact(ctx, word)
	if err != nil {
		return nil, err
	}
	if userID != 0 {
		if e := s.history.Record(ctx, userID, strings.TrimSpace(word)); e != nil {
			log.Printf("记录查词历史失败 user=%d word=%q: %v", userID, word, e)
		}
	}
	return entry, nil
}

// Suggest 前缀联想。
func (s *Service) Suggest(ctx context.Context, prefix string, limit int) ([]lexicon.Suggestion, error) {
	return s.lex.Suggest(ctx, prefix, limit)
}

// Similar 相近拼写建议。
func (s *Service) Similar(ctx context.Context, word string, limit int) ([]lexicon.Suggestion, error) {
	return s.lex.SuggestSimilar(ctx, word, limit)
}

// History 分页查询查词历史。
func (s *Service) History(ctx context.Context, userID uint64, page, size int) ([]models.DictionaryQueryRecord, int64, error) {
	return s.history.List(ctx, userID, page, size)
}

// DeleteHistory 删除一条查词历史。
func (s *Service) DeleteHistory(ctx context.Context, userID, id uint64) error {
	return s.history.Delete(ctx, userID, id)
}

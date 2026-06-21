package dictionary

import (
	"context"
	"log"
	"strings"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/lexicon"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/user"
)

// Service 查词服务。
type Service struct {
	lex     *lexicon.Repository
	history *HistoryRepository
	ai      ai.Provider
	level   user.LevelLookup
}

// NewService 构造查词服务。
func NewService(lex *lexicon.Repository, history *HistoryRepository, provider ai.Provider, level user.LevelLookup) *Service {
	return &Service{lex: lex, history: history, ai: provider, level: level}
}

// Examples 生成 AI 例句（DICT-03）。AI 失败不影响基础查词。
func (s *Service) Examples(ctx context.Context, userID uint64, word, topic, difficulty string) ([]ai.Example, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, lexicon.ErrNotFound
	}
	out, err := s.ai.GenerateExamples(ctx, ai.ExampleInput{
		Word: word, Topic: topic, Difficulty: difficulty, Level: s.level.Level(ctx, userID), Count: 3,
	})
	if err != nil {
		status, code, msg := ai.ErrorCode(err)
		return nil, httpx.NewError(status, code, msg)
	}
	return out, nil
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

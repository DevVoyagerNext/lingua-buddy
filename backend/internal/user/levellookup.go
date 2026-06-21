package user

import "context"

// LevelLookup 提供按用户取有效英语水平的能力，供 AI 模块个性化提示词。
type LevelLookup interface {
	Level(ctx context.Context, userID uint64) string
}

type levelLookup struct {
	repo *Repository
}

// NewLevelLookup 构造按用户查英语水平的实现（带默认兜底）。
func NewLevelLookup(repo *Repository) LevelLookup { return levelLookup{repo: repo} }

func (l levelLookup) Level(ctx context.Context, userID uint64) string {
	u, err := l.repo.FindByID(ctx, userID)
	if err != nil {
		return DefaultEnglishLevel
	}
	return EffectiveLevel(u.EnglishLevel)
}

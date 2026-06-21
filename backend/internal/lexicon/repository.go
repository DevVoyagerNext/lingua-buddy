package lexicon

import (
	"context"
	"errors"
	"sort"
	"strings"

	"gorm.io/gorm"

	"lingua-buddy/internal/models"
)

// ErrNotFound 词条不存在。
var ErrNotFound = errors.New("dictionary entry not found")

// Suggestion 联想/纠错建议项。
type Suggestion struct {
	Word  string `json:"word"`
	Gloss string `json:"gloss"` // 简短中文释义
}

// Repository 只读查询 ecdict。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造词典仓库。
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// FindExact 精确查词（大小写由 ecdict 的 ai_ci 排序规则不敏感处理，仅去首尾空格）。
func (r *Repository) FindExact(ctx context.Context, word string) (*Entry, error) {
	normalized := strings.TrimSpace(word)
	if normalized == "" {
		return nil, ErrNotFound
	}
	var m models.ECDICTEntry
	err := r.db.WithContext(ctx).Where("word = ?", normalized).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return FromModel(&m), nil
}

// GetByID 按词典 ID 查询。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*Entry, error) {
	var m models.ECDICTEntry
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return FromModel(&m), nil
}

// Suggest 前缀联想，按常用程度（frq 升序，NULL 最后）排序。
func (r *Repository) Suggest(ctx context.Context, prefix string, limit int) ([]Suggestion, error) {
	normalized := strings.ToLower(strings.TrimSpace(prefix))
	if normalized == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows []models.ECDICTEntry
	err := r.db.WithContext(ctx).
		Select("word", "translation", "frq").
		Where("word LIKE ?", normalized+"%").
		// frq 为 NULL 或 0 表示“无词频数据”，排最后；其余按 frq 升序（越小越常用）。
		Order("(frq IS NULL OR frq = 0)").
		Order("frq").
		Order("word").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Suggestion, 0, len(rows))
	for _, m := range rows {
		out = append(out, Suggestion{Word: m.Word, Gloss: glossOf(m.Translation)})
	}
	return out, nil
}

// glossOf 从 translation 列生成简短中文释义。
func glossOf(translation *string) string {
	if translation == nil {
		return ""
	}
	return CanonicalGloss(*translation)
}

// SuggestSimilar 在精确查词未命中时给出相近拼写建议（DICT-01）。
// 仅用 MySQL：取前缀候选后在内存计算 Levenshtein 距离过滤排序。
func (r *Repository) SuggestSimilar(ctx context.Context, word string, limit int) ([]Suggestion, error) {
	w := strings.ToLower(strings.TrimSpace(word))
	if len(w) < 2 {
		return nil, nil
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	prefixLen := 2
	if len(w) < prefixLen {
		prefixLen = len(w)
	}
	prefix := w[:prefixLen]
	wl := len([]rune(w))

	type cand struct {
		word  string
		gloss string
		frq   int
	}
	var rows []models.ECDICTEntry
	// 前缀 + 长度窗口大幅缩小候选；按词频优先保留常用词，避免 LIMIT 把目标词截断。
	err := r.db.WithContext(ctx).
		Select("word", "translation", "frq").
		Where("word LIKE ?", prefix+"%").
		Where("CHAR_LENGTH(word) BETWEEN ? AND ?", wl-2, wl+2).
		Order("(frq IS NULL OR frq = 0)").
		Order("frq").
		Limit(1500).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	var cands []cand
	for _, m := range rows {
		cw := strings.ToLower(m.Word)
		if cw == w || IsPhrase(m.Word) {
			continue
		}
		if Levenshtein(w, cw) > 2 {
			continue
		}
		frq := 1 << 30
		if m.FRQ != nil && *m.FRQ > 0 {
			frq = *m.FRQ
		}
		cands = append(cands, cand{word: m.Word, gloss: glossOf(m.Translation), frq: frq})
	}
	sort.Slice(cands, func(i, j int) bool {
		di := Levenshtein(w, strings.ToLower(cands[i].word))
		dj := Levenshtein(w, strings.ToLower(cands[j].word))
		if di != dj {
			return di < dj
		}
		return cands[i].frq < cands[j].frq
	})
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]Suggestion, 0, len(cands))
	for _, c := range cands {
		out = append(out, Suggestion{Word: c.word, Gloss: c.gloss})
	}
	return out, nil
}

// ListByExamTag 返回带指定考试标签码的单词（排除词组），用于创建词汇计划。
// tag 按词边界匹配，无索引、全表扫描，仅供低频的建计划调用。
func (r *Repository) ListByExamTag(ctx context.Context, tagCode string) ([]Entry, error) {
	var rows []models.ECDICTEntry
	pattern := "% " + tagCode + " %"
	err := r.db.WithContext(ctx).
		Select("id", "word", "frq", "bnc").
		Where("CONCAT(' ', tag, ' ') LIKE ?", pattern).
		Where("word NOT LIKE ?", "% %").
		Where("word NOT LIKE ?", "%-%").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for i := range rows {
		out = append(out, *FromModel(&rows[i]))
	}
	return out, nil
}

// FindSpellingNeighbors 取拼写相近（同首字母 + 长度相近）且有释义的候选词，
// 供“中文选英文”题挑选拼写迷惑性强的干扰项。
func (r *Repository) FindSpellingNeighbors(ctx context.Context, target *Entry, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 300
	}
	w := strings.ToLower(target.Word)
	if w == "" {
		return nil, nil
	}
	wl := len([]rune(w))
	prefix := w[:1]
	var rows []models.ECDICTEntry
	err := r.db.WithContext(ctx).
		Select("id", "word", "translation", "tag", "frq", "bnc").
		Where("word LIKE ?", prefix+"%").
		Where("CHAR_LENGTH(word) BETWEEN ? AND ?", wl-2, wl+2).
		Where("word <> ?", target.Word).
		Where("word NOT LIKE ?", "% %").
		Where("word NOT LIKE ?", "%-%").
		Where("translation IS NOT NULL AND translation <> ''").
		Order("(frq IS NULL OR frq = 0)").
		Order("frq").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for i := range rows {
		out = append(out, *FromModel(&rows[i]))
	}
	return out, nil
}

// FindFreqNeighbors 取目标词频率邻域内的候选词（含释义），供干扰项模块内存打分。
// 走有索引的 frq；目标无 frq 时退化为按 word 前缀邻域取词。
func (r *Repository) FindFreqNeighbors(ctx context.Context, target *Entry, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 200
	}
	q := r.db.WithContext(ctx).
		Select("id", "word", "translation", "tag", "frq", "bnc").
		Where("word <> ?", target.Word).
		Where("word NOT LIKE ?", "% %").
		Where("word NOT LIKE ?", "%-%").
		Where("translation IS NOT NULL AND translation <> ''")

	if target.FrequencyRank != nil {
		center := *target.FrequencyRank
		span := 4000
		q = q.Where("frq BETWEEN ? AND ?", center-span, center+span).Order("frq ASC")
	} else if len(target.Word) >= 2 {
		q = q.Where("word LIKE ?", strings.ToLower(target.Word[:2])+"%")
	}

	var rows []models.ECDICTEntry
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for i := range rows {
		out = append(out, *FromModel(&rows[i]))
	}
	return out, nil
}

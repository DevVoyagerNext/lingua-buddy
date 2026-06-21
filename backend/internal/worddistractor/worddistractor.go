// Package worddistractor 为单词选择题生成安全、合理的干扰项。
// 它只读 ecdict，不读用户状态、不判分、不出题、不调用 AI。
package worddistractor

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"lingua-buddy/internal/lexicon"
)

// isCleanGloss 过滤掉 ecdict 中的脏释义：非法 UTF-8、交叉引用（含 [ ] =）、
// 不含中文或过长的条目，避免它们成为选择题选项。
func isCleanGloss(g string) bool {
	if g == "" || !utf8.ValidString(g) {
		return false
	}
	if strings.ContainsAny(g, "[]=<>") {
		return false
	}
	runes := []rune(g)
	if len(runes) > 20 {
		return false
	}
	hasHan := false
	for _, r := range runes {
		if r == unicode.ReplacementChar {
			return false
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	return hasHan
}

// ErrInsufficient 无法找到足够的安全干扰项。
var ErrInsufficient = errors.New("insufficient distractors")

// GeneratorVersion 当前算法版本，写入答题记录。
const GeneratorVersion = "word-v1"

// Meta 生成元信息。
type Meta struct {
	Version string `json:"version"`
	Layer   string `json:"layer"` // strict/balanced/fallback
}

// Service 干扰项服务。
type Service struct {
	lex *lexicon.Repository
}

// NewService 构造干扰项服务。
func NewService(lex *lexicon.Repository) *Service { return &Service{lex: lex} }

// candidate 内部打分候选。
type candidate struct {
	entry lexicon.Entry
	gloss string
	score int
}

// targetForms 收集目标词需要排除的拼写（自身 + 词形变化），小写。
func targetForms(target *lexicon.Entry) map[string]bool {
	forms := map[string]bool{strings.ToLower(target.Word): true}
	for _, f := range target.WordForms {
		forms[strings.ToLower(f.Word)] = true
	}
	if target.Lemma != "" {
		forms[strings.ToLower(target.Lemma)] = true
	}
	return forms
}

func sameExamTag(a, b *lexicon.Entry) bool {
	set := map[string]bool{}
	for _, t := range a.RawTags {
		set[t] = true
	}
	for _, t := range b.RawTags {
		if set[t] {
			return true
		}
	}
	return false
}

func freqClose(a, b *lexicon.Entry) bool {
	if a.FrequencyRank == nil || b.FrequencyRank == nil {
		return false
	}
	d := *a.FrequencyRank - *b.FrequencyRank
	if d < 0 {
		d = -d
	}
	return d <= 3000
}

// FindMeaningDistractors 为“英文选中文”生成三个中文释义干扰项（与正确释义不同、彼此不同）。
func (s *Service) FindMeaningDistractors(ctx context.Context, target *lexicon.Entry, count int) ([]string, Meta, error) {
	correctGloss := target.CanonicalGlossOf()
	pool, err := s.lex.FindFreqNeighbors(ctx, target, 300)
	if err != nil {
		return nil, Meta{}, err
	}
	forms := targetForms(target)

	seenGloss := map[string]bool{correctGloss: true}
	var cands []candidate
	for i := range pool {
		e := pool[i]
		if forms[strings.ToLower(e.Word)] {
			continue
		}
		gloss := e.CanonicalGlossOf()
		if !isCleanGloss(gloss) || seenGloss[gloss] {
			continue
		}
		score := 0
		if sameExamTag(target, &e) {
			score += 10
		}
		if freqClose(target, &e) {
			score += 10
		}
		cands = append(cands, candidate{entry: e, gloss: gloss, score: score})
	}

	picked := pickDistinctGloss(cands, count, seenGloss)
	if len(picked) < count {
		return nil, Meta{}, ErrInsufficient
	}
	layer := "strict"
	return picked, Meta{Version: GeneratorVersion, Layer: layer}, nil
}

// FindWordDistractors 为“中文选英文”生成三个英文单词干扰项（拼写相近、释义不同）。
func (s *Service) FindWordDistractors(ctx context.Context, target *lexicon.Entry, count int) ([]string, Meta, error) {
	correctGloss := target.CanonicalGlossOf()
	forms := targetForms(target)

	// 主候选池：拼写邻域；不足时再并入频率邻域作为 fallback。
	spelling, err := s.lex.FindSpellingNeighbors(ctx, target, 400)
	if err != nil {
		return nil, Meta{}, err
	}
	layer := "strict"
	pool := spelling
	if countUsableWords(pool, forms, correctGloss) < count {
		freq, err := s.lex.FindFreqNeighbors(ctx, target, 300)
		if err != nil {
			return nil, Meta{}, err
		}
		pool = append(pool, freq...)
		layer = "balanced"
	}

	tw := strings.ToLower(target.Word)
	seenWord := map[string]bool{}
	seenGloss := map[string]bool{correctGloss: true}
	var cands []candidate
	for i := range pool {
		e := pool[i]
		lw := strings.ToLower(e.Word)
		if forms[lw] || seenWord[lw] {
			continue
		}
		gloss := e.CanonicalGlossOf()
		if !isCleanGloss(gloss) || seenGloss[gloss] {
			continue
		}
		seenWord[lw] = true
		dist := lexicon.Levenshtein(tw, lw)
		score := 0
		switch {
		case dist >= 1 && dist <= 3:
			score += 35
		case dist <= 5:
			score += 15
		}
		ld := len([]rune(tw)) - len([]rune(lw))
		if ld < 0 {
			ld = -ld
		}
		if ld <= 2 {
			score += 15
		}
		if strings.HasPrefix(lw, tw[:1]) {
			score += 15
		}
		if sameExamTag(target, &e) {
			score += 10
		}
		if freqClose(target, &e) {
			score += 10
		}
		cands = append(cands, candidate{entry: e, gloss: gloss, score: score})
	}

	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	var picked []string
	pickedGloss := map[string]bool{correctGloss: true}
	for _, c := range cands {
		if pickedGloss[c.gloss] {
			continue
		}
		picked = append(picked, c.entry.Word)
		pickedGloss[c.gloss] = true
		if len(picked) == count {
			break
		}
	}
	if len(picked) < count {
		return nil, Meta{}, ErrInsufficient
	}
	return picked, Meta{Version: GeneratorVersion, Layer: layer}, nil
}

func pickDistinctGloss(cands []candidate, count int, seen map[string]bool) []string {
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	picked := make([]string, 0, count)
	taken := map[string]bool{}
	for k := range seen {
		taken[k] = true
	}
	for _, c := range cands {
		if taken[c.gloss] {
			continue
		}
		picked = append(picked, c.gloss)
		taken[c.gloss] = true
		if len(picked) == count {
			break
		}
	}
	return picked
}

func countUsableWords(pool []lexicon.Entry, forms map[string]bool, correctGloss string) int {
	seen := map[string]bool{}
	n := 0
	for i := range pool {
		lw := strings.ToLower(pool[i].Word)
		if forms[lw] || seen[lw] {
			continue
		}
		g := pool[i].CanonicalGlossOf()
		if !isCleanGloss(g) || g == correctGloss {
			continue
		}
		seen[lw] = true
		n++
	}
	return n
}

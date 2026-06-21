// Package lexicon 封装只读词典 ecdict 的领域模型、解析与查询，
// 供词典、干扰项与单词学习模块共用。
package lexicon

import (
	"regexp"
	"strings"

	"lingua-buddy/internal/models"
)

// WordForm 词形变化。
type WordForm struct {
	Type string `json:"type"` // 过去式/复数 等中文标签
	Word string `json:"word"`
}

// Entry 词典领域模型（已解析）。
type Entry struct {
	ID            uint64     `json:"id"`
	Word          string     `json:"word"`
	Phonetic      string     `json:"phonetic"`
	Definitions   []string   `json:"definitions"`
	Translations  []string   `json:"translations"`
	CollinsStars  *int       `json:"collins_stars"`
	OxfordCore    bool       `json:"oxford_core"`
	Tags          []string   `json:"tags"`     // 可读考试标签
	RawTags       []string   `json:"raw_tags"` // 原始标签码（cet4 等）
	BNCRank       *int       `json:"bnc_rank"`
	FrequencyRank *int       `json:"frequency_rank"`
	WordForms     []WordForm `json:"word_forms"`
	Lemma         string     `json:"lemma"` // 若本词条是变形，指向原形（DICT-01）
}

var (
	// 行内多义分隔（字面量反斜杠 n）。
	lineSep = `\n`
	// 词性前缀，如 "vt. " "adj. "。
	posPrefixRe = regexp.MustCompile(`^([a-zA-Z]{1,5}\.\s*)+`)
	// 中文义项分隔。
	senseSplitRe = regexp.MustCompile(`[；;，,]`)
)

// 考试标签码 -> 可读名称。
var examTagLabels = map[string]string{
	"zk":    "中考",
	"gk":    "高考",
	"cet4":  "四级",
	"cet6":  "六级",
	"ky":    "考研",
	"toefl": "托福",
	"ielts": "雅思",
	"gre":   "GRE",
}

// 词形变化类型码 -> 中文标签。
var exchangeTypeLabels = map[string]string{
	"p": "过去式",
	"d": "过去分词",
	"i": "现在分词",
	"3": "第三人称单数",
	"r": "比较级",
	"t": "最高级",
	"s": "复数",
}

func splitLines(s string) []string {
	parts := strings.Split(s, lineSep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// FromModel 把只读模型解析为领域 Entry。
func FromModel(m *models.ECDICTEntry) *Entry {
	e := &Entry{ID: m.ID, Word: m.Word}
	if m.Phonetic != nil {
		e.Phonetic = strings.TrimSpace(*m.Phonetic)
	}
	if m.Definition != nil {
		e.Definitions = splitLines(*m.Definition)
	}
	if m.Translation != nil {
		e.Translations = splitLines(*m.Translation)
	}
	if m.Collins != nil && *m.Collins > 0 {
		v := *m.Collins
		e.CollinsStars = &v
	}
	if m.Oxford != nil && *m.Oxford == 1 {
		e.OxfordCore = true
	}
	if m.Tag != nil {
		for _, code := range strings.Fields(*m.Tag) {
			e.RawTags = append(e.RawTags, code)
			if label, ok := examTagLabels[code]; ok {
				e.Tags = append(e.Tags, label)
			}
		}
	}
	if m.BNC != nil && *m.BNC > 0 {
		v := *m.BNC
		e.BNCRank = &v
	}
	if m.FRQ != nil && *m.FRQ > 0 {
		v := *m.FRQ
		e.FrequencyRank = &v
	}
	if m.Exchange != nil {
		e.WordForms, e.Lemma = parseExchange(*m.Exchange)
	}
	// 本词条本身就是原形时，不把自己当作变形提示。
	if strings.EqualFold(e.Lemma, e.Word) {
		e.Lemma = ""
	}
	return e
}

// parseExchange 解析 exchange 编码，例如 i:going/p:went/d:gone/3:goes。
// 返回词形列表以及原形（类型码 0 指向 Lemma）。
func parseExchange(raw string) ([]WordForm, string) {
	var forms []WordForm
	var lemma string
	for _, item := range strings.Split(raw, "/") {
		kv := strings.SplitN(item, ":", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[1]) == "" {
			continue
		}
		val := strings.TrimSpace(kv[1])
		if kv[0] == "0" {
			lemma = val
			continue
		}
		label, ok := exchangeTypeLabels[kv[0]]
		if !ok {
			continue
		}
		forms = append(forms, WordForm{Type: label, Word: val})
	}
	return forms, lemma
}

// CanonicalGloss 从中文释义串提取确定性的简短 gloss，
// 出题展示、判分比对与干扰项去重三处统一调用（docs/05 第 5.4 节）。
func CanonicalGloss(translation string) string {
	lines := splitLines(translation)
	if len(lines) == 0 {
		return ""
	}
	first := posPrefixRe.ReplaceAllString(lines[0], "")
	frags := senseSplitRe.Split(first, -1)
	var picked []string
	for _, f := range frags {
		if t := strings.TrimSpace(f); t != "" {
			picked = append(picked, t)
			if len(picked) == 2 {
				break
			}
		}
	}
	return strings.Join(picked, "，")
}

// CanonicalGlossOf 返回 Entry 的规范 gloss（用第一条中文释义）。
func (e *Entry) CanonicalGlossOf() string {
	if len(e.Translations) == 0 {
		return ""
	}
	return CanonicalGloss(strings.Join(e.Translations, lineSep))
}

// IsPhrase 判断词条是否为词组（含空格或连字符），首版单词队列排除词组。
func IsPhrase(word string) bool {
	return strings.ContainsAny(word, " -")
}

// Levenshtein 计算两个字符串的编辑距离（按 rune）。
func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

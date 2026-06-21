package wordlearning

import (
	"math/rand"
	"strings"

	"lingua-buddy/internal/lexicon"
	"lingua-buddy/internal/models"
)

// QuestionView 返回给前端的一道题（不含正确答案）。
type QuestionView struct {
	QuestionType string   `json:"question_type"`
	Stage        string   `json:"stage"`
	MasteryLabel string   `json:"mastery_label"`
	Prompt       string   `json:"prompt"`
	Word         string   `json:"word,omitempty"` // 拼写题展示原词供朗读，可选；选择题为空
	Options      []string `json:"options,omitempty"`
	Token        string   `json:"token"`
	UserWordID   uint64   `json:"user_word_id"`
}

// QuestionBuilder 组装题目并签发令牌。
type QuestionBuilder struct {
	tokens *TokenManager
}

// NewQuestionBuilder 构造。
func NewQuestionBuilder(tokens *TokenManager) *QuestionBuilder {
	return &QuestionBuilder{tokens: tokens}
}

// Build 根据阶段与干扰项组装题目。distractors 为选择题的三个干扰项（拼写或释义）；拼写题传 nil。
func (b *QuestionBuilder) Build(entry *lexicon.Entry, uw *models.UserWord, planID, planItemID *uint64, distractors []string) (*QuestionView, error) {
	qtype := StageForQuestionType(uw.LearningStage)
	gloss := entry.CanonicalGlossOf()

	var prompt string
	var options []string
	switch qtype {
	case QTypeWordToMeaningChoice:
		prompt = entry.Word
		options = shuffle(append([]string{gloss}, distractors...))
	case QTypeMeaningToWordChoice:
		prompt = gloss
		options = shuffle(append([]string{entry.Word}, distractors...))
	default: // spelling
		prompt = gloss
		options = nil
	}

	tok := &QuestionToken{
		UserID:       uw.UserID,
		PlanID:       planID,
		PlanItemID:   planItemID,
		UserWordID:   uw.ID,
		Stage:        uw.LearningStage,
		QuestionType: qtype,
		QuestionKey:  questionKey(entry.Word),
		Options:      options,
		GenVersion:   "word-v1",
	}
	signed, err := b.tokens.Sign(tok)
	if err != nil {
		return nil, err
	}

	view := &QuestionView{
		QuestionType: qtype,
		Stage:        uw.LearningStage,
		MasteryLabel: MasteryLabel(uw.LearningStage),
		Prompt:       prompt,
		Options:      options,
		Token:        signed,
		UserWordID:   uw.ID,
	}
	if qtype == QTypeMeaningToWordSpell {
		view.Word = "" // 拼写题不回传答案
	}
	return view, nil
}

func questionKey(word string) string {
	return "word:" + strings.ToLower(strings.TrimSpace(word))
}

func shuffle(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

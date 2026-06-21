package ai

import (
	"context"
	"strings"
)

// Mock 是离线/无密钥时使用的确定性 Provider，便于本地联调与测试。
type Mock struct{}

// NewMock 构造 Mock Provider。
func NewMock() *Mock { return &Mock{} }

// Translate 返回带标记的占位翻译。
func (m *Mock) Translate(_ context.Context, in TranslationInput) (TranslationOutput, error) {
	return TranslationOutput{
		TranslatedText: "[mock译文] " + in.Text,
		KeyExpressions: []KeyExpression{{Expression: firstWords(in.Text), ExplanationZh: "示例关键表达说明"}},
		Alternatives:   []string{"[mock备选] " + in.Text},
	}, nil
}

// CompareTranslation 返回占位对比结果。
func (m *Mock) CompareTranslation(_ context.Context, in TranslationCompareInput) (TranslationCompareOutput, error) {
	return TranslationCompareOutput{
		ReferenceText: "[mock参考译文] " + in.SourceText,
		Accuracy:      "含义基本准确（mock）",
		GrammarIssues: "无明显语法问题（mock）",
		Naturalness:   "可更自然（mock）",
		Suggestion:    "[mock改写] " + in.UserText,
	}, nil
}

// AnalyzeGrammar 返回占位结构分析。
func (m *Mock) AnalyzeGrammar(_ context.Context, in GrammarInput) (GrammarAnalysisOutput, error) {
	return GrammarAnalysisOutput{
		SentenceType:  "simple",
		MainClause:    MainClause{Subject: firstWords(in.Text), Predicate: "is", Complement: "example"},
		Tense:         "simple_present",
		Voice:         "active",
		GrammarPoints: []GrammarPoint{{Name: "mock point", ExplanationZh: "这是示例语法点说明。"}},
		ExplanationZh: "这是 mock 语法分析，仅用于本地联调。",
	}, nil
}

// Correct 返回占位纠错。
func (m *Mock) Correct(_ context.Context, in CorrectionInput) (CorrectionOutput, error) {
	return CorrectionOutput{
		CorrectedText: in.Text,
		Issues: []Issue{{
			Type: "grammar", Original: firstWords(in.Text), Replacement: firstWords(in.Text),
			ExplanationZh: "mock 示例：未发现需要修改的明显问题。",
		}},
	}, nil
}

// GenerateExamples 占位例句。
func (m *Mock) GenerateExamples(_ context.Context, in ExampleInput) ([]Example, error) {
	return []Example{{English: "This is a mock example with " + in.Word + ".", Chinese: "这是包含 " + in.Word + " 的示例句。", WordMeaning: "示例含义"}}, nil
}

// Chat 占位对话。
func (m *Mock) Chat(_ context.Context, in ChatInput) (ChatOutput, error) {
	return ChatOutput{Reply: "[mock] Sure, let's continue. You said: " + in.UserMessage, Feedback: "mock 反馈：表达清晰。"}, nil
}

// ReviewEssay 占位批改。
func (m *Mock) ReviewEssay(_ context.Context, in EssayInput) (EssayReviewOutput, error) {
	return EssayReviewOutput{
		OverallComment: "[mock] 结构清晰，可继续提升用词。",
		Scores:         EssayScores{Grammar: 80, Vocabulary: 78, Structure: 82, Coherence: 80},
		RevisedText:    in.Body,
		RevisionReason: "mock：示例修改原因。",
	}, nil
}

// GenerateTranslationExercise 占位翻译题。
func (m *Mock) GenerateTranslationExercise(_ context.Context, in TranslationExerciseInput) (TranslationExercise, error) {
	if in.Direction == "en_to_zh" {
		return TranslationExercise{Text: "Knowledge is power."}, nil
	}
	return TranslationExercise{Text: "知识就是力量。"}, nil
}

// EvaluateTranslation 占位评价。
func (m *Mock) EvaluateTranslation(_ context.Context, in TranslationEvaluationInput) (TranslationEvaluation, error) {
	return TranslationEvaluation{
		ReferenceText: "[mock参考] " + in.SourceText,
		Accuracy:      "基本准确（mock）", GrammarIssues: "无明显问题（mock）",
		Naturalness: "可更自然（mock）", Suggestion: "[mock改写] " + in.UserText,
	}, nil
}

// GenerateEssayTopic 占位作文题。
func (m *Mock) GenerateEssayTopic(_ context.Context, in EssayTopicInput) (EssayTopic, error) {
	return EssayTopic{Title: "[mock] The Importance of Learning English", Requirement: "mock：不少于 120 词。"}, nil
}

func firstWords(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 30 {
		return s[:30]
	}
	return s
}

// Package ai 定义可替换的大模型 Provider 抽象与输入输出类型。
package ai

import "context"

// TranslationInput 翻译输入。
type TranslationInput struct {
	Text       string
	SourceLang string // zh/en/auto
	TargetLang string // zh/en
	Tone       string // default/daily/formal/business/academic
	Level      string // 英语水平
}

// KeyExpression 关键表达说明。
type KeyExpression struct {
	Expression    string `json:"expression"`
	ExplanationZh string `json:"explanation_zh"`
}

// TranslationOutput 翻译输出。
type TranslationOutput struct {
	TranslatedText string          `json:"translated_text"`
	KeyExpressions []KeyExpression `json:"key_expressions"`
	Alternatives   []string        `json:"alternatives"`
}

// TranslationCompareInput 用户译文对比输入。
type TranslationCompareInput struct {
	SourceText string
	UserText   string
	Level      string
}

// TranslationCompareOutput 用户译文对比输出。
type TranslationCompareOutput struct {
	ReferenceText string `json:"reference_text"`
	Accuracy      string `json:"accuracy"`
	GrammarIssues string `json:"grammar_issues"`
	Naturalness   string `json:"naturalness"`
	Suggestion    string `json:"suggestion"`
}

// GrammarInput 语法分析输入。
type GrammarInput struct {
	Text  string
	Level string
}

// Clause 从句。
type Clause struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GrammarPoint 语法点。
type GrammarPoint struct {
	Name          string `json:"name"`
	ExplanationZh string `json:"explanation_zh"`
}

// MainClause 句子主干。
type MainClause struct {
	Subject    string `json:"subject"`
	Predicate  string `json:"predicate"`
	Object     string `json:"object"`
	Complement string `json:"complement"`
}

// GrammarAnalysisOutput 语法分析输出。
type GrammarAnalysisOutput struct {
	SentenceType  string         `json:"sentence_type"`
	MainClause    MainClause     `json:"main_clause"`
	Clauses       []Clause       `json:"clauses"`
	Tense         string         `json:"tense"`
	Voice         string         `json:"voice"`
	GrammarPoints []GrammarPoint `json:"grammar_points"`
	ExplanationZh string         `json:"explanation_zh"`
}

// CorrectionInput 纠错/润色输入。
type CorrectionInput struct {
	Text  string
	Mode  string // correct / polish
	Style string // 润色风格：natural/concise/formal/advanced
	Level string
}

// Issue 错误项。
type Issue struct {
	Type          string `json:"type"`
	Original      string `json:"original"`
	Replacement   string `json:"replacement"`
	ExplanationZh string `json:"explanation_zh"`
}

// CorrectionOutput 纠错/润色输出。
type CorrectionOutput struct {
	CorrectedText string  `json:"corrected_text"`
	Issues        []Issue `json:"issues"`
}

// Provider 大模型能力抽象（首版含翻译、语法分析、纠错/润色）。
type Provider interface {
	Translate(ctx context.Context, in TranslationInput) (TranslationOutput, error)
	CompareTranslation(ctx context.Context, in TranslationCompareInput) (TranslationCompareOutput, error)
	AnalyzeGrammar(ctx context.Context, in GrammarInput) (GrammarAnalysisOutput, error)
	Correct(ctx context.Context, in CorrectionInput) (CorrectionOutput, error)
}

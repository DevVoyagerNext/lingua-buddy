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

// ExampleInput AI 例句输入。
type ExampleInput struct {
	Word       string
	Topic      string
	Difficulty string
	Level      string
	Count      int
}

// Example AI 例句。
type Example struct {
	English     string `json:"english"`
	Chinese     string `json:"chinese"`
	WordMeaning string `json:"word_meaning"`
}

// ChatTurn 对话历史一轮。
type ChatTurn struct {
	Role    string `json:"role"` // user/assistant
	Content string `json:"content"`
}

// ChatInput 情景对话输入。
type ChatInput struct {
	Scene       string
	Difficulty  string
	Role        string
	Goal        string
	Level       string
	History     []ChatTurn
	UserMessage string
}

// ChatOutput 情景对话输出。
type ChatOutput struct {
	Reply    string `json:"reply"`
	Feedback string `json:"feedback"` // 对用户上一条回复的中文反馈，可空
}

// EssayInput 作文批改输入。
type EssayInput struct {
	Title       string
	Body        string
	EssayType   string
	Requirement string
	TargetExam  string
	Level       string
}

// EssayScores 分项评分。
type EssayScores struct {
	Grammar    int `json:"grammar"`
	Vocabulary int `json:"vocabulary"`
	Structure  int `json:"structure"`
	Coherence  int `json:"coherence"`
}

// EssayReviewOutput 作文批改输出。
type EssayReviewOutput struct {
	OverallComment string      `json:"overall_comment"`
	Scores         EssayScores `json:"scores"`
	Issues         []Issue     `json:"issues"`
	RevisedText    string      `json:"revised_text"`
	RevisionReason string      `json:"revision_reason"`
}

// TranslationExerciseInput 翻译训练出题输入。
type TranslationExerciseInput struct {
	Direction  string // zh_to_en / en_to_zh
	Difficulty string
	Level      string
}

// TranslationExercise 翻译训练题目。
type TranslationExercise struct {
	Text string `json:"text"` // 待翻译原文
}

// TranslationEvaluationInput 翻译训练评价输入。
type TranslationEvaluationInput struct {
	Direction  string
	SourceText string
	UserText   string
	Level      string
}

// TranslationEvaluation 翻译训练评价输出。
type TranslationEvaluation struct {
	ReferenceText string `json:"reference_text"`
	Accuracy      string `json:"accuracy"`
	GrammarIssues string `json:"grammar_issues"`
	Naturalness   string `json:"naturalness"`
	Suggestion    string `json:"suggestion"`
}

// EssayTopicInput 作文训练出题输入。
type EssayTopicInput struct {
	EssayType  string
	Difficulty string
	Level      string
}

// EssayTopic 作文训练题目。
type EssayTopic struct {
	Title       string `json:"title"`
	Requirement string `json:"requirement"`
}

// Provider 大模型能力抽象。
type Provider interface {
	Translate(ctx context.Context, in TranslationInput) (TranslationOutput, error)
	CompareTranslation(ctx context.Context, in TranslationCompareInput) (TranslationCompareOutput, error)
	AnalyzeGrammar(ctx context.Context, in GrammarInput) (GrammarAnalysisOutput, error)
	Correct(ctx context.Context, in CorrectionInput) (CorrectionOutput, error)
	GenerateExamples(ctx context.Context, in ExampleInput) ([]Example, error)
	Chat(ctx context.Context, in ChatInput) (ChatOutput, error)
	ReviewEssay(ctx context.Context, in EssayInput) (EssayReviewOutput, error)
	GenerateTranslationExercise(ctx context.Context, in TranslationExerciseInput) (TranslationExercise, error)
	EvaluateTranslation(ctx context.Context, in TranslationEvaluationInput) (TranslationEvaluation, error)
	GenerateEssayTopic(ctx context.Context, in EssayTopicInput) (EssayTopic, error)
}

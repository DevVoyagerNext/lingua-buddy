package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 通用错误。
var (
	ErrTimeout         = errors.New("ai timeout")
	ErrInvalidResponse = errors.New("ai invalid response")
)

// DashScope 通过 OpenAI 兼容模式调用通义千问。
type DashScope struct {
	base   string
	key    string
	model  string
	client *http.Client
}

// NewDashScope 构造千问 Provider。
func NewDashScope(base, key, model string) *DashScope {
	if model == "" {
		model = "qwen-plus"
	}
	return &DashScope{
		base:   strings.TrimRight(base, "/"),
		key:    key,
		model:  model,
		client: &http.Client{Timeout: 40 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
	Temperature    float64        `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// chatJSON 发送一次对话并把返回内容解析进 out（要求模型输出 JSON）。
func (d *DashScope) chatJSON(ctx context.Context, system, user string, out any) error {
	reqBody := chatRequest{
		Model: d.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		ResponseFormat: map[string]any{"type": "json_object"},
		Temperature:    0.3,
	}
	buf, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.base+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.key)

	resp, err := d.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrTimeout
		}
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dashscope http %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil || len(cr.Choices) == 0 {
		return ErrInvalidResponse
	}
	content := stripFences(cr.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), out); err != nil {
		return ErrInvalidResponse
	}
	return nil
}

// Translate 翻译。
func (d *DashScope) Translate(ctx context.Context, in TranslationInput) (TranslationOutput, error) {
	system := "你是英语学习助手。请将用户文本翻译，并给出关键表达说明与可选备选译法。" +
		"只返回 JSON：{\"translated_text\":string,\"key_expressions\":[{\"expression\":string,\"explanation_zh\":string}],\"alternatives\":[string]}。" +
		"解释用中文，控制在该学习者水平能理解的范围。"
	user := fmt.Sprintf("源语言:%s 目标语言:%s 语气:%s 学习者水平:%s\n原文:\n%s",
		in.SourceLang, in.TargetLang, in.Tone, in.Level, in.Text)
	var out TranslationOutput
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// CompareTranslation 用户译文对比。
func (d *DashScope) CompareTranslation(ctx context.Context, in TranslationCompareInput) (TranslationCompareOutput, error) {
	system := "你是英语写作评审。对比用户译文与原文，给参考译文与准确性、语法、自然度反馈和改写建议。" +
		"只返回 JSON：{\"reference_text\":string,\"accuracy\":string,\"grammar_issues\":string,\"naturalness\":string,\"suggestion\":string}，解释用中文。"
	user := fmt.Sprintf("学习者水平:%s\n原文:\n%s\n用户译文:\n%s", in.Level, in.SourceText, in.UserText)
	var out TranslationCompareOutput
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// AnalyzeGrammar 语法分析（不改写原句）。
func (d *DashScope) AnalyzeGrammar(ctx context.Context, in GrammarInput) (GrammarAnalysisOutput, error) {
	system := "你是英语语法分析老师。只分析句子结构，不修改原文、不判断对错。" +
		"只返回 JSON：{\"sentence_type\":string,\"main_clause\":{\"subject\":string,\"predicate\":string,\"object\":string,\"complement\":string}," +
		"\"clauses\":[{\"type\":string,\"text\":string}],\"tense\":string,\"voice\":string," +
		"\"grammar_points\":[{\"name\":string,\"explanation_zh\":string}],\"explanation_zh\":string}。解释用中文。"
	user := fmt.Sprintf("学习者水平:%s\n句子:\n%s", in.Level, in.Text)
	var out GrammarAnalysisOutput
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// Correct 纠错或润色。
func (d *DashScope) Correct(ctx context.Context, in CorrectionInput) (CorrectionOutput, error) {
	task := "纠正语法、拼写、用词、标点和不自然表达"
	if in.Mode == "polish" {
		task = fmt.Sprintf("按“%s”风格润色，尽量保持原意，不凭空增加事实", in.Style)
	}
	system := "你是英语写作助手。" + task + "。" +
		"只返回 JSON：{\"corrected_text\":string,\"issues\":[{\"type\":string,\"original\":string,\"replacement\":string,\"explanation_zh\":string}]}。" +
		"type 取值 grammar/spelling/word_choice/punctuation/unnatural。解释用中文。"
	user := fmt.Sprintf("学习者水平:%s\n文本:\n%s", in.Level, in.Text)
	var out CorrectionOutput
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// GenerateExamples 生成 AI 例句。
func (d *DashScope) GenerateExamples(ctx context.Context, in ExampleInput) ([]Example, error) {
	n := in.Count
	if n <= 0 {
		n = 3
	}
	system := fmt.Sprintf("你是英语例句老师。为目标单词生成 %d 个例句。"+
		"只返回 JSON：{\"examples\":[{\"english\":string,\"chinese\":string,\"word_meaning\":string}]}。"+
		"english 为英文例句，chinese 为中文翻译，word_meaning 说明目标词在句中的含义。", n)
	user := fmt.Sprintf("目标单词:%s 主题:%s 难度:%s 学习者水平:%s", in.Word, in.Topic, in.Difficulty, in.Level)
	var wrap struct {
		Examples []Example `json:"examples"`
	}
	if err := d.chatJSON(ctx, system, user, &wrap); err != nil {
		return nil, err
	}
	return wrap.Examples, nil
}

// Chat 情景对话。
func (d *DashScope) Chat(ctx context.Context, in ChatInput) (ChatOutput, error) {
	system := fmt.Sprintf("你在进行英语情景对话练习。场景:%s，你扮演:%s，难度:%s，学习者水平:%s。"+
		"用英文回复以维持对话；同时对用户上一条英文回复给简短中文反馈（语法/用词/更自然说法），无明显问题则反馈为空字符串。"+
		"只返回 JSON：{\"reply\":string,\"feedback\":string}。reply 用英文，feedback 用中文。",
		in.Scene, in.Role, in.Difficulty, in.Level)
	var sb strings.Builder
	for _, t := range in.History {
		sb.WriteString(t.Role)
		sb.WriteString(": ")
		sb.WriteString(t.Content)
		sb.WriteString("\n")
	}
	sb.WriteString("user: ")
	sb.WriteString(in.UserMessage)
	var out ChatOutput
	err := d.chatJSON(ctx, system, sb.String(), &out)
	return out, err
}

// ReviewEssay 作文批改。
func (d *DashScope) ReviewEssay(ctx context.Context, in EssayInput) (EssayReviewOutput, error) {
	system := "你是英语作文批改老师。给出总体评价、分项评分(0-100)、问题清单、修改后参考文章与修改原因。" +
		"评分仅供学习参考。只返回 JSON：{\"overall_comment\":string,\"scores\":{\"grammar\":int,\"vocabulary\":int,\"structure\":int,\"coherence\":int}," +
		"\"issues\":[{\"type\":string,\"original\":string,\"replacement\":string,\"explanation_zh\":string}],\"revised_text\":string,\"revision_reason\":string}。解释用中文。"
	user := fmt.Sprintf("作文类型:%s 目标考试:%s 题目要求:%s 学习者水平:%s\n标题:%s\n正文:\n%s",
		in.EssayType, in.TargetExam, in.Requirement, in.Level, in.Title, in.Body)
	var out EssayReviewOutput
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// GenerateTranslationExercise 生成翻译训练题目。
func (d *DashScope) GenerateTranslationExercise(ctx context.Context, in TranslationExerciseInput) (TranslationExercise, error) {
	dir := "中译英"
	if in.Direction == "en_to_zh" {
		dir = "英译中"
	}
	system := "你是翻译训练出题老师。生成一句适合练习的待翻译文本。" +
		"只返回 JSON：{\"text\":string}。" +
		map[bool]string{true: "text 为英文原文。", false: "text 为中文原文。"}[in.Direction == "en_to_zh"]
	user := fmt.Sprintf("方向:%s 难度:%s 学习者水平:%s", dir, in.Difficulty, in.Level)
	var out TranslationExercise
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// EvaluateTranslation 评价翻译训练答案。
func (d *DashScope) EvaluateTranslation(ctx context.Context, in TranslationEvaluationInput) (TranslationEvaluation, error) {
	system := "你是翻译训练评审。给参考译文与准确性、语法、自然度反馈和改写建议。" +
		"只返回 JSON：{\"reference_text\":string,\"accuracy\":string,\"grammar_issues\":string,\"naturalness\":string,\"suggestion\":string}，解释用中文。"
	user := fmt.Sprintf("方向:%s 学习者水平:%s\n原文:\n%s\n用户译文:\n%s", in.Direction, in.Level, in.SourceText, in.UserText)
	var out TranslationEvaluation
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

// GenerateEssayTopic 生成作文训练题目。
func (d *DashScope) GenerateEssayTopic(ctx context.Context, in EssayTopicInput) (EssayTopic, error) {
	system := "你是作文训练出题老师。生成一个英语作文题目与简要要求。" +
		"只返回 JSON：{\"title\":string,\"requirement\":string}。title 为英文题目，requirement 用中文写要求。"
	user := fmt.Sprintf("作文类型:%s 难度:%s 学习者水平:%s", in.EssayType, in.Difficulty, in.Level)
	var out EssayTopic
	err := d.chatJSON(ctx, system, user, &out)
	return out, err
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/app"
	"lingua-buddy/internal/config"
	"lingua-buddy/internal/lexicon"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/platform/database"
	"lingua-buddy/internal/wordlearning"
)

// 集成测试：用真实 MySQL 跑通学习闭环。答案使用 Go 原生 UTF-8 字符串，避免终端编码干扰。

func testConfig() config.Config {
	return config.Config{
		AppEnv:              "test",
		HTTPPort:            "0",
		JWTAccessSecret:     "test-jwt-secret",
		QuestionTokenSecret: "test-qtoken-secret",
		DB:                  config.DBConfig{User: "root", Password: "123456", Addr: "127.0.0.1:3306", Name: "lingua"},
	}
}

type harness struct {
	t      *testing.T
	engine *gin.Engine
	db     *gorm.DB
	lex    *lexicon.Repository
	token  string
}

func newHarness(t *testing.T) *harness {
	gin.SetMode(gin.TestMode)
	cfg := testConfig()
	db, err := database.Open(cfg.DB.DSN(), false)
	if err != nil {
		t.Skipf("无法连接 MySQL，跳过集成测试: %v", err)
	}
	if err := db.AutoMigrate(models.BusinessModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return &harness{t: t, engine: app.New(cfg, db), db: db, lex: lexicon.NewRepository(db)}
}

func (h *harness) req(method, path string, body any) (int, map[string]any) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, path, &buf)
	r.Header.Set("Content-Type", "application/json")
	if h.token != "" {
		r.Header.Set("Authorization", "Bearer "+h.token)
	}
	w := httptest.NewRecorder()
	h.engine.ServeHTTP(w, r)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func (h *harness) mustData(code int, resp map[string]any) map[string]any {
	if code != http.StatusOK {
		h.t.Fatalf("期望 200，得到 %d: %v", code, resp)
	}
	d, _ := resp["data"].(map[string]any)
	return d
}

// register 用唯一用户名注册并保存 token。
func (h *harness) register() {
	name := fmt.Sprintf("itest_%d", time.Now().UnixNano())
	d := h.mustData(h.req(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username": name, "password": "password123",
	}))
	h.token, _ = d["token"].(string)
	if h.token == "" {
		h.t.Fatal("注册未返回 token")
	}
}

// forceDue 把某用户单词的 next_review_at 提前到过去，使其立即到期。
func (h *harness) forceDue(uwID uint64) {
	if err := h.db.Model(&models.UserWord{}).Where("id = ?", uwID).
		Update("next_review_at", time.Now().Add(-time.Minute)).Error; err != nil {
		h.t.Fatalf("forceDue 失败: %v", err)
	}
}

func (h *harness) wordOf(uwID uint64) string {
	var uw models.UserWord
	if err := h.db.First(&uw, uwID).Error; err != nil {
		h.t.Fatalf("读取 user_word 失败: %v", err)
	}
	return uw.Word
}

// correctAnswerFor 根据题型推导正确答案（与服务端 judge 同源），保证字节一致。
func (h *harness) correctAnswerFor(qtype, prompt string, uwID uint64) string {
	word := h.wordOf(uwID)
	entry, err := h.lex.FindExact(context.Background(), word)
	if err != nil {
		h.t.Fatalf("查词失败 %q: %v", word, err)
	}
	if qtype == wordlearning.QTypeWordToMeaningChoice {
		return entry.CanonicalGlossOf()
	}
	return entry.Word
}

func TestLearningLoopStages(t *testing.T) {
	h := newHarness(t)
	h.register()

	// 创建四级计划。
	d := h.mustData(h.req(http.MethodPost, "/api/v1/word-learning/plans", map[string]any{"source_value": "cet4"}))
	plan := d["plan"].(map[string]any)
	if int(plan["source_snapshot_count"].(float64)) < 3000 {
		t.Fatalf("计划词数异常: %v", plan["source_snapshot_count"])
	}

	// 取第一道题并激活活跃窗口。
	code, resp := h.req(http.MethodGet, "/api/v1/word-learning/next", nil)
	q := h.mustData(code, resp)
	uwID := uint64(q["user_word_id"].(float64))
	if q["stage"] != "recognition" || q["question_type"] != wordlearning.QTypeWordToMeaningChoice {
		t.Fatalf("首题应为 recognition 英文选中文，得到 %v/%v", q["stage"], q["question_type"])
	}

	// 驱动同一个词走完 recognition -> discrimination -> spelling -> mastered。
	expectStages := []struct {
		stage     string
		corrects  int // 该阶段需要连续答对几次才晋级
		nextStage string
	}{
		{"recognition", 2, "discrimination"},
		{"discrimination", 2, "spelling"},
		{"spelling", 3, "mastered"},
	}

	sub := 0
	for _, st := range expectStages {
		for i := 0; i < st.corrects; i++ {
			h.forceDue(uwID)
			// 取该词的题：通过反复取题直到命中目标 uwID（活跃窗口有多个词）。
			tok, qtype := h.fetchQuestionFor(uwID)
			ans := h.correctAnswerFor(qtype, "", uwID)
			sub++
			data := h.mustData(h.req(http.MethodPost, "/api/v1/word-learning/answer", map[string]any{
				"submission_id": fmt.Sprintf("sub-%d", sub),
				"token":         tok,
				"answer":        ans,
			}))
			if data["correct"] != true {
				t.Fatalf("阶段 %s 第%d次：正确答案 %q 被判错, resp=%v", st.stage, i+1, ans, data)
			}
		}
		// 晋级后再取该词，阶段应已变化。
		h.forceDue(uwID)
		_, nextQType := h.fetchQuestionFor(uwID)
		_ = nextQType
		var uw models.UserWord
		h.db.First(&uw, uwID)
		if uw.LearningStage != st.nextStage {
			t.Fatalf("阶段 %s 连对%d次后应进入 %s，实际 %s", st.stage, st.corrects, st.nextStage, uw.LearningStage)
		}
	}

	t.Logf("单词成功走完 recognition→discrimination→spelling→mastered，共提交 %d 次", sub)
}

// fetchQuestionFor 反复取题直到拿到目标 user_word 的题，返回 token 与题型。
func (h *harness) fetchQuestionFor(uwID uint64) (string, string) {
	for attempt := 0; attempt < 60; attempt++ {
		code, resp := h.req(http.MethodGet, "/api/v1/word-learning/next", nil)
		d := h.mustData(code, resp)
		gotID := uint64(d["user_word_id"].(float64))
		if gotID == uwID {
			return d["token"].(string), d["question_type"].(string)
		}
		// 不是目标词：把它推到未来，避免反复命中。
		h.db.Model(&models.UserWord{}).Where("id = ?", gotID).
			Update("next_review_at", time.Now().Add(time.Hour))
	}
	h.t.Fatalf("未能取到目标词 %d 的题", uwID)
	return "", ""
}

func TestWrongAnswerJudgedWrong(t *testing.T) {
	h := newHarness(t)
	h.register()
	// 收藏一个散词，单词本里只有它。
	h.mustData(h.req(http.MethodPost, "/api/v1/vocabulary", map[string]any{"word": "ability", "familiarity": "unknown"}))
	code, resp := h.req(http.MethodGet, "/api/v1/word-learning/next", nil)
	q := h.mustData(code, resp)
	data := h.mustData(h.req(http.MethodPost, "/api/v1/word-learning/answer", map[string]any{
		"submission_id": "wrong-1",
		"token":         q["token"],
		"answer":        "这是一个肯定错误的答案",
	}))
	if data["correct"] != false {
		t.Fatalf("错误答案应判错: %v", data)
	}
}

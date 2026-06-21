package wordlearning

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
)

// Handler 暴露生词本与单词学习接口。
type Handler struct {
	svc  *Service
	repo *Repository
}

// NewHandler 构造处理器。
func NewHandler(svc *Service, repo *Repository) *Handler { return &Handler{svc: svc, repo: repo} }

// Register 注册路由（已登录组）。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/vocabulary", h.collect)
	rg.GET("/vocabulary", h.listWords)
	rg.DELETE("/vocabulary/:id", h.removeWord)

	rg.POST("/word-learning/plans", h.createPlan)
	rg.GET("/word-learning/plans", h.listPlans)
	rg.GET("/word-learning/plans/:id", h.getPlan)
	rg.POST("/word-learning/plans/:id/activate", h.activatePlan)
	rg.POST("/word-learning/plans/:id/pause", h.pausePlan)
	rg.GET("/word-learning/due", h.due)
	rg.GET("/word-learning/next", h.next)
	rg.POST("/word-learning/answer", h.answer)
	rg.POST("/word-learning/plan-items/:id/skip", h.skip)
	rg.GET("/word-learning/words", h.planWords)
	rg.GET("/word-learning/wrong-answers", h.wrongAnswers)
	h.RegisterGroups(rg)
}

// WordView 生词视图。
type WordView struct {
	ID                 uint64    `json:"id"`
	Word               string    `json:"word"`
	Stage              string    `json:"stage"`
	MasteryLabel       string    `json:"mastery_label"`
	StageCorrectStreak int       `json:"stage_correct_streak"`
	NextReviewAt       time.Time `json:"next_review_at"`
	CreatedAt          time.Time `json:"created_at"`
	Definition         string    `json:"definition"`
}

func toWordView(u models.UserWord) WordView {
	return WordView{
		ID: u.ID, Word: u.Word, Stage: u.LearningStage,
		MasteryLabel: MasteryLabel(u.LearningStage), StageCorrectStreak: u.StageCorrectStreak,
		NextReviewAt: u.NextReviewAt, CreatedAt: u.CreatedAt,
	}
}

type collectReq struct {
	Word        string `json:"word"`
	Familiarity string `json:"familiarity"`
}

func (h *Handler) collect(c *gin.Context) {
	var req collectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	uw, err := h.svc.CollectWord(c.Request.Context(), httpx.MustUserID(c), req.Word, req.Familiarity)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toWordView(*uw))
}

func (h *Handler) listWords(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.svc.ListWords(c.Request.Context(), httpx.MustUserID(c), c.Query("stage"), c.Query("keyword"), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views := make([]WordView, 0, len(items))
	for _, it := range items {
		v := toWordView(it.Word)
		v.Definition = it.Definition
		views = append(views, v)
	}
	httpx.OK(c, httpx.Page{Items: views, Page: page, PageSize: size, Total: total})
}

func (h *Handler) removeWord(c *gin.Context) {
	id := parseID(c)
	if err := h.svc.RemoveWord(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

type createPlanReq struct {
	Name        string `json:"name"`
	SourceValue string `json:"source_value"`
}

func (h *Handler) createPlan(c *gin.Context) {
	var req createPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	plan, counts, err := h.svc.CreatePlan(c.Request.Context(), httpx.MustUserID(c), req.Name, req.SourceValue)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, PlanDetail{Plan: plan, Counts: counts})
}

func (h *Handler) listPlans(c *gin.Context) {
	plans, err := h.svc.ListPlans(c.Request.Context(), httpx.MustUserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, plans)
}

func (h *Handler) getPlan(c *gin.Context) {
	detail, err := h.svc.GetPlan(c.Request.Context(), httpx.MustUserID(c), parseID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, detail)
}

func (h *Handler) activatePlan(c *gin.Context) {
	if err := h.svc.ActivatePlan(c.Request.Context(), httpx.MustUserID(c), parseID(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"status": "active"})
}

func (h *Handler) pausePlan(c *gin.Context) {
	if err := h.svc.PausePlan(c.Request.Context(), httpx.MustUserID(c), parseID(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"status": "paused"})
}

func (h *Handler) due(c *gin.Context) {
	uid := httpx.MustUserID(c)
	dueCount, err := h.repo.CountDueWords(c.Request.Context(), uid, time.Now())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	resp := gin.H{"due_count": dueCount}
	if plan, _ := h.repo.GetActivePlan(c.Request.Context(), uid); plan != nil {
		if counts, err := h.repo.CountPlanItems(c.Request.Context(), plan.ID); err == nil {
			resp["plan"] = plan
			resp["counts"] = counts
		}
	}
	httpx.OK(c, resp)
}

func (h *Handler) next(c *gin.Context) {
	res, err := h.svc.Next(c.Request.Context(), httpx.MustUserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if res.Question != nil {
		httpx.OK(c, res.Question)
		return
	}
	// NO_DUE_WORDS / NO_ACTIVE_PLAN：以业务码返回 200，前端据此引导。
	c.JSON(200, httpx.Response{Code: res.Status, Message: statusMessage(res.Status), Data: gin.H{"next_due_at": res.NextDueAt}})
}

func statusMessage(status string) string {
	switch status {
	case "NO_ACTIVE_PLAN":
		return "当前没有可学的单词，去创建计划或收藏生词吧"
	case "NO_DUE_WORDS":
		return "暂时没有到期的单词，稍后再来"
	default:
		return status
	}
}

type answerReq struct {
	SubmissionID string `json:"submission_id"`
	Token        string `json:"token"`
	Answer       string `json:"answer"`
	UsedHint     bool   `json:"used_hint"`
}

func (h *Handler) answer(c *gin.Context) {
	var req answerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.SubmitAnswer(c.Request.Context(), httpx.MustUserID(c), req.SubmissionID, req.Token, req.Answer, req.UsedHint)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *Handler) skip(c *gin.Context) {
	if err := h.svc.SkipItem(c.Request.Context(), httpx.MustUserID(c), parseID(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"skipped": true})
}

func (h *Handler) planWords(c *gin.Context) {
	planID, err := strconv.ParseUint(c.Query("plan_id"), 10, 64)
	if err != nil {
		httpx.Fail(c, httpx.ErrValidation("缺少 plan_id"))
		return
	}
	page, size := httpx.ParsePagination(c)
	rows, total, err := h.svc.ListPlanWords(c.Request.Context(), httpx.MustUserID(c), planID, page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: rows, Page: page, PageSize: size, Total: total})
}

func (h *Handler) wrongAnswers(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	rows, total, err := h.svc.ListWrongAnswers(c.Request.Context(), httpx.MustUserID(c), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: rows, Page: page, PageSize: size, Total: total})
}

func parseID(c *gin.Context) uint64 {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	return id
}

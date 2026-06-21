package dictionary

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/lexicon"
)

// Handler 暴露词典 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造词典处理器。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册词典路由（已登录组）。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/dictionary/entries/:word", h.lookup)
	rg.GET("/dictionary/suggestions", h.suggest)
	rg.GET("/dictionary/history", h.history)
	rg.DELETE("/dictionary/history/:id", h.deleteHistory)
}

func (h *Handler) lookup(c *gin.Context) {
	word := c.Param("word")
	entry, err := h.svc.Lookup(c.Request.Context(), httpx.MustUserID(c), word)
	if errors.Is(err, lexicon.ErrNotFound) {
		// 未命中：附带相近拼写建议返回 NOT_FOUND。
		suggestions, _ := h.svc.Similar(c.Request.Context(), word, 5)
		c.JSON(http.StatusNotFound, httpx.Response{
			Code:    "NOT_FOUND",
			Message: "未找到该单词",
			Data:    gin.H{"suggestions": suggestions},
		})
		return
	}
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, entry)
}

func (h *Handler) suggest(c *gin.Context) {
	q := c.Query("q")
	if len([]rune(q)) < 2 {
		httpx.OK(c, []lexicon.Suggestion{})
		return
	}
	items, err := h.svc.Suggest(c.Request.Context(), q, 10)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, items)
}

func (h *Handler) history(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.svc.History(c.Request.Context(), httpx.MustUserID(c), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) deleteHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, httpx.ErrValidation("无效的 ID"))
		return
	}
	if err := h.svc.DeleteHistory(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		httpx.Fail(c, httpx.ErrNotFound("历史记录不存在"))
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

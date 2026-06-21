package history

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/httpx"
)

// Handler 暴露统一历史接口。
type Handler struct {
	repo *Repository
}

// NewHandler 构造历史处理器。
func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/history", h.list)
	rg.DELETE("/history/:id", h.delete)
}

func (h *Handler) list(c *gin.Context) {
	page, size := httpx.ParsePagination(c)
	items, total, err := h.repo.List(c.Request.Context(), httpx.MustUserID(c), c.Query("type"), page, size)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, httpx.Page{Items: items, Page: page, PageSize: size, Total: total})
}

func (h *Handler) delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, httpx.ErrValidation("无效的 ID"))
		return
	}
	if err := h.repo.Delete(c.Request.Context(), httpx.MustUserID(c), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.Fail(c, httpx.ErrNotFound("历史记录不存在"))
			return
		}
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

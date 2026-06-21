package user

import (
	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/httpx"
)

// Handler 暴露 /users/me 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造用户处理器。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册需要登录的用户路由（rg 已挂鉴权中间件）。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/users/me", h.me)
	rg.PATCH("/users/me", h.update)
}

func (h *Handler) me(c *gin.Context) {
	u, err := h.svc.Me(c.Request.Context(), httpx.MustUserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, ToView(u))
}

type updateReq struct {
	Email        *string `json:"email"`
	EnglishLevel *string `json:"english_level"`
}

func (h *Handler) update(c *gin.Context) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	u, err := h.svc.Update(c.Request.Context(), httpx.MustUserID(c), UpdateInput{
		Email:        req.Email,
		EnglishLevel: req.EnglishLevel,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, ToView(u))
}

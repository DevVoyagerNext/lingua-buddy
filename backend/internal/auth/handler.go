package auth

import (
	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/user"
)

// Handler 暴露认证相关 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造认证处理器。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/auth/register", h.register)
	rg.POST("/auth/login", h.login)
}

type registerReq struct {
	Method   string  `json:"method"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	Email    *string `json:"email"`
}

type authResp struct {
	Token string    `json:"token"`
	User  user.View `json:"user"`
}

func (h *Handler) register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.Register(c.Request.Context(), RegistrationRequest{
		Method:   RegistrationMethod(req.Method),
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, authResp{Token: res.Token, User: user.ToView(res.User)})
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, authResp{Token: res.Token, User: user.ToView(res.User)})
}

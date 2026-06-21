// Package middleware 提供 Gin 中间件。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"lingua-buddy/internal/auth"
	"lingua-buddy/internal/httpx"
)

// AuthRequired 校验 Authorization: Bearer <token>，写入当前用户 ID。
func AuthRequired(jwt *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			httpx.Fail(c, httpx.ErrUnauthorized("缺少或无效的 Authorization 头"))
			c.Abort()
			return
		}
		userID, err := jwt.Parse(parts[1])
		if err != nil {
			httpx.Fail(c, httpx.ErrUnauthorized("登录已失效，请重新登录"))
			c.Abort()
			return
		}
		httpx.SetUserID(c, userID)
		c.Next()
	}
}

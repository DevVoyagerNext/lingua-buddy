package httpx

import "github.com/gin-gonic/gin"

const ctxUserIDKey = "auth_user_id"

// SetUserID 由鉴权中间件写入当前登录用户 ID。
func SetUserID(c *gin.Context, id uint64) {
	c.Set(ctxUserIDKey, id)
}

// CurrentUserID 返回当前登录用户 ID；未登录返回 0 和 false。
func CurrentUserID(c *gin.Context) (uint64, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}

// MustUserID 返回当前用户 ID，约定仅在鉴权中间件之后调用。
func MustUserID(c *gin.Context) uint64 {
	id, _ := CurrentUserID(c)
	return id
}

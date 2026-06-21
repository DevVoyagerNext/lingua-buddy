package httpx

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePagination 从 query 解析 page/page_size，默认第 1 页每页 20，最大 100。
func ParsePagination(c *gin.Context) (page, size int) {
	page = 1
	size = 20
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("page_size")); err == nil && v > 0 {
		size = v
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

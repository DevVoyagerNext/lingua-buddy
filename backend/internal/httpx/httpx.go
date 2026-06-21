// Package httpx 提供统一的 API 响应、错误码与分页结构。
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应体。
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Page 分页响应体。
type Page struct {
	Items    any   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// APIError 携带 HTTP 状态、业务错误码与可读信息。
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// NewError 构造一个 APIError。
func NewError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// 常用错误码（与 docs/02 第 11 节一致）。
var (
	ErrValidation   = func(msg string) *APIError { return NewError(http.StatusBadRequest, "VALIDATION_ERROR", msg) }
	ErrUnauthorized = func(msg string) *APIError { return NewError(http.StatusUnauthorized, "UNAUTHORIZED", msg) }
	ErrForbidden    = func(msg string) *APIError { return NewError(http.StatusForbidden, "FORBIDDEN", msg) }
	ErrNotFound     = func(msg string) *APIError { return NewError(http.StatusNotFound, "NOT_FOUND", msg) }
	ErrConflict     = func(msg string) *APIError { return NewError(http.StatusConflict, "CONFLICT", msg) }
	ErrInternal     = func(msg string) *APIError { return NewError(http.StatusInternalServerError, "INTERNAL_ERROR", msg) }
)

// OK 返回成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: "OK", Message: "success", Data: data})
}

// Fail 根据错误返回失败响应。非 APIError 一律按 INTERNAL_ERROR 处理。
func Fail(c *gin.Context, err error) {
	if apiErr, ok := err.(*APIError); ok {
		c.JSON(apiErr.Status, Response{Code: apiErr.Code, Message: apiErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, Response{Code: "INTERNAL_ERROR", Message: err.Error()})
}

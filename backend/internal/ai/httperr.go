package ai

import (
	"errors"
	"net/http"
)

// ErrorCode 把 Provider 错误映射为 HTTP 状态、业务码与可读信息，供 handler 使用。
func ErrorCode(err error) (status int, code, message string) {
	switch {
	case errors.Is(err, ErrTimeout):
		return http.StatusGatewayTimeout, "AI_TIMEOUT", "AI 调用超时，请重试"
	case errors.Is(err, ErrInvalidResponse):
		return http.StatusBadGateway, "AI_INVALID_RESPONSE", "AI 返回无法解析，请重试"
	default:
		return http.StatusBadGateway, "AI_INVALID_RESPONSE", "AI 调用失败，请重试"
	}
}

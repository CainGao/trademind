// Package errors 自定义错误类型（规范 V1.0 §4.2）。
//
// 所有业务错误必须用本包构造，Handler 通过 response.HandleError 统一处理。
package errors

import "fmt"

// 错误码（规范 V1.0 §3.4）
const (
	ErrValidation      = 1001 // 参数校验失败
	ErrNotFound        = 1002 // 资源不存在
	ErrConflict        = 1003 // 资源冲突
	ErrBusinessRule    = 1004 // 业务规则违反

	ErrUnauthorized    = 2001 // 未登录
	ErrTokenExpired    = 2002 // Token 过期
	ErrForbidden       = 2003 // 无权限

	ErrAIKeyMissing    = 3001 // 未配置 AI Key
	ErrAIQuotaExceeded = 3002 // AI 额度用完
	ErrAITimeout       = 3003 // AI 调用超时
)

// AppError 应用错误。带错误码 + 消息 + 原因。
type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

// Validation 参数校验错误。
func Validation(format string, args ...interface{}) *AppError {
	return &AppError{Code: ErrValidation, Message: fmt.Sprintf(format, args...)}
}

// NotFound 资源不存在。
func NotFound(resource string, id interface{}) *AppError {
	return &AppError{Code: ErrNotFound, Message: fmt.Sprintf("%s 不存在 [id=%v]", resource, id)}
}

// Forbidden 无权限。
func Forbidden(msg string) *AppError {
	return &AppError{Code: ErrForbidden, Message: msg}
}

// BusinessRule 业务规则违反。
func BusinessRule(format string, args ...interface{}) *AppError {
	return &AppError{Code: ErrBusinessRule, Message: fmt.Sprintf(format, args...)}
}

// Wrap 包装底层错误，带上错误码和上下文。
func Wrap(code int, msg string, cause error) *AppError {
	return &AppError{Code: code, Message: msg, Cause: cause}
}

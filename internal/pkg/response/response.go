// Package response 统一 API 响应格式（规范 V1.0 §3.2）。
//
// 所有 Handler 必须用本包返回，禁止裸 c.JSON()。
package response

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构。
// Code=0 表示成功，非 0 表示错误码（规范 V1.0 §3.4）。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据。
type PageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// Success 成功响应。
// 对 slice 类型自动将 nil 转为空数组，防止 JSON 输出 null（gotcha #45 扩展）。
func Success(c *gin.Context, data interface{}) {
	if data != nil {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			data = reflect.MakeSlice(v.Type(), 0, 0).Interface()
		}
	}
	c.JSON(200, Response{Code: 0, Message: "success", Data: data})
}

// SuccessPage 分页响应。
// 自动将 nil 切片转为空数组，防止 JSON 输出 null（gotcha #45）。
func SuccessPage(c *gin.Context, list interface{}, total int64, page, size int) {
	if list != nil {
		v := reflect.ValueOf(list)
		if v.Kind() == reflect.Slice && v.IsNil() {
			list = reflect.MakeSlice(v.Type(), 0, 0).Interface()
		}
	}
	c.JSON(200, Response{Code: 0, Message: "success", Data: PageData{
		List: list, Total: total, Page: page, Size: size,
	}})
}

// Created 创建成功（HTTP 201）。
func Created(c *gin.Context, data interface{}) {
	c.JSON(201, Response{Code: 0, Message: "created", Data: data})
}

// BadRequest 参数错误（HTTP 400）。
func BadRequest(c *gin.Context, msg string) {
	c.JSON(400, Response{Code: 400, Message: msg})
}

// Unauthorized 未认证（HTTP 401）。
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(401, Response{Code: 2001, Message: msg})
}

// Forbidden 无权限（HTTP 403）。
func Forbidden(c *gin.Context, msg string) {
	c.JSON(403, Response{Code: 2003, Message: msg})
}

// NotFound 资源不存在（HTTP 404）。
func NotFound(c *gin.Context, msg string) {
	c.JSON(404, Response{Code: 1002, Message: msg})
}

// Conflict 资源冲突（HTTP 409）—— 用户名重复、唯一约束冲突等。
func Conflict(c *gin.Context, msg string) {
	c.JSON(409, Response{Code: 1001, Message: msg})
}

// InternalError 内部错误（HTTP 500）。
func InternalError(c *gin.Context, msg string) {
	c.JSON(500, Response{Code: 5000, Message: msg})
}

// TooManyRequests 请求过多/被限流（HTTP 429）—— 登录防暴力破解等场景。
// 调用方可在调用前设置 c.Header("Retry-After", "300")。
func TooManyRequests(c *gin.Context, msg string) {
	c.JSON(429, Response{Code: 2004, Message: msg})
}

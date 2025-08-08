package core

import (
	"context"
	"net/http"

	"github.com/gfa-inc/gfa/middlewares/request_id"

	"github.com/gin-gonic/gin"
)

// Response represents processing result
type Response[T any] struct {
	Success bool   `json:"success" validate:"required"`
	Code    string `json:"code" validate:"required"`
	Message string `json:"msg"`
	Data    T      `json:"data"`
	TraceID string `json:"traceId,omitempty" validate:"required"`
}

type PaginatedData[T any] struct {
	Data  T     `json:"list" validate:"required"`
	Total int64 `json:"total" validate:"required"`
}

func NewSucceedResponse[T any](c context.Context, data T) Response[T] {
	traceID, _ := c.Value(request_id.ContextKey).(string)
	return Response[T]{
		Success: true,
		Code:    "0",
		Message: "",
		Data:    data,
		TraceID: traceID,
	}
}

func NewFailedResponse(c context.Context, code string, message string) Response[any] {
	traceID, _ := c.Value(request_id.ContextKey).(string)
	return Response[any]{
		Success: false,
		Code:    code,
		Message: message,
		Data:    nil,
		TraceID: traceID,
	}
}

// OK returns processing result successfully
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, NewSucceedResponse(c, data))
}

// Fail returns error code and message
func Fail(c *gin.Context, code string, message string) {
	c.JSON(http.StatusServiceUnavailable, NewFailedResponse(c, code, message))
}

// Paginated returns paginated data
func Paginated[T any](data T, total int64) PaginatedData[T] {
	return PaginatedData[T]{data, total}
}

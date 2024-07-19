package core

import (
	"context"
	"github.com/gfa-inc/gfa/middlewares/request_id"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents processing result
type Response struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"msg"`
	Data    any    `json:"data"`
	TraceID string `json:"traceId,omitempty"`
}

type PaginatedData struct {
	Data  any   `json:"list"`
	Total int64 `json:"total"`
}

func NewSucceedResponse(c context.Context, data any) Response {
	traceID, _ := c.Value(request_id.ContextKey).(string)
	return Response{
		Success: true,
		Code:    "0",
		Message: "",
		Data:    data,
		TraceID: traceID,
	}
}

func NewFailedResponse(c context.Context, code string, message string) Response {
	traceID, _ := c.Value(request_id.ContextKey).(string)
	return Response{
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
func Paginated(data any, total int64) PaginatedData {
	return PaginatedData{data, total}
}

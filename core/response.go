package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents processing result
type Response struct {
	Success bool        `json:"success"`
	Code    string      `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
	TraceID string      `json:"traceId,omitempty"`
}

type PaginatedData struct {
	Data  interface{} `json:"list"`
	Total int64       `json:"total"`
}

func NewSucceedResponse(data interface{}) Response {
	return Response{
		Success: true,
		Code:    "0",
		Message: "",
		Data:    data,
	}
}

func NewFailedResponse(code string, message string) Response {
	return Response{
		Success: false,
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// OK returns processing result successfully
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, NewSucceedResponse(data))
}

// Fail returns error code and message
func Fail(c *gin.Context, code string, message string) {
	c.JSON(http.StatusServiceUnavailable, NewFailedResponse(code, message))
}

// Paginated returns paginated data
func Paginated(data interface{}, total int64) PaginatedData {
	return PaginatedData{data, total}
}

package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// SuccessResponse returns a standard success response
func SuccessResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(code, gin.H{
		"success": true,
		"data":    data,
	})
}

// ErrorResponse returns a standard error response
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"success": false,
		"error":   message,
	})
}

// PaginatedResponse returns a paginated response
func PaginatedResponse(c *gin.Context, code int, data interface{}, total int64, page, pageSize int) {
	c.JSON(code, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetIntQuery retrieves a query parameter as an integer or returns a default value
func GetIntQuery(c *gin.Context, key string, defaultValue int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

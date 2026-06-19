package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestID(env string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" && env == "dev" {
			id = uuid.NewString()
		}
		if id != "" {
			c.Set(RequestIDKey, id)
			c.Header("X-Request-ID", id)
		}
		c.Next()
	}
}

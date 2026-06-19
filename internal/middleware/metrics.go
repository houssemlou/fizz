package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/houssemlou/fizz/internal/metrics"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		metrics.RequestsTotal.WithLabelValues(
			c.Request.Method,
			route,
			strconv.Itoa(c.Writer.Status()),
		).Inc()

		metrics.RequestDuration.WithLabelValues(
			c.Request.Method,
			route,
		).Observe(time.Since(start).Seconds())
	}
}

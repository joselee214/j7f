package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joselee214/j7f/components/log"
	"time"
)

func Logger(l *log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		params, _ := c.Get("raw-data")
		l.Info(fmt.Sprintf("Common:%3d | %v | %s |  %s  %s "+
			"Headers:%+v "+
			"Params:%+v "+
			"Errors:%s`",
			statusCode, latency, clientIP, method, path,
			c.GetHeader("Common-Params"),
			params,
			c.Errors.String(),
		))
	}
}

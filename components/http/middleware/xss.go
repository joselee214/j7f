package middleware

import (
	"github.com/gin-gonic/gin"
	"io"
)

func XSS(w io.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

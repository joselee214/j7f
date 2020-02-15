package middleware

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
)

func JsonParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		var param interface{}
		body, err := c.GetRawData()
		c.Set("raw-data", string(body))
		if err != nil {
			err = c.AbortWithError(http.StatusForbidden, err)
			if err != nil {
				_ = c.Error(err)
			}
		}

		if len(body) > 0 {
			err = json.Unmarshal(body, &param)
			if err != nil {
				 err = c.AbortWithError(http.StatusForbidden, err)
				if err != nil {
					_ = c.Error(err)
				}
			}
		}

		c.Set("param", param)
		c.Next()
	}
}

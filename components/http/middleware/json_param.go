package middleware

import (
	"github.com/gin-gonic/gin"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
	//"fmt"
)

func JsonParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()

		if err != nil {
			err = c.AbortWithError(http.StatusForbidden, err)
			if err != nil {
				_ = c.Error(err)
			}
		}

		c.Set("raw-data", string(body))
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		if len(body) > 0 {
			var param interface{}
			err = json.Unmarshal(body, &param)
			if err == nil {
				c.Set("json-param", param)
			}
			//if err != nil {
			//	 err = c.AbortWithError(http.StatusForbidden, err)
				//if err != nil {
				//	_ = c.Error(err)
				//}
			//}
		}
		c.Next()
	}
}

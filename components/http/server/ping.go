package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func PingInit(g *gin.Engine) {
	s := &PingController{}
	j7ping := g.Group("/j7ping")
	j7ping.GET("",s.pong)
}


type PingController struct {
	Controller
}

func (ctrl *PingController) pong(ctx *gin.Context)  {
	p,_ := ctx.GetQuery("ping")
	ctx.JSON(http.StatusOK, gin.H{"data": p, "msg": "pong", "result": 1})
}
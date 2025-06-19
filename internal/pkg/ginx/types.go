package ginx

import "github.com/gin-gonic/gin"

type Handler interface {
	PrivateRoutes(server *gin.Engine)
	PublicRoutes(server *gin.Engine)
}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

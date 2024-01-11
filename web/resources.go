package web

import (
	"embed"
	"github.com/gin-gonic/gin"
	"net/http"
)

//go:embed *.html *.svg *.txt *.css
var resources embed.FS

func Routes(engine *gin.Engine) {
	staticHandler := gin.WrapH(http.FileServer(http.FS(resources)))
	engine.GET("/", staticHandler)
	engine.GET("/:static", staticHandler)
}

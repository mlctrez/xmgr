package api

import (
	_ "embed"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
	"github.com/mlctrez/xmgr/runner"
	"github.com/mlctrez/xmgr/web"
	"github.com/nats-io/nats.go"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Api struct {
	log      service.Logger
	engine   *gin.Engine
	runner   *runner.Runner
	natsConn *nats.Conn
}

func New(log service.Logger, runner *runner.Runner, natsConn *nats.Conn) *Api {
	var engine *gin.Engine
	if os.Getenv("GIN_MODE") == "release" {
		engine = gin.New()
		engine.Use(gin.Recovery())
	} else {
		engine = gin.Default()
	}
	web.Routes(engine)
	a := &Api{log: log, runner: runner, natsConn: natsConn, engine: engine}
	engine.GET("/xonotic-assets/:path", a.Assets)

	return a
}

func (a *Api) Handler() http.Handler {
	return a.engine
}

func (a *Api) Assets(context *gin.Context) {
	assetPath := context.Param("path")
	if !strings.HasSuffix(assetPath, ".pk3") {
		context.Status(http.StatusNotFound)
		_ = a.log.Info(context.RemoteIP(), "invalid asset path", assetPath)
		return
	}

	open, err := os.Open(filepath.Join(os.Getenv("XONOTIC_ASSETS"), assetPath))
	if err != nil {
		context.Status(http.StatusNotFound)
		_ = a.log.Error("open ", err.Error())
		return
	}
	defer func(open *os.File) { _ = open.Close() }(open)
	context.Header("Content-Type", "application/zip")
	context.Header("Content-Disposition", "attachment; filename="+assetPath)
	context.File(open.Name())
}

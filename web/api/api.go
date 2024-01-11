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

	return a
}

func (a *Api) Handler() http.Handler {
	return a.engine
}

func (a *Api) Help(ctx *gin.Context) {
	_ = a.natsConn.Publish("xonotic.stdin", []byte("help\n"))
}

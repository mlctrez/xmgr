package service

import (
	"context"
	_ "embed"
	"errors"
	"github.com/kardianos/service"
	"github.com/mlctrez/servicego"
	"github.com/mlctrez/xmgr/runner"
	"github.com/mlctrez/xmgr/web/api"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var _ servicego.Service = (*Service)(nil)

type Service struct {
	servicego.Defaults
	server     *http.Server
	natsServer *server.Server
	api        *api.Api
	natsConn   *nats.Conn
	runner     *runner.Runner
	monitor    *nats.Subscription
}

func (svc *Service) Config() *service.Config {
	config := svc.Defaults.Config()
	config.UserName = "xonotic"
	return config
}

func xonoticDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "xonotic"), nil
}

func (svc *Service) Start(s service.Service) (err error) {
	_ = svc.Log().Info("starting")

	if err = svc.writeServerCfg(); err != nil {
		return err
	}

	var dir string
	if dir, err = xonoticDir(); err != nil {
		return err
	}
	_ = os.Setenv("PATH", os.Getenv("PATH")+":"+dir)
	_ = os.Setenv("LD_LIBRARY_PATH", filepath.Join(dir, "d0_blind_id", ".libs"))

	if err = svc.startNats(); err != nil {
		return err
	}

	// TODO: remove this
	svc.monitor, err = svc.natsConn.Subscribe("xonotic.>", func(msg *nats.Msg) {
		_ = svc.Log().Info("NATS ", msg.Subject, " ", strings.TrimSpace(string(msg.Data)))
	})

	//cmd := exec.Command("xonotic-local-dedicated", "+serverconfig", "server.cfg")
	cmd := exec.Command("./all", "run", "dedicated")
	cmd.Dir = dir

	svc.runner = runner.New("xonotic", cmd, svc.natsConn)
	if err = svc.runner.Start(); err != nil {
		return err
	}

	svc.api = api.New(svc.Log(), svc.runner, svc.natsConn)

	return svc.startHttp()
}

func (svc *Service) Stop(s service.Service) (err error) {
	_ = svc.Log().Info("stopping")

	if svc.runner != nil {
		if err = svc.runner.Signal(os.Interrupt); err != nil {
			_ = svc.Log().Error("runner.Signal", err)
		}
		if err = svc.runner.Wait(); err != nil {
			_ = svc.Log().Error("runner.Wait", err)
		}
	}

	svc.stopNats()

	if err = svc.stopHttp(); err != nil {
		_ = svc.Log().Error("stopHttp", err)
		os.Exit(1)
	}

	_ = svc.Log().Info("normal exit")

	return nil
}

//go:embed server.cfg
var serverCfg string

func (svc *Service) writeServerCfg() (err error) {

	if os.Getenv("XONOTIC_HOSTNAME") == "" {
		os.Setenv("XONOTIC_HOSTNAME", "local.mlctrez.com")
	}
	if os.Getenv("ADDRESS") == "" {
		os.Setenv("ADDRESS", "127.0.0.1")
	}
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "26000")
	}
	data := map[string]string{}
	for _, attr := range []string{"XONOTIC_HOSTNAME", "ADDRESS", "PORT"} {
		data[attr] = os.Getenv(attr)
	}

	var home string
	home, err = os.UserHomeDir()
	if err != nil {
		return err
	}

	parse, err := template.New("server_cfg").Parse(serverCfg)
	if err != nil {
		return err
	}

	// TODO: handle locations on other operating systems
	configFile := filepath.Join(home, ".xonotic", "data", "server.cfg")
	var create *os.File
	if create, err = os.Create(configFile); err != nil {
		return err
	}

	err = parse.Execute(create, data)
	if err != nil {
		return err
	}

	return create.Close()
}

func (svc *Service) startNats() (err error) {
	if svc.natsServer, err = server.NewServer(svc.natsOptions()); err != nil {
		return err
	}
	go svc.natsServer.Start()

	if !svc.natsServer.ReadyForConnections(5 * time.Second) {
		svc.natsServer.Shutdown()
		return errors.New("unable to start nats server")
	}

	_ = svc.Log().Info("nats url ", svc.natsServer.ClientURL())

	if svc.natsConn, err = nats.Connect("", nats.InProcessServer(svc.natsServer)); err != nil {
		svc.natsServer.Shutdown()
		return err
	}

	return nil
}

func (svc *Service) stopNats() {
	if svc.monitor != nil {
		if err := svc.monitor.Unsubscribe(); err != nil {
			_ = svc.Log().Error("monitor.Unsubscribe", err)
		}
	}

	if svc.natsConn != nil {
		svc.natsConn.Close()
		svc.natsConn = nil
	}

	if svc.natsServer != nil {
		svc.natsServer.Shutdown()
	}
}

func (svc *Service) startHttp() error {
	listen, err := net.Listen("tcp", ":26001")
	if err != nil {
		return err
	}

	svc.server = &http.Server{Handler: svc.api.Handler()}

	go func() {
		serverErr := svc.server.Serve(listen)
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			_ = svc.Log().Error(serverErr)
		}
	}()

	return nil
}

func (svc *Service) stopHttp() error {
	if svc.server == nil {
		return nil
	}

	stopContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return svc.server.Shutdown(stopContext)
}

func (svc *Service) natsOptions() *server.Options {
	return &server.Options{
		NoSigs: true,
		Port:   26002,
		Host:   "0.0.0.0",
	}
}

package main

import (
	"github.com/mlctrez/servicego"
	"github.com/mlctrez/xmgr/service"
)

func main() {
	servicego.Run(&service.Service{})
}

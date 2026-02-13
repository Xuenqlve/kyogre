package main

import (
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/app"
	"github.com/xuenqlve/kyogre/internal/config"
	_ "github.com/xuenqlve/kyogre/pkg/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		return
	}
	server, err := app.NewServer(cfg)
	if err != nil {
		log.Errorf("app new server err:%v", err)
		return
	}
	if err = server.Run(); err != nil {
		log.Errorf("server run err:%v", err)
		return
	}
}

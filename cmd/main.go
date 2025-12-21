package main

import (
	"github.com/xuenqlve/kyogre/internal/app"
	"github.com/xuenqlve/kyogre/internal/config"
	_ "github.com/xuenqlve/kyogre/pkg/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
		return
	}
	server, err := app.NewServer(cfg)
	if err != nil {
		panic(err)
		return
	}
	if err = server.Run(); err != nil {
		panic(err)
		return
	}
}

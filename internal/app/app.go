package app

import (
	"context"

	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/common/log"
	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/data_source"
)

type Server struct {
	pipeline string
	cfg      config.Config
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewServer(cfg config.Config) (*Server, error) {
	ser := &Server{
		pipeline: cfg.Name,
		cfg:      cfg,
	}
	log.Init(ser.cfg.LogLevel, ser.cfg.LogFile)
	if err := ser.Configure(); err != nil {
		return nil, err
	}
	return ser, nil
}

func (s *Server) Configure() (err error) {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	for dsType, dsCfg := range s.cfg.DataSource {
		var dataSource data_source.DataSource
		if dataSource, err = data_source.GetDataSource(data_source.DataSourceType(dsType)); err != nil {
			return errors.Trace(err)
		}
		if err = dataSource.Configure(s.pipeline, dsCfg); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (s *Server) Run() error {
	return nil
}

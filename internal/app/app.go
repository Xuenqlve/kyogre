package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/data_source"
	"github.com/xuenqlve/kyogre/internal/pipeline"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin/pressure"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
	_ "github.com/xuenqlve/kyogre/pkg/generator"
	_ "github.com/xuenqlve/kyogre/pkg/iquery"
	_ "github.com/xuenqlve/kyogre/pkg/metadata"
	_ "github.com/xuenqlve/kyogre/pkg/pressure"
	_ "github.com/xuenqlve/kyogre/pkg/scenario"
)

type Server struct {
	pipeline  string
	cfg       config.Config
	ctx       context.Context
	cancel    context.CancelFunc
	engine    *pipeline.PipelineEngine
	iquery    *iquery.Manager
	scenarios *scenario.Manager
	pressure  *pressure.Manager
	metadata  *metadata.Manager
	generator *generator.Manager
	httpSrv   *http.Server
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

	if len(s.cfg.IQuery) > 0 {
		if s.iquery, err = iquery.IQueryManager(s.pipeline, s.cfg.IQuery); err != nil {
			return errors.Trace(err)
		}
	}
	if err = s.initMetadata(); err != nil {
		return errors.Trace(err)
	}

	if err = s.initGenerator(); err != nil {
		return errors.Trace(err)
	}

	if s.scenarios, err = scenario.ManagerScenario(s.pipeline, s.cfg.Scenario); err != nil {
		return errors.Trace(err)
	}

	if s.pressure, err = pressure.ManagerPressure(s.pipeline, s.cfg.Pressure); err != nil {
		return errors.Trace(err)
	}

	s.engine = pipeline.NewEngine(s.pipeline)
	//if err = s.engine.Build(s.cfg.Pipelines); err != nil {
	//	return errors.Trace(err)
	//}
	return nil
}

func (s *Server) initMetadata() error {
	s.metadata = metadata.NewManager()
	return s.metadata.Configure(s.pipeline, s.cfg.Metadata)
}

func (s *Server) initGenerator() error {
	s.generator = generator.NewManager()
	if err := s.generator.Configure(s.pipeline, s.cfg.Generator, s.metadata); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *Server) initScenario() error {
	s.scenarios = scenario.NewManager()
	if err := s.scenarios.Configure(s.pipeline, s.cfg.Scenario, s.generator, s.iquery); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *Server) BuildPipeline() error {
	for _, cfg := range s.cfg.Pipelines {

		scenario, err := s.scenarios.GetScenario(cfg.Scenario)
		if err != nil {
			return errors.Trace(err)
		}

		pipeline.NewPipeline(s.pipeline)
	}
}

func (s *Server) Run() error {
	if s.engine == nil {
		return fmt.Errorf("pipeline engine not initialized")
	}
	if err := s.engine.StartAll(s.ctx); err != nil {
		return err
	}

	apiErr := make(chan error, 1)
	go func() {
		apiErr <- s.startAPI(s.ctx)
	}()
	select {
	case <-s.ctx.Done():
		s.engine.StopAll()
		return <-apiErr
	case err := <-apiErr:
		s.engine.StopAll()
		return err
	}
}

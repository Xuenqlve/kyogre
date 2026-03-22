package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/data_source"
	"github.com/xuenqlve/kyogre/internal/event"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin/pressure"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
	_ "github.com/xuenqlve/kyogre/pkg/generator"
	_ "github.com/xuenqlve/kyogre/pkg/lookup"
	_ "github.com/xuenqlve/kyogre/pkg/metadata"
	_ "github.com/xuenqlve/kyogre/pkg/pressure"
	_ "github.com/xuenqlve/kyogre/pkg/scenario"
	mockScenario "github.com/xuenqlve/kyogre/pkg/scenario/mock"
)

type Server struct {
	pipeline       string
	exitOnComplete bool
	cfg            config.Config
	ctx            context.Context
	cancel         context.CancelFunc
	engine         *PipelineEngine
	iquery         *iquery.Manager
	scenarios      *scenario.Manager
	pressure       *pressure.Manager
	metadata       *metadata.Manager
	generator      *generator.Manager
	httpSrv        *http.Server
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
	event.EventAdmin.Init()
	s.registerEventHandlers()
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
	s.metadata = metadata.NewManager()
	if err = s.metadata.Configure(s.pipeline, s.cfg.Metadata); err != nil {
		return errors.Trace(err)
	}
	// metadata 需要在 scenario/lookup 编排前初始化，确保 SchemaStore 可用。
	for key := range s.cfg.Metadata {
		md, mdErr := s.metadata.GetMetadata(key)
		if mdErr != nil {
			return errors.Trace(mdErr)
		}
		if mdErr = md.Initialize(s.ctx); mdErr != nil {
			return errors.Trace(mdErr)
		}
	}

	s.generator = generator.NewManager()
	if err = s.generator.Configure(s.pipeline, s.cfg.Generator); err != nil {
		return errors.Trace(err)
	}

	s.scenarios = scenario.NewManager()
	if err = s.scenarios.Configure(s.pipeline, s.cfg.Scenario, s.generator, s.iquery, s.metadata); err != nil {
		return errors.Trace(err)
	}
	s.pressure = pressure.NewManager()
	if err = s.pressure.Configure(s.pipeline, s.cfg.Pressure); err != nil {
		return errors.Trace(err)
	}
	s.engine = NewEngine(s.pipeline)
	for _, spec := range s.cfg.Pipelines {
		s.engine.RegisterPipeline(spec.Name, NewPipeline(spec, s.scenarios, s.pressure))
	}
	s.exitOnComplete = s.shouldExitOnComplete()
	return nil
}

func (s *Server) registerEventHandlers() {
	handleFatal := func(e event.Event) {
		if s.cancel != nil {
			s.cancel()
		}
	}
	event.EventAdmin.Register(event.PipelineError, handleFatal)
	event.EventAdmin.Register(event.WorkerError, handleFatal)
	event.EventAdmin.Register(event.GoroutinePanic, handleFatal)
	event.EventAdmin.Register(event.ServerShutdown, handleFatal)
}

func (s *Server) Run() error {
	if s.engine == nil {
		return fmt.Errorf("pipeline engine not initialized")
	}
	if err := s.engine.StartAll(s.ctx); err != nil {
		return err
	}
	if s.exitOnComplete {
		go func() {
			s.engine.WaitAllStopped(s.ctx)
			// Context may already be canceled if shutdown was requested elsewhere.
			if s.ctx.Err() != nil {
				return
			}
			log.Infof("[%s] mock pipelines finished, shutting down server", s.pipeline)
			log.Infof("pipeline status snapshot: %+v", s.engine.ListStatus())
			s.cancel()
		}()
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

func (s *Server) shouldExitOnComplete() bool {
	if len(s.cfg.Pipelines) == 0 {
		return false
	}
	mockType := string(mockScenario.ScenarioType)
	for _, spec := range s.cfg.Pipelines {
		scCfg, ok := s.cfg.Scenario[spec.Scenario]
		if !ok {
			return false
		}
		if scCfg.Type != mockType && scCfg.Type != "mock" {
			return false
		}
	}
	return true
}

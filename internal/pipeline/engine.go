package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin/pressure"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
)

// PipelineState 描述当前管线状态
type PipelineState string

const (
	StateStopped  PipelineState = "stopped"
	StateRunning  PipelineState = "running"
	StateStopping PipelineState = "stopping"
)

// PipelineStatus 暴露给上层的状态视图
type PipelineStatus struct {
	Name      string        `json:"name"`
	State     PipelineState `json:"state"`
	LastError string        `json:"last_error,omitempty"`
}

// PipelineEngine 管理多条压测管线的生命周期
type PipelineEngine struct {
	pipelineName string
	mu           sync.RWMutex
	pipelines    map[string]*Pipeline
}

func NewEngine(pipelineName string) *PipelineEngine {
	return &PipelineEngine{pipelineName: pipelineName, pipelines: make(map[string]*Pipeline)}
}

// Build 根据配置创建所有 Pipeline 模板
func (e *PipelineEngine) Build(cfg config.Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(cfg.Pipelines) == 0 {
		return fmt.Errorf("no pipelines configured")
	}
	for _, spec := range cfg.Pipelines {
		if spec.Name == "" {
			return fmt.Errorf("pipeline name is required")
		}
		e.pipelines[spec.Name] = newPipeline(e.pipelineName, spec)
	}
	return nil
}

// StartAll 启动所有已创建的管线
func (e *PipelineEngine) StartAll(ctx context.Context) error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for name, p := range e.pipelines {
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("start pipeline %s failed: %w", name, err)
		}
	}
	return nil
}

// StartPipeline 启动指定管线
func (e *PipelineEngine) StartPipeline(ctx context.Context, name string) error {
	p, err := e.getPipeline(name)
	if err != nil {
		return err
	}
	return p.Start(ctx)
}

// StopPipeline 停止指定管线
func (e *PipelineEngine) StopPipeline(name string) error {
	p, err := e.getPipeline(name)
	if err != nil {
		return err
	}
	return p.Stop()
}

// StopAll 停止所有管线
func (e *PipelineEngine) StopAll() {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, p := range e.pipelines {
		_ = p.Stop()
	}
}

// Status 返回单条管线状态
func (e *PipelineEngine) Status(name string) (PipelineStatus, error) {
	p, err := e.getPipeline(name)
	if err != nil {
		return PipelineStatus{}, err
	}
	return p.Status(), nil
}

// ListStatus 返回所有管线状态
func (e *PipelineEngine) ListStatus() []PipelineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	statuses := make([]PipelineStatus, 0, len(e.pipelines))
	for _, p := range e.pipelines {
		statuses = append(statuses, p.Status())
	}
	return statuses
}

func (e *PipelineEngine) getPipeline(name string) (*Pipeline, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline %s not found", name)
	}
	return p, nil
}

// Pipeline 表示一条可启动/停止的链路
type Pipeline struct {
	name         string
	pipelineName string
	spec         config.PipelineSpec

	mu         sync.Mutex
	state      PipelineState
	lastErr    error
	cancel     context.CancelFunc
	done       chan struct{}
	metadata   map[string]metadata.Metadata
	generators map[string]generator.Generator
	scenario   scenario.Scenario
	pressure   pressure.Pressure
	point      message.Point
}

func newPipeline(pipelineName string, spec config.PipelineSpec) *Pipeline {
	return &Pipeline{
		name:         spec.Name,
		pipelineName: pipelineName,
		spec:         spec,
		state:        StateStopped,
	}
}

func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == StateRunning {
		return fmt.Errorf("pipeline %s already running", p.name)
	}
	if err := p.prepareModules(); err != nil {
		p.cleanupModules()
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.state = StateRunning
	go p.runLoop(runCtx)
	return nil
}

func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if p.state != StateRunning && p.state != StateStopping {
		p.mu.Unlock()
		return nil
	}
	if p.state == StateStopping {
		done := p.done
		p.mu.Unlock()
		if done != nil {
			<-done
		}
		return nil
	}
	p.state = StateStopping
	cancel := p.cancel
	done := p.done
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	return nil
}

func (p *Pipeline) Status() PipelineStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	status := PipelineStatus{Name: p.name, State: p.state}
	if p.lastErr != nil {
		status.LastError = p.lastErr.Error()
	}
	return status
}

func (p *Pipeline) prepareModules() error {
	if err := p.buildMetadata(); err != nil {
		return err
	}
	if err := p.buildGenerators(); err != nil {
		return err
	}
	if err := p.buildScenario(); err != nil {
		return err
	}
	if err := p.buildPressure(); err != nil {
		return err
	}
	return nil
}

func (p *Pipeline) runLoop(ctx context.Context) {
	err := p.execute(ctx)
	p.mu.Lock()
	p.lastErr = err
	p.state = StateStopped
	p.cancel = nil
	if p.done != nil {
		close(p.done)
	}
	p.done = nil
	p.cleanupModules()
	p.mu.Unlock()
}

func (p *Pipeline) execute(ctx context.Context) error {
	defer p.shutdown()
	if err := p.initializeMetadata(ctx); err != nil {
		return err
	}
	if err := p.pressure.Start(ctx); err != nil {
		return err
	}
	if err := p.scenario.Preparation(ctx); err != nil {
		return err
	}
	p.point = make(message.Point, 1024)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range p.point.OutPoint() {
			if msg == nil {
				if p.cancel != nil {
					p.cancel()
				}
				return
			}
			p.pressure.Execute(msg)
		}
	}()
	go func() {
		<-ctx.Done()
		if p.point != nil {
			p.point.Close()
		}
	}()
	if err := p.scenario.Start(p.point.InPoint()); err != nil {
		return err
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

func (p *Pipeline) buildMetadata() error {
	p.metadata = make(map[string]metadata.Metadata)
	for key, mold := range p.spec.Metadata {
		md, err := metadata.GetMetadata(metadata.MetadataType(mold.Type))
		if err != nil {
			return err
		}
		if err = md.Configure(p.pipelineName, mold.Config); err != nil {
			return err
		}
		p.metadata[key] = md
	}
	return nil
}

func (p *Pipeline) buildGenerators() error {
	p.generators = make(map[string]generator.Generator)
	for _, spec := range p.spec.Generators {
		gen, err := generator.GetGenerator(generator.Type(spec.Type))
		if err != nil {
			return err
		}
		if err = gen.Configure(p.pipelineName, spec.Config); err != nil {
			return err
		}
		if spec.MetadataRef != "" {
			md, ok := p.metadata[spec.MetadataRef]
			if !ok {
				return fmt.Errorf("generator %s metadata-ref %s not found", spec.Name, spec.MetadataRef)
			}
			gen.RegisterMetadata(md)
		}
		p.generators[spec.Name] = gen
	}
	return nil
}

func (p *Pipeline) buildScenario() error {
	if p.spec.Scenario.Type == "" {
		return fmt.Errorf("scenario type required")
	}
	sc, err := scenario.GetScenario(scenario.Type(p.spec.Scenario.Type))
	if err != nil {
		return err
	}
	if err = sc.Configure(p.pipelineName, p.spec.Scenario.Config); err != nil {
		return err
	}
	if ref := p.spec.Scenario.MetadataRef; ref != "" {
		md, ok := p.metadata[ref]
		if !ok {
			return fmt.Errorf("scenario metadata-ref %s not found", ref)
		}
		sc.RegisterMetadata(md)
	}
	for _, name := range p.spec.Scenario.Generators {
		gen, ok := p.generators[name]
		if !ok {
			return fmt.Errorf("scenario generator %s not found", name)
		}
		sc.RegisterGenerator(gen)
	}
	p.scenario = sc
	return nil
}

func (p *Pipeline) buildPressure() error {
	if p.spec.Pressure.Type == "" {
		return fmt.Errorf("pressure type required")
	}
	pr, err := pressure.GetPressure(pressure.PressureType(p.spec.Pressure.Type))
	if err != nil {
		return err
	}
	if err = pr.Configure(p.pipelineName, p.spec.Pressure.Config); err != nil {
		return err
	}
	p.pressure = pr
	return nil
}

func (p *Pipeline) initializeMetadata(ctx context.Context) error {
	for key, md := range p.metadata {
		if err := md.Initialize(ctx); err != nil {
			return fmt.Errorf("metadata %s initialize failed: %w", key, err)
		}
	}
	return nil
}

func (p *Pipeline) shutdown() {
	if p.scenario != nil {
		_ = p.scenario.Close()
	}
	if p.pressure != nil {
		_ = p.pressure.Close()
	}
	for _, md := range p.metadata {
		_ = md.Close()
	}
	for _, gen := range p.generators {
		gen.Close()
	}
}

func (p *Pipeline) cleanupModules() {
	p.metadata = nil
	p.generators = nil
	p.scenario = nil
	p.pressure = nil
	p.point = nil
}

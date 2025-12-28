package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/message"
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
func (e *PipelineEngine) Build(cfg []config.PipelineSpec) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(cfg) == 0 {
		return fmt.Errorf("no pipelines configured")
	}
	for _, spec := range cfg {
		if spec.Name == "" {
			return fmt.Errorf("pipeline name is required")
		}

		//e.pipelines[spec.Name] = newPipeline(e.pipelineName, spec)
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

	mu       sync.Mutex
	state    PipelineState
	lastErr  error
	cancel   context.CancelFunc
	done     chan struct{}
	scenario scenario.Scenario
	pressure pressure.Pressure
	point    message.Point
}

func NewPipeline(pipelineName string, scenario scenario.Scenario, pressure pressure.Pressure) *Pipeline {
	return &Pipeline{
		pipelineName: pipelineName,
		scenario:     scenario,
		pressure:     pressure,
	}
}

func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan struct{})
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

func (p *Pipeline) initializeMetadata(ctx context.Context) error {
	//for key, md := range p.metadata {
	//	if err := md.Initialize(ctx); err != nil {
	//		return fmt.Errorf("metadata %s initialize failed: %w", key, err)
	//	}
	//}
	return nil
}

func (p *Pipeline) shutdown() {
	if p.scenario != nil {
		_ = p.scenario.Close()
	}
	if p.pressure != nil {
		_ = p.pressure.Close()
	}
}

func (p *Pipeline) cleanupModules() {
	p.point = nil
	p.pressure = nil
	p.scenario = nil
}

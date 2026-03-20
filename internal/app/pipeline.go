package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/event"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/pressure"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
	"runtime/debug"
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

func (e *PipelineEngine) RegisterPipeline(name string, pipeline *Pipeline) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pipelines[name] = pipeline
	return
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

// WaitAllStopped blocks until all tracked pipelines report completion or ctx is canceled.
func (e *PipelineEngine) WaitAllStopped(ctx context.Context) {
	e.mu.RLock()
	doneChans := make([]<-chan struct{}, 0, len(e.pipelines))
	for _, p := range e.pipelines {
		doneChans = append(doneChans, p.Done())
	}
	e.mu.RUnlock()

	for _, ch := range doneChans {
		if ch == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ch:
		}
	}
}

// Pipeline 表示一条可启动/停止的链路
type Pipeline struct {
	name        string
	spec        config.PipelineSpec
	scenarioMgr *scenario.Manager
	pressureMgr *pressure.Manager
	mu          sync.Mutex
	state       PipelineState
	lastErr     error
	cancel      context.CancelFunc
	done        chan struct{}
	director    *scenario.Director
	pressure    *pressure.Controller
	point       message.Point
}

func NewPipeline(spec config.PipelineSpec, scenarioMgr *scenario.Manager, pressureMgr *pressure.Manager) *Pipeline {
	return &Pipeline{
		name:        spec.Name,
		spec:        spec,
		scenarioMgr: scenarioMgr,
		pressureMgr: pressureMgr,
		state:       StateStopped,
	}
}

func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.state == StateRunning || p.state == StateStopping {
		state := p.state
		p.mu.Unlock()
		return fmt.Errorf("pipeline %s is %s", p.name, state)
	}
	if err := p.ensureModulesLocked(); err != nil {
		p.mu.Unlock()
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	ready := make(chan error, 1)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.state = StateRunning
	p.lastErr = nil
	p.monitorScenarioCompletion(runCtx)
	go p.runLoop(runCtx, ready)
	p.mu.Unlock()
	if err := <-ready; err != nil {
		p.waitUntilStopped()
		return err
	}
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

// Done exposes an observable channel that closes when the pipeline stops.
func (p *Pipeline) Done() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done
}

func (p *Pipeline) ensureModulesLocked() error {
	if p.director == nil {
		if p.scenarioMgr == nil {
			return fmt.Errorf("scenario manager not configured for pipeline %s", p.name)
		}
		director, err := p.scenarioMgr.GetDirector(p.spec.Scenario)
		if err != nil {
			return err
		}
		p.director = director
	}

	if p.pressure == nil {
		if p.pressureMgr == nil {
			return fmt.Errorf("pressure manager not configured for pipeline %s", p.name)
		}
		ctrl, err := p.pressureMgr.GetPressureController(p.spec.Pressure)
		if err != nil {
			return err
		}
		p.pressure = ctrl
	}
	return nil
}

func (p *Pipeline) runLoop(ctx context.Context, ready chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			event.EventAdmin.Upload(event.Event{
				Type: event.GoroutinePanic,
				Key:  p.name,
				Value: map[string]any{
					"component": "pipeline",
					"panic":     r,
					"stack":     string(debug.Stack()),
				},
			})
		}
	}()
	err := p.execute(ctx, &ready)
	signalReady(&ready, err)
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

func (p *Pipeline) execute(ctx context.Context, ready *chan<- error) error {
	defer p.shutdown()
	p.point = make(message.Point, 1024)
	if err := p.pressure.Start(ctx, p.point.OutPoint()); err != nil {
		p.uploadError("pressure", err)
		signalReady(ready, err)
		return err
	}
	if err := p.director.Preparation(ctx); err != nil {
		p.uploadError("scenario-director", err)
		signalReady(ready, err)
		return err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if p.point != nil {
			p.point.Close()
		}
	}()
	if err := p.director.Start(ctx, p.point.InPoint()); err != nil {
		p.uploadError("scenario-director", err)
		signalReady(ready, err)
		return err
	}
	signalReady(ready, nil)
	<-ctx.Done()
	wg.Wait()
	if err := p.director.Err(); err != nil {
		p.uploadError("scenario-director", err)
		return err
	}
	return nil
}

func (p *Pipeline) shutdown() {
	if p.director != nil {
		_ = p.director.Close(true)
	}
	if p.pressure != nil {
		_ = p.pressure.Close()
	}
}

func (p *Pipeline) cleanupModules() {
	p.point = nil
	p.pressure = nil
	p.director = nil
}

func (p *Pipeline) waitUntilStopped() {
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()
	if done != nil {
		<-done
	}
}

func signalReady(ch *chan<- error, err error) {
	if ch == nil || *ch == nil {
		return
	}
	(*ch) <- err
	close(*ch)
	*ch = nil
}

func (p *Pipeline) monitorScenarioCompletion(ctx context.Context) {
	done := p.director.Done()
	if done == nil {
		return
	}
	go func() {
		select {
		case <-done:
			if summary := p.director.Summary(); summary != nil {
				log.Infof("[%s] scenario completed summary=%v", p.name, summary)
			} else {
				log.Infof("[%s] scenario completed", p.name)
			}
			if err := p.Stop(); err != nil {
				log.Warnf("pipeline %s failed to stop on completion: %v", p.name, err)
				p.uploadError("pipeline-stop", err)
			}
		case <-ctx.Done():
		}
	}()
}

func (p *Pipeline) uploadError(component string, err error) {
	if err == nil {
		return
	}
	event.EventAdmin.Upload(event.Event{
		Type: event.PipelineError,
		Key:  p.name,
		Value: map[string]any{
			"component": component,
			"err":       err.Error(),
		},
	})
}

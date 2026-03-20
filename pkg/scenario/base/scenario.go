package base

import (
	"context"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	pluginGenerator "github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	pluginScenario "github.com/xuenqlve/kyogre/internal/plugin/scenario"
)

const ScenarioType pluginScenario.Type = "base"

type Scenario struct {
	pipeline string
	cfg      Config
	builder  ContextBuilder
	summary  map[string]any
}

func init() {
	pluginScenario.RegisterScenario(ScenarioType, &Scenario{}, false)
}

func (s *Scenario) Configure(pipeline string, data map[string]any) error {
	s.pipeline = pipeline
	s.summary = nil

	if err := mapstructure.Decode(data, &s.cfg); err != nil {
		return errors.Trace(err)
	}
	if err := s.cfg.Normalize(); err != nil {
		return errors.Trace(err)
	}
	builder, err := GetBuilder(BuilderType(s.cfg.Builder))
	if err != nil {
		return err
	}
	if err = builder.Configure(pipeline, s.cfg.BuilderConfig); err != nil {
		return err
	}
	s.builder = builder
	return nil
}

func (s *Scenario) Start(ctx context.Context, md metadata.Metadata, seq *iquery.Sequencer, ctxChan chan<- pluginGenerator.GenerationContext) {
	startAt := time.Now()
	sentCount := 0
	if ctxChan == nil {
		s.summary = s.buildSummary(sentCount, startAt)
		return
	}
	if s.builder == nil {
		log.Errorf("[%s] base scenario builder is nil", s.pipeline)
		s.summary = s.buildSummary(sentCount, startAt)
		return
	}
	factory := NewSelectorFactory(nil)
	targets, err := s.builder.LoadTargets(md, s.cfg.TargetSelector.Schemas)
	if err != nil {
		log.Errorf("[%s] base scenario load targets failed: %v", s.pipeline, err)
		s.summary = s.buildSummary(sentCount, startAt)
		return
	}
	targetSelector, err := factory.Target(s.cfg.TargetSelector, targets)
	if err != nil {
		log.Errorf("[%s] base scenario target selector failed: %v", s.pipeline, err)
		s.summary = s.buildSummary(sentCount, startAt)
		return
	}
	operationSelector, err := factory.Operation(s.cfg.OperationSelector)
	if err != nil {
		log.Errorf("[%s] base scenario operation selector failed: %v", s.pipeline, err)
		s.summary = s.buildSummary(sentCount, startAt)
		return
	}
	rowCountSelector, err := factory.Int(s.cfg.RowCountSelector)
	if err != nil {
		log.Errorf("[%s] base scenario row count selector failed: %v", s.pipeline, err)
		s.summary = s.buildSummary(sentCount, startAt)
		return
	}
	var txSelector selectorPick[int]
	if s.cfg.Mode == ModeTransaction {
		txSelector, err = factory.Int(s.cfg.TransactionSizeSelector)
		if err != nil {
			log.Errorf("[%s] base scenario transaction selector failed: %v", s.pipeline, err)
			s.summary = s.buildSummary(sentCount, startAt)
			return
		}
	}
	binder := NewLookupBinder(s.cfg.Lookup, seq)
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	for i := 0; i < s.cfg.MessageCount; i++ {
		select {
		case <-ctx.Done():
			s.summary = s.buildSummary(sentCount, startAt)
			return
		default:
		}
		plan, err := s.nextPlan(ctx, targetSelector, operationSelector, rowCountSelector, txSelector, binder)
		if err != nil {
			log.Errorf("[%s] base scenario build plan failed: %v", s.pipeline, err)
			s.summary = s.buildSummary(sentCount, startAt)
			return
		}
		genCtx, err := s.builder.Build(plan)
		if err != nil {
			log.Errorf("[%s] base scenario build generation context failed: %v", s.pipeline, err)
			s.summary = s.buildSummary(sentCount, startAt)
			return
		}
		if genCtx == nil {
			continue
		}
		select {
		case <-ctx.Done():
			s.summary = s.buildSummary(sentCount, startAt)
			return
		case ctxChan <- genCtx:
		}
		sentCount++
		if interval <= 0 {
			continue
		}
		select {
		case <-ctx.Done():
			s.summary = s.buildSummary(sentCount, startAt)
			return
		case <-time.After(interval):
		}
	}
	s.summary = s.buildSummary(sentCount, startAt)
}

func (s *Scenario) Summary() map[string]any {
	if s.summary == nil {
		return nil
	}
	out := make(map[string]any, len(s.summary))
	for k, v := range s.summary {
		out[k] = v
	}
	return out
}

type selectorPick[T any] interface {
	Pick() (T, error)
}

func (s *Scenario) nextPlan(
	ctx context.Context,
	targetSelector selectorPick[Target],
	operationSelector selectorPick[string],
	rowCountSelector selectorPick[int],
	txSelector selectorPick[int],
	binder *LookupBinder,
) (Plan, error) {
	target, err := targetSelector.Pick()
	if err != nil {
		return Plan{}, err
	}
	operation, err := operationSelector.Pick()
	if err != nil {
		return Plan{}, err
	}
	rowCount, err := rowCountSelector.Pick()
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Builder:        s.cfg.Builder,
		Mode:           s.cfg.Mode,
		Operation:      operation,
		Target:         target.Clone(),
		RowsPerMessage: rowCount,
		Columns:        append([]string(nil), s.cfg.Columns...),
		Hint:           s.cfg.Hint,
		WriteType:      s.cfg.WriteType,
	}
	if s.cfg.Mode == ModeTransaction && txSelector != nil {
		txSize, err := txSelector.Pick()
		if err != nil {
			return Plan{}, err
		}
		plan.TransactionSize = txSize
	}
	if binder != nil {
		providers, err := binder.Providers(ctx, plan)
		if err != nil {
			return Plan{}, err
		}
		plan.Providers = providers
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (s *Scenario) buildSummary(sentCount int, startAt time.Time) map[string]any {
	return map[string]any{
		"pipeline":      s.pipeline,
		"builder":       s.cfg.Builder,
		"mode":          s.cfg.Mode,
		"messages":      sentCount,
		"duration":      time.Since(startAt).String(),
		"interval_ms":   s.cfg.IntervalMS,
		"message_count": s.cfg.MessageCount,
	}
}

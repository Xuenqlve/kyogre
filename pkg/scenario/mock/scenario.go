package mock

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
	"github.com/xuenqlve/kyogre/pkg/generator_context"
	"github.com/xuenqlve/kyogre/pkg/message"
)

const ScenarioType scenario.Type = "mock"

type Config struct {
	MessageCount int          `mapstructure:"message-count"`
	IntervalMS   int          `mapstructure:"interval-ms"`
	WorkerCount  int          `mapstructure:"worker-count"`
	IQuery       IQueryConfig `mapstructure:"lookup"`
}

type IQueryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
}

type Scenario struct {
	cfg          Config
	pipeline     string
	startAt      time.Time
	sentCount    atomic.Int64
	completeOnce sync.Once
	specCursor   atomic.Uint64
	summary      map[string]any
}

func init() {
	scenario.RegisterScenario(ScenarioType, &Scenario{}, false)
}

func (s *Scenario) Configure(pipeline string, data map[string]any) error {
	s.pipeline = pipeline
	s.sentCount.Store(0)
	s.completeOnce = sync.Once{}
	s.summary = nil

	if err := mapstructure.Decode(data, &s.cfg); err != nil {
		return errors.Trace(err)
	}
	if s.cfg.MessageCount <= 0 {
		s.cfg.MessageCount = 5
	}
	if s.cfg.IntervalMS <= 0 {
		s.cfg.IntervalMS = 100
	}
	return nil
}

func (s *Scenario) Start(ctx context.Context, metadata metadata.Metadata, sequencer *iquery.Sequencer, ctxChan chan<- generator.GenerationContext) {
	if ctxChan == nil {
		return
	}
	s.startAt = time.Now()
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	specs := s.buildSpecs(metadata)
	snapshot := s.buildStrategySnapshot()
	for i := 0; i < s.cfg.MessageCount; i++ {
		select {
		case <-ctx.Done():
			s.notifyComplete()
			return
		case ctxChan <- s.buildContext(sequencer, specs, snapshot):
		}
		s.sentCount.Add(1)
		if interval > 0 {
			select {
			case <-ctx.Done():
				s.notifyComplete()
				return
			case <-time.After(interval):
			}
		}
	}
	s.notifyComplete()
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

func (s *Scenario) Close() error {
	return nil
}

type specInfo struct {
	spec          iquery.SequenceSpec
	meta          metadata.Metadata
	iqueryEnabled bool
}

func (s *Scenario) buildContext(seq *iquery.Sequencer, specs []specInfo, snapshot *generator.StrategySnapshot) generator.GenerationContext {
	value := map[string]any{}
	var provider iquery.Provider

	info, ok := s.nextSpec(specs)
	if ok {
		s.enrichValueWithMetadata(value, info)
		if seq != nil && info.iqueryEnabled {
			provider = s.reserveProvider(seq, info)
			if provider != nil {
				s.mergeProviderValue(value, provider)
			}
		}
	}

	return generator_context.NewMockContext(
		value,
		generator_context.WithIQueryProvider(info.spec.Schema.UniqueID(), provider),
		generator_context.WithStrategy(snapshot),
	)
}

func (s *Scenario) buildSpecs(md metadata.Metadata) []specInfo {
	specs := make([]specInfo, 0)
	for _, schemaKey := range md.SchemaKeys() {
		fields, err := md.SchemaPrimaryField(schemaKey)
		if err != nil {
			log.Warnf("mock scenario schema primary field failure schema=%s: %v", schemaKey.UniqueID(), err)
			continue
		}
		specs = append(specs, specInfo{
			spec: iquery.SequenceSpec{
				Schema: schemaKey,
				Fields: fields,
			},
			meta:          md,
			iqueryEnabled: md.IQueryEnabled(),
		})
	}
	return specs
}

func (s *Scenario) buildStrategySnapshot() *generator.StrategySnapshot {
	return generator.NewSnapshot(
		generator.WithCountFixed(3),
		generator.WithInclude("value", "seq"),
		generator.WithExclude("schema"),
		generator.WithValue("value", generator.ValueSpec{Mode: "fixed", Value: "strategy-match"}),
		generator.WithTemplate("mock_template", map[string]any{"version": "v1"}),
		generator.WithCustom(map[string]any{"suffix": "from-strategy"}),
	)
}

func (s *Scenario) nextSpec(specs []specInfo) (specInfo, bool) {
	if len(specs) == 0 {
		return specInfo{}, false
	}
	idx := int(s.specCursor.Add(1)-1) % len(specs)
	return specs[idx], true
}

func (s *Scenario) reserveProvider(seq *iquery.Sequencer, info specInfo) iquery.Provider {
	ctx := context.Background()
	provider, err := seq.ReserveInsert(ctx, info.spec, 1)
	if err != nil {
		log.Warnf("mock scenario sequencer reserve failure schema=%s: %v", info.spec.Schema.UniqueID(), err)
		return nil
	}
	if provider == nil {
		log.Warnf("mock scenario sequencer provider is nil schema=%s", info.spec.Schema.UniqueID())
		return nil
	}
	return provider
}

func (s *Scenario) mergeProviderValue(value map[string]any, provider iquery.Provider) {
	if provider == nil {
		return
	}
	rows := provider.Rows()
	if len(rows) == 0 {
		return
	}
	for k, v := range rows[0] {
		value[k] = v
	}
}

func (s *Scenario) enrichValueWithMetadata(value map[string]any, info specInfo) {
	if info.meta == nil {
		return
	}
	store := info.meta.SchemaStore()
	if store == nil {
		return
	}
	schema, err := store.GetSchema(info.spec.Schema)
	if err != nil {
		log.Warnf("mock scenario schema load failure schema=%s: %v", info.spec.Schema.UniqueID(), err)
		return
	}
	switch typed := schema.(type) {
	case map[string]any:
		for k, v := range typed {
			if _, ok := value[k]; ok {
				continue
			}
			value[k] = v
		}
	case *mysql_schema.Table:
		for _, column := range typed.Columns {
			if _, ok := value[column.Name]; ok {
				continue
			}
			if !column.DefaultVal.IsNull && column.DefaultVal.ValueString != "" {
				value[column.Name] = column.DefaultVal.ValueString
			} else {
				value[column.Name] = nil
			}
		}
	default:
		value["schema"] = schema
	}
}

func (s *Scenario) notifyComplete() {
	s.completeOnce.Do(func() {
		total := s.sentCount.Load()
		duration := time.Since(s.startAt)
		s.summary = map[string]any{
			"pipeline":     s.pipeline,
			"messages":     total,
			"duration":     duration.String(),
			"interval_ms":  s.cfg.IntervalMS,
			"message_type": message.MockType,
		}
		log.Infof("[%s] mock scenario completed summary=%v", s.pipeline, s.summary)
	})
}

package mysql

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	pluginScenario "github.com/xuenqlve/kyogre/internal/plugin/scenario"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

const (
	ScenarioType pluginScenario.Type = "mysql"
	modeRow      string              = "row"
	modeTx       string              = "transaction"
)

type Config struct {
	Mode            string   `mapstructure:"mode"`
	MessageCount    int      `mapstructure:"message-count"`
	IntervalMS      int      `mapstructure:"interval-ms"`
	WorkerCount     int      `mapstructure:"worker-count"`
	Operation       string   `mapstructure:"operation"`
	Schemas         []string `mapstructure:"schemas"`
	RowsPerMessage  int      `mapstructure:"rows-per-message"`
	TransactionSize int      `mapstructure:"transaction-size"`
	Columns         []string `mapstructure:"columns"`
	Hint            string   `mapstructure:"hint"`
	WriteType       string   `mapstructure:"write-type"`
}

func (c *Config) Normalize() error {
	if c.Mode == "" {
		c.Mode = modeRow
	}
	switch c.Mode {
	case modeRow, modeTx:
	default:
		return fmt.Errorf("unsupported mysql scenario mode: %s", c.Mode)
	}
	if c.MessageCount <= 0 {
		c.MessageCount = 1
	}
	if c.IntervalMS < 0 {
		c.IntervalMS = 0
	}
	if c.Operation == "" {
		c.Operation = "insert"
	}
	if c.RowsPerMessage <= 0 {
		c.RowsPerMessage = 1
	}
	if c.TransactionSize <= 0 {
		c.TransactionSize = 2
	}
	return nil
}

type Scenario struct {
	cfg      Config
	pipeline string
	delegate *base.Scenario
}

func init() {
	pluginScenario.RegisterScenario(ScenarioType, &Scenario{}, false)
}

func (s *Scenario) Configure(pipeline string, data map[string]any) error {
	s.pipeline = pipeline
	s.cfg = Config{}
	s.delegate = &base.Scenario{}

	if err := mapstructure.Decode(data, &s.cfg); err != nil {
		return errors.Trace(err)
	}
	if err := s.cfg.Normalize(); err != nil {
		return errors.Trace(err)
	}
	if err := s.delegate.Configure(pipeline, s.baseConfig()); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *Scenario) Start(ctx context.Context, md metadata.Metadata, seq *iquery.Sequencer, ctxChan chan<- generator.GenerationContext) {
	if s.delegate == nil {
		return
	}
	s.delegate.Start(ctx, md, seq, ctxChan)
}

func (s *Scenario) Summary() map[string]any {
	if s.delegate == nil {
		return nil
	}
	return s.delegate.Summary()
}

func (s *Scenario) RuntimeError() error {
	if s.delegate == nil {
		return nil
	}
	return s.delegate.RuntimeError()
}

func (s *Scenario) baseConfig() map[string]any {
	cfg := map[string]any{
		"builder":       string(BuilderType),
		"message-count": s.cfg.MessageCount,
		"interval-ms":   s.cfg.IntervalMS,
		"mode":          s.cfg.Mode,
		"columns":       append([]string(nil), s.cfg.Columns...),
		"hint":          s.cfg.Hint,
		"write-type":    s.cfg.WriteType,
		"target-selector": map[string]any{
			"strategy": "round-robin",
			"schemas":  append([]string(nil), s.cfg.Schemas...),
		},
		"operation-selector": map[string]any{
			"strategy": "round-robin",
			"values":   []string{s.cfg.Operation},
		},
		"row-count-selector": map[string]any{
			"fixed": s.cfg.RowsPerMessage,
		},
	}
	if s.cfg.Mode == modeTx {
		cfg["transaction-size-selector"] = map[string]any{
			"fixed": s.cfg.TransactionSize,
		}
	}
	return cfg
}

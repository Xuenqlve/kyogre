package mysql

import (
	"context"
	"fmt"
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
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	"github.com/xuenqlve/kyogre/pkg/message"
)

const (
	ScenarioType scenario.Type = "mysql"
	modeRow      string        = "row"
	modeTx       string        = "transaction"
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
		c.Operation = message.Insert
	}
	if c.Operation != message.Insert {
		return fmt.Errorf("mysql scenario only supports insert in current phase, got: %s", c.Operation)
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
	if err := s.cfg.Normalize(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *Scenario) Start(ctx context.Context, md metadata.Metadata, _ *iquery.Sequencer, ctxChan chan<- generator.GenerationContext) {
	if ctxChan == nil {
		return
	}
	s.startAt = time.Now()
	tables := s.loadTables(md)
	if len(tables) == 0 {
		log.Warnf("[%s] mysql scenario has no available tables", s.pipeline)
		s.notifyComplete()
		return
	}
	snapshot := generator.NewSnapshot(generator.WithCountFixed(s.cfg.RowsPerMessage))
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	for i := 0; i < s.cfg.MessageCount; i++ {
		genCtx := s.buildContext(tables, snapshot)
		if genCtx == nil {
			continue
		}
		select {
		case <-ctx.Done():
			s.notifyComplete()
			return
		case ctxChan <- genCtx:
		}
		s.sentCount.Add(1)
		if interval <= 0 {
			continue
		}
		select {
		case <-ctx.Done():
			s.notifyComplete()
			return
		case <-time.After(interval):
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

func (s *Scenario) buildContext(tables []*mysql_schema.Table, snapshot *generator.StrategySnapshot) generator.GenerationContext {
	table, ok := s.nextTable(tables)
	if !ok || table == nil {
		return nil
	}
	if s.cfg.Mode == modeTx {
		rows := make([]genctx.MySQLRowSpec, 0, s.cfg.TransactionSize)
		for i := 0; i < s.cfg.TransactionSize; i++ {
			rows = append(rows, s.buildRowSpec(table))
		}
		return genctx.NewMySQLTransactionContext(rows, genctx.WithStrategy(snapshot))
	}
	return genctx.NewMySQLRowContext(s.buildRowSpec(table), genctx.WithStrategy(snapshot))
}

func (s *Scenario) buildRowSpec(table *mysql_schema.Table) genctx.MySQLRowSpec {
	columns := append([]string(nil), s.cfg.Columns...)
	return genctx.MySQLRowSpec{
		Operation: s.cfg.Operation,
		Hint:      s.cfg.Hint,
		WriteType: s.cfg.WriteType,
		Columns:   columns,
		Schema:    table,
	}
}

func (s *Scenario) nextTable(tables []*mysql_schema.Table) (*mysql_schema.Table, bool) {
	if len(tables) == 0 {
		return nil, false
	}
	idx := int(s.specCursor.Add(1)-1) % len(tables)
	return tables[idx], true
}

func (s *Scenario) loadTables(md metadata.Metadata) []*mysql_schema.Table {
	if md == nil || md.SchemaStore() == nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(s.cfg.Schemas))
	for _, name := range s.cfg.Schemas {
		if name != "" {
			allowed[name] = struct{}{}
		}
	}
	tables := make([]*mysql_schema.Table, 0, len(md.SchemaKeys()))
	for _, key := range md.SchemaKeys() {
		schema, err := md.SchemaStore().GetSchema(key)
		if err != nil {
			log.Warnf("[%s] mysql scenario load schema failure key=%s: %v", s.pipeline, key.UniqueID(), err)
			continue
		}
		table, ok := schema.(*mysql_schema.Table)
		if !ok {
			log.Warnf("[%s] mysql scenario schema type mismatch key=%s: %T", s.pipeline, key.UniqueID(), schema)
			continue
		}
		name := fmt.Sprintf("%s.%s", table.Database, table.Table)
		if len(allowed) > 0 {
			if _, ok = allowed[name]; !ok {
				continue
			}
		}
		tables = append(tables, table)
	}
	return tables
}

func (s *Scenario) notifyComplete() {
	s.completeOnce.Do(func() {
		kind := genctx.MySQLRow
		if s.cfg.Mode == modeTx {
			kind = genctx.MySQLTransaction
		}
		s.summary = map[string]any{
			"pipeline":         s.pipeline,
			"messages":         s.sentCount.Load(),
			"duration":         time.Since(s.startAt).String(),
			"mode":             s.cfg.Mode,
			"operation":        s.cfg.Operation,
			"rows_per_message": s.cfg.RowsPerMessage,
			"transaction_size": s.cfg.TransactionSize,
			"message_type":     kind,
		}
		log.Infof("[%s] mysql scenario completed summary=%v", s.pipeline, s.summary)
	})
}

func (s *Scenario) Close() error {
	return nil
}

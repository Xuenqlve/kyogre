package mysql

import (
	"context"
	"database/sql"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/pkg/data_source"
	"github.com/xuenqlve/timburr/pkg/tool/relational_database/mysql_schema"
	"github.com/xuenqlve/timburr/pkg/tool/schema_store"
)

type PressureConfig struct {
	DataSource        string `mapstructure:"data-source" json:"data-source" yaml:"data-source" toml:"data-source"`
	WorkerCount       int    `mapstructure:"worker-count" json:"worker-count" yaml:"worker-count" toml:"worker-count" `
	WorkerQueueLength int    `mapstructure:"worker-queue-length" json:"worker-queue-length" toml:"worker-queue-length" yaml:"worker-queue-length"`
}

type Pressure struct {
	ctx         context.Context
	pipeline    string
	cfg         PressureConfig
	conn        *sql.DB
	schemaStore schema_store.SchemaStore
	workers     []*Worker
	index       int
}

func (p *Pressure) Configure(pipeline string, data map[string]any) (err error) {
	p.pipeline = pipeline
	if err = mapstructure.Decode(data, &p.cfg); err != nil {
		return errors.Trace(err)
	}
	if p.conn, err = data_source.MySQLConnection(p.cfg.DataSource); err != nil {
		return errors.Trace(err)
	}

	p.schemaStore = schema_store.NewBaseSchemaStore(mysql_schema.NewSchema(p.conn))
	p.workers = make([]*Worker, 0, p.cfg.WorkerCount)
	p.index = 0
	return nil
}

func (p *Pressure) Start(ctx context.Context) error {
	p.ctx = ctx
	for i := 0; i < p.cfg.WorkerCount; i++ {
		worker := NewWorker(p.ctx, p.pipeline, i, p.cfg.WorkerQueueLength, p.conn, p.schemaStore)
		worker.Start()
		p.workers = append(p.workers, worker)
	}
	return nil
}

func (p *Pressure) getWorker() *Worker {
	defer func() {
		p.index++
		if p.index >= p.cfg.WorkerCount {
			p.index = 0
		}
	}()
	return p.workers[p.index]
}

func (p *Pressure) Execute(msg message.Message) {
	worker := p.getWorker()
	worker.SendMessage(msg)
}

func (p *Pressure) Close() error {
	for _, worker := range p.workers {
		if err := worker.Close(); err != nil {
			log.Errorf("mysql pressure worker close error:%v", err)
		}
	}
	return p.conn.Close()
}

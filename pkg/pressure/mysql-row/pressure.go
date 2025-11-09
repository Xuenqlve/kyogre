package mysql_row

import (
	"context"
	"database/sql"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

const (
	MySQLDML plugin.PressureType = "mysql-dml"
)

func init() {
	plugin.RegisterPressure(MySQLDML, &Pressure{}, true)
}

type Pressure struct {
	ctx         context.Context
	pipeline    string
	cfg         Config
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
	if err = p.cfg.Validate(); err != nil {
		return errors.Trace(err)
	}
	if p.conn, err = mysql.Connection(p.cfg.DataSource); err != nil {
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
			log.Errorf("mysql_config-row pressure worker close error:%v", err)
		}
	}
	return p.conn.Close()
}

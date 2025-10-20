package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/timburr/pkg/tool/relational_database/mysql_schema"
	"github.com/xuenqlve/timburr/pkg/tool/schema_store"
)

type Worker struct {
	ctx              context.Context
	wg               sync.WaitGroup
	pipeline         string
	index            int
	schemaStore      schema_store.SchemaStore
	conn             *sql.DB
	rowQueue         chan *message.MySQLRowMessage
	transactionQueue chan *message.MySQLTransactionMessage
}

func NewWorker(ctx context.Context, pipeline string, index, workQueueLength int, conn *sql.DB, schemaStore schema_store.SchemaStore) *Worker {
	return &Worker{
		ctx:              ctx,
		pipeline:         pipeline,
		index:            index,
		schemaStore:      schemaStore,
		conn:             conn,
		rowQueue:         make(chan *message.MySQLRowMessage, workQueueLength),
		transactionQueue: make(chan *message.MySQLTransactionMessage, workQueueLength),
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case msg, ok := <-w.rowQueue:
				if !ok {
					return
				}
				if err := w.ExecRowMessage(msg.SQLRows); err != nil {
					log.Errorf("sql row processing failed: %v", err)
				}
			case <-w.ctx.Done():
				return
			}
		}
	}()
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case msg, ok := <-w.transactionQueue:
				if !ok {
					return
				}
				if err := w.ExecTransactionMessage(msg.SQLRows); err != nil {
					log.Errorf("sql transaction processing failed: %v", err)
				}
			case <-w.ctx.Done():
				return
			}
		}
	}()
	return
}

func (w *Worker) SendMessage(msg message.Message) {
	switch msg.Type() {
	case message.MySQLRow:
		w.rowQueue <- msg.(*message.MySQLRowMessage)
	case message.MySQLTransaction:
		w.transactionQueue <- msg.(*message.MySQLTransactionMessage)
	default:
		log.Errorf("not supported message type: %v", msg.Type())

	}
}

func (w *Worker) ExecRowMessage(row message.SQLRows) error {
	tableDef, err := w.tableDef(row.Database, row.Table)
	if err != nil {
		return err
	}
	query, args, err := w.generateSQL(row, tableDef)
	if err != nil {
		return err
	}
	if row.Hint != "" {
		query = SQLWithAnnotation(row.Hint, query)
	}
	return w.write(query, args...)
}

func (w *Worker) ExecTransactionMessage(transaction []message.SQLRows) (err error) {
	sqlSets, err := w.generateTransactionSQL(transaction)
	if err != nil {
		return err
	}
	return w.transactionWrite(sqlSets)
}

func (w *Worker) tableDef(database, table string) (tableDef *mysql_schema.Table, err error) {
	var ok bool
	def, err := w.schemaStore.GetSchema(&mysql_schema.Index{
		Database: database,
		Table:    table,
	})
	if err != nil {
		return
	}
	if tableDef, ok = def.(*mysql_schema.Table); !ok {
		err = fmt.Errorf("get mysql schema_store %s.%s is not *mysql_schema.Table", database, table)
		return
	}
	return tableDef, nil
}

func (w *Worker) generateSQL(row message.SQLRows, tableDef *mysql_schema.Table) (query string, args []any, err error) {
	if row.Operation == schema_store.Delete {
		query, args = GenerateDeleteSQL(row.Contents, tableDef)
		return
	}
	switch row.WriteType {
	case message.InsertIgnore:
		return GenerateInsertIgnoreSQL(row.Contents, tableDef)
	case message.InsertOnDuplicateKey:
		return GenerateInsertOnDuplicateKeyUpdateSQL(row.Contents, tableDef)
	case message.Replace:
		return GenerateReplaceSQL(row.Contents, tableDef)
	default:
		if row.Operation == schema_store.Insert {
			return GenerateInsertSQL(row.Contents, tableDef)
		}
		if row.Operation == schema_store.Update {
			return GenerateUpdateSQL(row.Contents, tableDef, true)
		}
	}
	err = fmt.Errorf("not supported operation:%v write type:%v", row.Operation, row.WriteType)
	return
}

func (w *Worker) write(query string, args ...any) error {
	_, err := w.conn.ExecContext(w.ctx, query, args...)
	return errors.Annotatef(err, "query: %v ", query)
}

type Transaction struct {
	sql  string
	args []any
}

func (w *Worker) generateTransactionSQL(transaction []message.SQLRows) (sqlSets []Transaction, err error) {
	tableDefs := make(map[string]*mysql_schema.Table)
	sqlSets = make([]Transaction, 0, len(transaction))
	for _, row := range transaction {
		key := fmt.Sprintf("%v.%v", row.Database, row.Table)
		tableDef, exist := tableDefs[key]
		if !exist {
			tableDef, err = w.tableDef(row.Database, row.Table)
			if err != nil {
				return
			}
			tableDefs[key] = tableDef
		}
		var query string
		var args []any
		query, args, err = w.generateSQL(row, tableDef)
		if err != nil {
			return
		}
		if row.Hint != "" {
			query = SQLWithAnnotation(row.Hint, query)
		}
		sqlSets = append(sqlSets, Transaction{
			sql:  query,
			args: args,
		})
	}
	return sqlSets, nil
}

func (w *Worker) transactionWrite(sqlSets []Transaction) error {
	tx, err := w.conn.BeginTx(w.ctx, nil)
	if err != nil {
		if tx != nil {
			_ = tx.Rollback()
		}
		return errors.Trace(err)
	}
	for _, sqlData := range sqlSets {
		_, err = tx.ExecContext(w.ctx, sqlData.sql, sqlData.args...)
		if err != nil {
			if tx != nil {
				_ = tx.Rollback()
			}
			return errors.Annotatef(err, "query: %v", sqlData.sql)
		}
	}
	return tx.Commit()
}

func SQLWithAnnotation(annotationContent, sql string) string {
	return fmt.Sprintf("/*%s*/%s", annotationContent, sql)
}

func (w *Worker) Close() error {
	close(w.rowQueue)
	close(w.transactionQueue)
	w.wg.Wait()
	return nil
}

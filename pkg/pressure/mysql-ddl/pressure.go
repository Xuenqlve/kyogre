package mysql_ddl

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/data_source/mysql"
	"github.com/xuenqlve/common/ddl_parser"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/pressure"
	ds "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

// MigrationTask 表示一个迁移任务
type MigrationTask struct {
	Database  string
	Table     string
	SQL       string
	StartTime time.Time
	Cancel    context.CancelFunc
	Cmd       *exec.Cmd
	Done      chan struct{} // 用于通知任务完成
}

// Pressure 实现 mysql DDL 迁移的压力测试引擎
type Pressure struct {
	ctx          context.Context
	cancel       context.CancelFunc
	pipeline     string
	cfg          PressureConfig
	sourceConfig mysql.Config
	conn         *sql.DB
	migrations   map[string]*MigrationTask // key: schema.table
	migrationMu  sync.Mutex
	ddlQueue     chan message.Message
	semaphore    chan struct{} // 用于限制并发迁移数
	wg           sync.WaitGroup
}

const (
	MySQLDDL pressure.PressureType = "mysql-ddl"
)

func init() {
	pressure.RegisterPressure(MySQLDDL, &Pressure{}, true)
}

// Configure 初始化压力测试配置
func (p *Pressure) Configure(pipeline string, data map[string]any) (err error) {
	p.pipeline = pipeline

	// 解析配置
	if err = mapstructure.Decode(data, &p.cfg); err != nil {
		return errors.Trace(err)
	}

	// 设置默认值
	if err = p.cfg.Validate(); err != nil {
		return errors.Trace(err)
	}

	p.sourceConfig, err = ds.Config(p.cfg.DataSource)
	if err != nil {
		return errors.Trace(err)
	}
	// 解析源库连接配置
	if p.conn, err = ds.Connection(p.cfg.DataSource); err != nil {
		return errors.Trace(err)
	}
	// 初始化迁移管理器
	p.migrations = make(map[string]*MigrationTask)
	p.ddlQueue = make(chan message.Message, 100)
	p.semaphore = make(chan struct{}, p.cfg.MaxConcurrent)
	log.Infof("mysql pressure configured successfully for pipeline: %s", pipeline)
	return nil
}

// Start 启动 mysql 迁移服务
func (p *Pressure) Start(ctx context.Context) error {
	// 验证 mysql 二进制是否可用
	if _, err := exec.LookPath(p.cfg.GhostBinary); err != nil {
		return errors.Errorf("mysql binary not found: %s, please install mysql first", p.cfg.GhostBinary)
	}

	// 创建带有取消的上下文
	p.ctx, p.cancel = context.WithCancel(ctx)

	// 启动 DDL 处理 goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.handleDDLMessages()
	}()

	log.Infof("mysql pressure started for pipeline: %s", p.pipeline)
	return nil
}

// Execute 处理 DDL 消息
func (p *Pressure) Execute(msg message.Message) {
	if msg == nil {
		return
	}
	select {
	case p.ddlQueue <- msg:
		// 消息成功发送到队列
	case <-p.ctx.Done():
		// 上下文已取消
		log.Warnf("context cancelled, dropping message")
	default:
		// 队列满，丢弃消息并记录警告
		log.Warnf("DDL queue is full, dropping message")
	}
}

// handleDDLMessages 处理 DDL 队列中的消息
func (p *Pressure) handleDDLMessages() {
	for {
		select {
		case msg, ok := <-p.ddlQueue:
			if !ok {
				return
			}
			if err := p.processDDLMessage(msg); err != nil {
				log.Errorf("failed to process DDL message: %v", err)
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// processDDLMessage 处理单个 DDL 消息
func (p *Pressure) processDDLMessage(msg message.Message) error {
	if msg == nil {
		return nil
	}

	// 提取 DDL 信息
	switch m := msg.(type) {
	case *message.MySQLDDLMessage:
		return p.executeDDL(m)
	default:
		log.Warnf("unsupported message type: %T", m)
		return nil
	}
}

func (p *Pressure) executeDDL(msg *message.MySQLDDLMessage) error {
	ddlSQL, err := msg.GenerateSQL()
	if err != nil {
		return errors.Trace(err)
	}
	switch msg.Operation {
	case schema_store.ALTER_TABLE.String():
		statement, ok := msg.DDLStatement.Statement.(*ddl_parser.AlterTableStatement)
		if !ok {
			return errors.New("invalid alter table statement")
		}
		switch statement.AlterType() {
		case ddl_parser.RenameTable.String():
			// RENAME TABLE 操作需要等待该表的前一个迁移完成
			if err := p.waitForTableMigration(msg.Database, msg.Table); err != nil {
				return err
			}
			return p.execTableDDL(msg.Database, ddlSQL, msg.Hint)
		default:
			return p.executeLockTableDDL(msg.Database, msg.Table, ddlSQL, msg.Hint)
		}
	case schema_store.CREATE_DATABASE.String(), schema_store.DROP_DATABASE.String():
		return p.execDBDDL(ddlSQL, msg.Hint)
	default:
		// 其他 DDL 操作（如 CREATE TABLE、DROP TABLE 等）
		// 如果有表名信息，等待该表的前一个迁移完成
		if msg.Table != "" {
			if err := p.waitForTableMigration(msg.Database, msg.Table); err != nil {
				return err
			}
		}
		return p.execTableDDL(msg.Database, ddlSQL, msg.Hint)
	}
}

// waitForTableMigration 等待指定表的迁移任务完成
func (p *Pressure) waitForTableMigration(database, table string) error {
	key := fmt.Sprintf("%s.%s", database, table)

	p.migrationMu.Lock()
	task, exists := p.migrations[key]
	p.migrationMu.Unlock()

	if !exists {
		// 没有进行中的迁移，直接返回
		return nil
	}

	log.Infof("waiting for migration to complete for table %s", key)
	select {
	case <-task.Done:
		log.Infof("migration completed for table %s, proceeding", key)
		return nil
	case <-p.ctx.Done():
		return errors.Errorf("context cancelled while waiting for migration of table %s", key)
	}
}

func (p *Pressure) execDBDDL(stmt, hint string) error {
	if hint != "" {
		stmt = SQLWithAnnotation(hint, stmt)
	}
	if _, err := p.conn.ExecContext(p.ctx, stmt); err != nil {
		log.Errorf("ddl_parser exec failed, statement: %s, err: %v", stmt, err)
		return err
	}
	return nil
}

func (p *Pressure) execTableDDL(database, stmt, hint string) error {
	// 从 SQL 语句中提取表名（这是一个简化的实现，可能需要根据实际情况调整）
	// 对于 RENAME TABLE 操作，需要等待可能存在的迁移任务完成
	// 这里我们可以尝试从语句中解析表名，但为了安全起见，可以跳过或记录日志
	log.Debugf("executing table DDL for database %s", database)

	useSql := "use `" + database + "`"
	if hint != "" {
		stmt = SQLWithAnnotation(hint, stmt)
		useSql = SQLWithAnnotation(hint, useSql)
	}
	tx, err := p.conn.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(p.ctx, useSql); err != nil {
		_ = tx.Rollback()
		log.Errorf("ddl_parser exec failed, statement: %s, err: %v", useSql, err)
		return err
	}
	if _, err = tx.ExecContext(p.ctx, stmt); err != nil {
		_ = tx.Rollback()
		log.Errorf("ddl_parser exec failed, statement: %s, err: %v", stmt, err)
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// executeLockTableDDL 执行 DDL 迁移
func (p *Pressure) executeLockTableDDL(database, table, ddlSql, hint string) error {
	if database == "" || table == "" {
		return errors.Errorf("database and table are required for DDL execution")
	}

	key := fmt.Sprintf("%s.%s", database, table)

	// 等待该表的前一个迁移完成
	if err := p.waitForTableMigration(database, table); err != nil {
		return err
	}

	// 获取信号量（限制并发）
	select {
	case p.semaphore <- struct{}{}:
		// 成功获取信号量
	case <-p.ctx.Done():
		return errors.Errorf("context cancelled")
	}

	// 创建迁移任务
	taskCtx, taskCancel := context.WithTimeout(p.ctx, time.Duration(p.cfg.Timeout)*time.Second)
	task := &MigrationTask{
		Database:  database,
		Table:     table,
		SQL:       ddlSql,
		StartTime: time.Now(),
		Cancel:    taskCancel,
		Done:      make(chan struct{}), // 初始化 Done 通道
	}

	// 存储迁移任务
	p.migrationMu.Lock()
	p.migrations[key] = task
	p.migrationMu.Unlock()

	// 执行迁移（非阻塞）
	p.wg.Add(1)
	go func() {
		defer func() {
			p.wg.Done()
			task.Cancel()
			<-p.semaphore // 释放信号量

			p.migrationMu.Lock()
			delete(p.migrations, key)
			p.migrationMu.Unlock()

			close(task.Done) // 关闭 Done 通道，通知等待者任务已完成
		}()
		if err := p.runGhostMigration(taskCtx, task); err != nil {
			log.Errorf("migration failed for %s: %v", key, err)
		} else {
			log.Infof("migration completed successfully for %s", key)
		}
	}()

	return nil
}

// runGhostMigration 运行 mysql 迁移
func (p *Pressure) runGhostMigration(ctx context.Context, task *MigrationTask) error {
	if task == nil {
		return errors.Errorf("task is nil")
	}

	// 构建 mysql 命令
	args := p.buildGhostArgs(task)
	cmd := exec.CommandContext(ctx, p.cfg.GhostBinary, args...)

	// 存储命令引用
	task.Cmd = cmd

	log.Infof("starting mysql migration: %s.%s, command: %s %s",
		task.Database, task.Table, p.cfg.GhostBinary, strings.Join(args, " "))

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("mysql execution failed: %v, output: %s", err, string(output))
	}

	log.Infof("mysql migration completed for %s.%s, output: %s",
		task.Database, task.Table, string(output))

	return nil
}

// buildGhostArgs 构建 mysql 命令参数
func (p *Pressure) buildGhostArgs(task *MigrationTask) []string {
	args := []string{
		"--host=" + p.sourceConfig.Host,
		"--port=" + fmt.Sprintf("%d", p.sourceConfig.Port),
		"--user=" + p.sourceConfig.Username,
		"--password=" + p.sourceConfig.Password,
		"--database=" + task.Database,
		"--table=" + task.Table,
		"--alter=" + task.SQL,
		"--chunk-size=" + fmt.Sprintf("%d", p.cfg.ChunkSize),
		"--max-load=Threads_running=" + fmt.Sprintf("%d", p.cfg.MaxLoad),
		"--verbose",
	}

	// 添加可选参数
	if p.cfg.ExecuteChanges {
		args = append(args, "--execute")
	} else {
		args = append(args, "--test-on-replica")
	}

	if p.cfg.AllowOnMaster {
		args = append(args, "--allow-on-master")
	}

	if p.cfg.CutOver {
		args = append(args, "--cut-over=default")
	} else {
		args = append(args, "--postpone-cut-over-flag-file=/tmp/ghost-postpone")
	}
	if p.cfg.DropOldTable {
		args = append(args, "--initially-drop-old-table")
	}
	return args
}

// Close 优雅关闭压力测试引擎
func (p *Pressure) Close() error {

	if p.ddlQueue != nil {
		close(p.ddlQueue)
	}

	// 等待所有 goroutine 完成
	// 设置超时防止无限等待
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Infof("all workers stopped")
	case <-time.After(30 * time.Second):
		log.Warnf("timeout waiting for workers to stop")
	}

	// 关闭 DDL 队列

	// 停止所有进行中的迁移
	p.migrationMu.Lock()
	for key, task := range p.migrations {
		if task != nil && task.Cmd != nil && task.Cmd.Process != nil {
			log.Infof("stopping migration for %s", key)
			if err := task.Cmd.Process.Kill(); err != nil {
				log.Errorf("failed to kill migration process for %s: %v", key, err)
			}
			if task.Cancel != nil {
				task.Cancel()
			}
		}
	}
	p.migrationMu.Unlock()

	// 关闭数据库连接
	log.Infof("mysql pressure closed for pipeline: %s", p.pipeline)
	return nil
}

func SQLWithAnnotation(annotationContent, sql string) string {
	return fmt.Sprintf("/*%s*/%s", annotationContent, sql)
}

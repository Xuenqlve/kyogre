package gh_ost

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/common/errors"
	"github.com/xuenqlve/kyogre/internal/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/pkg/data_source"
)

// PressureConfig 配置 gh-ost DDL 迁移的参数
type PressureConfig struct {
	// 源数据源名称
	SourceDataSource string `mapstructure:"source-data-source" json:"source-data-source" yaml:"source-data-source" toml:"source-data-source"`
	// gh-ost 二进制文件路径（默认使用 PATH 中的 gh-ost）
	GhostBinary string `mapstructure:"ghost-binary" json:"ghost-binary" yaml:"ghost-binary" toml:"ghost-binary"`
	// 最大并发迁移数
	MaxConcurrentMigrations int `mapstructure:"max-concurrent-migrations" json:"max-concurrent-migrations" yaml:"max-concurrent-migrations" toml:"max-concurrent-migrations"`
	// 批处理大小
	ChunkSize int `mapstructure:"chunk-size" json:"chunk-size" yaml:"chunk-size" toml:"chunk-size"`
	// 最大负载
	MaxLoad int `mapstructure:"max-load" json:"max-load" yaml:"max-load" toml:"max-load"`
	// 是否启用 DML 变更（默认 true）
	ExecuteChanges bool `mapstructure:"execute-changes" json:"execute-changes" yaml:"execute-changes" toml:"execute-changes"`
	// 是否允许在主库上运行
	AllowOnMaster bool `mapstructure:"allow-on-master" json:"allow-on-master" yaml:"allow-on-master" toml:"allow-on-master"`
	// 超时时间（秒）
	Timeout int `mapstructure:"timeout" json:"timeout" yaml:"timeout" toml:"timeout"`
	// 是否切换表（完成后切换旧表和新表）
	CutOver bool `mapstructure:"cut-over" json:"cut-over" yaml:"cut-over" toml:"cut-over"`
}

// MigrationTask 表示一个迁移任务
type MigrationTask struct {
	Database  string
	Table     string
	SQL       string
	StartTime time.Time
	Cancel    context.CancelFunc
	Cmd       *exec.Cmd
}

// Pressure 实现 gh-ost DDL 迁移的压力测试引擎
type Pressure struct {
	ctx          context.Context
	cancel       context.CancelFunc
	pipeline     string
	cfg          PressureConfig
	sourceConn   *sql.DB
	sourceConfig *MySQLConfig
	migrations   map[string]*MigrationTask // key: schema.table
	migrationMu  sync.Mutex
	ddlQueue     chan message.Message
	semaphore    chan struct{} // 用于限制并发迁移数
	wg           sync.WaitGroup
}

// MySQLConfig MySQL 连接配置（从 DSN 解析）
type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// Configure 初始化压力测试配置
func (p *Pressure) Configure(pipeline string, data map[string]any) (err error) {
	p.pipeline = pipeline

	// 解析配置
	if err = mapstructure.Decode(data, &p.cfg); err != nil {
		return errors.Trace(err)
	}

	// 验证必要的配置
	if p.cfg.SourceDataSource == "" {
		return errors.Errorf("source-data-source is required")
	}

	// 设置默认值
	if p.cfg.GhostBinary == "" {
		p.cfg.GhostBinary = "gh-ost" // 使用 PATH 中的 gh-ost
	}
	if p.cfg.MaxConcurrentMigrations <= 0 {
		p.cfg.MaxConcurrentMigrations = 1
	}
	if p.cfg.MaxLoad <= 0 {
		p.cfg.MaxLoad = 100
	}
	if p.cfg.ChunkSize <= 0 {
		p.cfg.ChunkSize = 1000
	}
	if p.cfg.Timeout <= 0 {
		p.cfg.Timeout = 3600 // 默认 1 小时
	}

	// 获取源库连接
	if p.sourceConn, err = data_source.MySQLConnection(p.cfg.SourceDataSource); err != nil {
		return errors.Trace(err)
	}

	// 解析源库连接配置
	if p.sourceConfig, err = parseMySQLDSN(p.cfg.SourceDataSource); err != nil {
		p.sourceConn.Close()
		return errors.Trace(err)
	}

	// 初始化迁移管理器
	p.migrations = make(map[string]*MigrationTask)
	p.ddlQueue = make(chan message.Message, 100)
	p.semaphore = make(chan struct{}, p.cfg.MaxConcurrentMigrations)

	log.Infof("gh-ost pressure configured successfully for pipeline: %s", pipeline)
	return nil
}

// Start 启动 gh-ost 迁移服务
func (p *Pressure) Start(ctx context.Context) error {
	if p.sourceConn == nil {
		return errors.Errorf("pressure not configured, call Configure first")
	}

	// 验证 gh-ost 二进制是否可用
	if _, err := exec.LookPath(p.cfg.GhostBinary); err != nil {
		return errors.Errorf("gh-ost binary not found: %s, please install gh-ost first", p.cfg.GhostBinary)
	}

	// 创建带有取消的上下文
	p.ctx, p.cancel = context.WithCancel(ctx)

	// 启动 DDL 处理 goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.handleDDLMessages()
	}()

	log.Infof("gh-ost pressure started for pipeline: %s", p.pipeline)
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
	case *DDLMessage:
		return p.executeDDL(m)
	default:
		log.Warnf("unsupported message type: %T", m)
		return nil
	}
}

// executeDDL 执行 DDL 迁移
func (p *Pressure) executeDDL(ddlMsg *DDLMessage) error {
	if ddlMsg == nil || ddlMsg.SQL == "" {
		return errors.Errorf("invalid DDL message")
	}

	database := ddlMsg.Database
	table := ddlMsg.Table
	ddlSQL := ddlMsg.SQL

	if database == "" || table == "" {
		return errors.Errorf("database and table are required for DDL execution")
	}

	key := fmt.Sprintf("%s.%s", database, table)

	// 检查是否已有该表的迁移
	p.migrationMu.Lock()
	_, exists := p.migrations[key]
	p.migrationMu.Unlock()

	if exists {
		return errors.Errorf("migration for table %s is already in progress", key)
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
		SQL:       ddlSQL,
		StartTime: time.Now(),
		Cancel:    taskCancel,
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
		}()

		if err := p.runGhostMigration(taskCtx, task); err != nil {
			log.Errorf("migration failed for %s: %v", key, err)
		} else {
			log.Infof("migration completed successfully for %s", key)
		}
	}()

	return nil
}

// runGhostMigration 运行 gh-ost 迁移
func (p *Pressure) runGhostMigration(ctx context.Context, task *MigrationTask) error {
	if task == nil {
		return errors.Errorf("task is nil")
	}

	// 构建 gh-ost 命令
	args := p.buildGhostArgs(task)
	cmd := exec.CommandContext(ctx, p.cfg.GhostBinary, args...)

	// 存储命令引用
	task.Cmd = cmd

	log.Infof("starting gh-ost migration: %s.%s, command: %s %s",
		task.Database, task.Table, p.cfg.GhostBinary, strings.Join(args, " "))

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("gh-ost execution failed: %v, output: %s", err, string(output))
	}

	log.Infof("gh-ost migration completed for %s.%s, output: %s",
		task.Database, task.Table, string(output))

	return nil
}

// buildGhostArgs 构建 gh-ost 命令参数
func (p *Pressure) buildGhostArgs(task *MigrationTask) []string {
	args := []string{
		"--host=" + p.sourceConfig.Host,
		"--port=" + p.sourceConfig.Port,
		"--user=" + p.sourceConfig.User,
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

	return args
}

// parseMySQLDSN 解析 MySQL DSN 连接字符串
// 格式: user:password@tcp(host:port)/database
func parseMySQLDSN(dsn string) (*MySQLConfig, error) {
	config := &MySQLConfig{
		Port: "3306", // 默认端口
	}

	// 简化的 DSN 解析
	// 注：实际使用应该使用更健壮的解析器
	parts := strings.Split(dsn, "@")
	if len(parts) < 2 {
		return nil, errors.Errorf("invalid DSN format")
	}

	// 解析用户名和密码
	userPass := strings.Split(parts[0], ":")
	if len(userPass) >= 1 {
		config.User = userPass[0]
	}
	if len(userPass) >= 2 {
		config.Password = userPass[1]
	}

	// 解析主机、端口和数据库
	remaining := parts[1]
	if strings.HasPrefix(remaining, "tcp(") {
		end := strings.Index(remaining, ")")
		if end > 4 {
			hostPort := remaining[4:end]
			hp := strings.Split(hostPort, ":")
			if len(hp) >= 1 {
				config.Host = hp[0]
			}
			if len(hp) >= 2 {
				config.Port = hp[1]
			}

			// 数据库名
			dbPart := remaining[end+1:]
			if strings.HasPrefix(dbPart, "/") {
				dbName := strings.TrimPrefix(dbPart, "/")
				// 移除查询参数
				if idx := strings.Index(dbName, "?"); idx > 0 {
					dbName = dbName[:idx]
				}
				config.Database = dbName
			}
		}
	}

	return config, nil
}

// Close 优雅关闭压力测试引擎
func (p *Pressure) Close() error {
	if p.cancel != nil {
		p.cancel()
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
	if p.ddlQueue != nil {
		close(p.ddlQueue)
	}

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
	if p.sourceConn != nil {
		if err := p.sourceConn.Close(); err != nil {
			log.Errorf("failed to close source connection: %v", err)
			return errors.Trace(err)
		}
	}

	log.Infof("gh-ost pressure closed for pipeline: %s", p.pipeline)
	return nil
}

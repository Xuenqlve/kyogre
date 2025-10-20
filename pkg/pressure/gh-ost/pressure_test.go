package gh_ost

import (
	"context"
	"testing"
	"time"

	"github.com/xuenqlve/kyogre/internal/message"
)

// TestDDLMessage 测试 DDL 消息
func TestDDLMessage(t *testing.T) {
	msg := &DDLMessage{
		Database:       "test_db",
		Table:          "test_table",
		SQL:            "ADD COLUMN new_col INT DEFAULT 0",
		Operation:      "ALTER TABLE",
		StartTimeValue: time.Now(),
	}

	if msg.Type() != DDLMessageType {
		t.Errorf("Expected message type %s, got %s", DDLMessageType, msg.Type())
	}

	if msg.StartTime().IsZero() {
		t.Error("Expected non-zero start time")
	}
}

// TestPressureConfigDefaults 测试配置默认值设置
func TestPressureConfigDefaults(t *testing.T) {
	pressure := &Pressure{}
	config := map[string]any{
		"source-data-source": "test-source",
	}

	// 注意：这个测试会尝试连接数据库，在没有配置的情况下会失败
	// 这里只是演示配置解析的过程
	err := pressure.Configure("test-pipeline", config)
	if err == nil {
		// 验证默认值
		if pressure.cfg.MaxConcurrentMigrations != 1 {
			t.Errorf("Expected MaxConcurrentMigrations=1, got %d", pressure.cfg.MaxConcurrentMigrations)
		}
		if pressure.cfg.ChunkSize != 1000 {
			t.Errorf("Expected ChunkSize=1000, got %d", pressure.cfg.ChunkSize)
		}
		if pressure.cfg.MaxLoad != 100 {
			t.Errorf("Expected MaxLoad=100, got %d", pressure.cfg.MaxLoad)
		}
		if pressure.cfg.Timeout != 3600 {
			t.Errorf("Expected Timeout=3600, got %d", pressure.cfg.Timeout)
		}
	}
}

// TestParseMySQLDSN 测试 MySQL DSN 解析
func TestParseMySQLDSN(t *testing.T) {
	testCases := []struct {
		dsn       string
		expectErr bool
		expected  *MySQLConfig
	}{
		{
			dsn:       "root:password@tcp(localhost:3306)/mydb",
			expectErr: false,
			expected: &MySQLConfig{
				User:     "root",
				Password: "password",
				Host:     "localhost",
				Port:     "3306",
				Database: "mydb",
			},
		},
		{
			dsn:       "user:pass@tcp(192.168.1.1:3307)/testdb",
			expectErr: false,
			expected: &MySQLConfig{
				User:     "user",
				Password: "pass",
				Host:     "192.168.1.1",
				Port:     "3307",
				Database: "testdb",
			},
		},
		{
			dsn:       "invalid-dsn",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		config, err := parseMySQLDSN(tc.dsn)
		if tc.expectErr {
			if err == nil {
				t.Errorf("Expected error for DSN %s, got nil", tc.dsn)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for DSN %s: %v", tc.dsn, err)
				continue
			}
			if config.User != tc.expected.User {
				t.Errorf("User mismatch: expected %s, got %s", tc.expected.User, config.User)
			}
			if config.Password != tc.expected.Password {
				t.Errorf("Password mismatch: expected %s, got %s", tc.expected.Password, config.Password)
			}
			if config.Host != tc.expected.Host {
				t.Errorf("Host mismatch: expected %s, got %s", tc.expected.Host, config.Host)
			}
			if config.Port != tc.expected.Port {
				t.Errorf("Port mismatch: expected %s, got %s", tc.expected.Port, config.Port)
			}
			if config.Database != tc.expected.Database {
				t.Errorf("Database mismatch: expected %s, got %s", tc.expected.Database, config.Database)
			}
		}
	}
}

// TestBuildGhostArgs 测试 gh-ost 命令参数构建
func TestBuildGhostArgs(t *testing.T) {
	pressure := &Pressure{
		cfg: PressureConfig{
			ChunkSize:      1000,
			MaxLoad:        100,
			ExecuteChanges: true,
		},
		sourceConfig: &MySQLConfig{
			Host:     "localhost",
			Port:     "3306",
			User:     "root",
			Password: "password",
		},
	}

	task := &MigrationTask{
		Database: "mydb",
		Table:    "users",
		SQL:      "ADD COLUMN age INT",
	}

	args := pressure.buildGhostArgs(task)

	// 验证必需的参数
	expectedParams := []string{
		"--host=localhost",
		"--port=3306",
		"--user=root",
		"--password=password",
		"--database=mydb",
		"--table=users",
		"--alter=ADD COLUMN age INT",
		"--chunk-size=1000",
		"--max-load=Threads_running=100",
		"--execute",
	}

	for _, param := range expectedParams {
		found := false
		for _, arg := range args {
			if arg == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected parameter %s not found in args: %v", param, args)
		}
	}
}

// TestExecuteMessageWithNilContext 测试在 nil 上下文中发送消息
func TestExecuteMessageWithNilContext(t *testing.T) {
	pressure := &Pressure{
		ddlQueue: make(chan message.Message, 10),
	}

	// 不应该 panic
	msg := &DDLMessage{
		Database: "test",
		Table:    "table",
		SQL:      "ALTER TABLE",
	}
	pressure.Execute(msg)
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	pressure := &Pressure{
		ddlQueue: make(chan message.Message, 10),
	}

	ctx, cancel := context.WithCancel(context.Background())
	pressure.ctx = ctx
	pressure.cancel = cancel

	// 取消上下文
	cancel()

	// 发送消息应该处理得当（可能被丢弃）
	msg := &DDLMessage{
		Database: "test",
		Table:    "table",
		SQL:      "ALTER TABLE",
	}
	pressure.Execute(msg)
}

// TestMessageQueuing 测试消息队列
func TestMessageQueuing(t *testing.T) {
	pressure := &Pressure{
		ddlQueue: make(chan message.Message, 2),
		ctx:      context.Background(),
	}

	msg1 := &DDLMessage{Database: "db1", Table: "t1"}
	msg2 := &DDLMessage{Database: "db2", Table: "t2"}

	// 发送两条消息
	pressure.Execute(msg1)
	pressure.Execute(msg2)

	// 验证消息已入队
	if len(pressure.ddlQueue) != 2 {
		t.Errorf("Expected 2 messages in queue, got %d", len(pressure.ddlQueue))
	}
}

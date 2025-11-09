package mysql_ddl

import "github.com/xuenqlve/common/errors"

// PressureConfig 配置 mysql_config DDL 迁移的参数
type PressureConfig struct {
	// 源数据源名称
	DataSource string `mapstructure:"data-source" json:"data-source" yaml:"data-source" toml:"data-source"`
	// ghost 二进制文件路径（默认使用 PATH 中的 gh-ost）
	GhostBinary string `mapstructure:"ghost-binary" json:"ghost-binary" yaml:"ghost-binary" toml:"ghost-binary"`
	// 最大并发迁移数
	MaxConcurrent int `mapstructure:"max-concurrent" json:"max-concurrent" yaml:"max-concurrent" toml:"max-concurrent"`
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
	CutOver      bool `mapstructure:"cut-over" json:"cut-over" yaml:"cut-over" toml:"cut-over"`
	DropOldTable bool `mapstructure:"drop-old-table" json:"drop-old_table" yaml:"drop-old_table" toml:"drop-old_table"`
}

func (cfg *PressureConfig) Validate() error {
	if cfg.DataSource == "" {
		return errors.New("data-source is required")
	}
	if cfg.GhostBinary == "" {
		cfg.GhostBinary = "gh-ost" // 使用 PATH 中的 mysql_config
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if cfg.MaxLoad <= 0 {
		cfg.MaxLoad = 100
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 1000
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3600 // 默认 1 小时
	}
	cfg.AllowOnMaster = true
	cfg.CutOver = true
	cfg.ExecuteChanges = true
	cfg.DropOldTable = true
	return nil
}

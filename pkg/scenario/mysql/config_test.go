package mysql

import (
	"testing"

	"github.com/xuenqlve/kyogre/internal/config"
	mysqlgen "github.com/xuenqlve/kyogre/pkg/generator/mysql"
)

func TestConfigValidateDefaults(t *testing.T) {
	cfg := Config{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if cfg.WorkerCount != 1 {
		t.Fatalf("WorkerCount = %d, want 1", cfg.WorkerCount)
	}

	if cfg.GenerationStrategy == nil || cfg.GenerationStrategy.RandomConfig == nil {
		t.Fatalf("GenerationStrategy defaults not applied")
	}

	if cfg.DependencyConfig.Type != mysqlgen.DML {
		t.Fatalf("DependencyConfig.Type = %s, want %s", cfg.DependencyConfig.Type, mysqlgen.DML)
	}

	dep, err := cfg.DependencyConfig.GetDependencyConfig()
	if err != nil {
		t.Fatalf("GetDependencyConfig() error = %v", err)
	}

	if _, ok := dep.(*mysqlgen.DMLConfig); !ok {
		t.Fatalf("DependencyConfig should default to *DMLConfig, got %T", dep)
	}
}

func TestConfigValidateIQuery(t *testing.T) {
	cfg := Config{EnableIQuery: true}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("Validate() expect error when iquery enabled without key/module")
	}

	cfg = Config{
		IQuery: IQueryConfig{
			Enabled: true,
			Key:     "mysql",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error with key: %v", err)
	}

	cfg = Config{
		IQuery: IQueryConfig{
			Enabled: true,
			Key:     "mysql",
			Module:  &config.ConfigureMold{Type: "mysql"},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("Validate() expect error when both key and module are set")
	}
}

func TestConfigValidateDependency(t *testing.T) {
	cfg := Config{
		DependencyConfig: DependencyConfigData{
			Type: "unknown",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("Validate() expect error for unknown dependency type")
	}
}

package generator

import (
	"testing"

	"github.com/xuenqlve/kyogre/pkg/generator/mysql"
)

func TestConfigLoaderLoadDMLConfig(t *testing.T) {
	loader := mysql.NewConfigLoader()

	data := map[string]any{
		"type":         "dml",
		"operation":    "insert",
		"table_select": "random",
		"count":        "10",
	}

	config, err := loader.LoadDependencyConfig(data)
	if err != nil {
		t.Fatalf("LoadDependencyConfig() error = %v", err)
	}

	dmlConfig, ok := config.(*mysql.DMLConfig)
	if !ok {
		t.Fatalf("Expected *DMLConfig, got %T", config)
	}

	if dmlConfig.Operation != "insert" {
		t.Errorf("Operation = %v, want insert", dmlConfig.Operation)
	}

	if dmlConfig.Count != "10" {
		t.Errorf("Count = %v, want 10", dmlConfig.Count)
	}

	if config.Type() != "dml" {
		t.Errorf("Type() = %v, want dml", config.Type())
	}
}

func TestConfigLoaderLoadTransactionConfig(t *testing.T) {
	loader := mysql.NewConfigLoader()

	data := map[string]any{
		"type": "transaction",
		"operations": map[string]any{
			"users": map[string]any{
				"operation":  "insert",
				"count":      5,
				"write_type": "insert",
			},
		},
	}

	config, err := loader.LoadDependencyConfig(data)
	if err != nil {
		t.Fatalf("LoadDependencyConfig() error = %v", err)
	}

	transConfig, ok := config.(*mysql.TransactionConfig)
	if !ok {
		t.Fatalf("Expected *TransactionConfig, got %T", config)
	}

	if len(transConfig.Operations) != 1 {
		t.Errorf("Operations count = %d, want 1", len(transConfig.Operations))
	}

	if config.Type() != "transaction" {
		t.Errorf("Type() = %v, want transaction", config.Type())
	}
}

func TestConfigLoaderLoadDDLConfig(t *testing.T) {
	loader := mysql.NewConfigLoader()

	data := map[string]any{
		"type":    "ddl",
		"DDLType": "ALTER",
	}

	config, err := loader.LoadDependencyConfig(data)
	if err != nil {
		t.Fatalf("LoadDependencyConfig() error = %v", err)
	}

	ddlConfig, ok := config.(*mysql.DDLConfig)
	if !ok {
		t.Fatalf("Expected *DDLConfig, got %T", config)
	}

	if ddlConfig.DDLType != "ALTER" {
		t.Errorf("DDLType = %v, want ALTER", ddlConfig.DDLType)
	}

	if config.Type() != "ddl" {
		t.Errorf("Type() = %v, want ddl", config.Type())
	}
}

func TestConfigLoaderInvalidType(t *testing.T) {
	loader := mysql.NewConfigLoader()

	data := map[string]any{
		"type": "unknown",
	}

	_, err := loader.LoadDependencyConfig(data)
	if err == nil {
		t.Fatalf("LoadDependencyConfig() expected error for invalid type")
	}
}

func TestConfigLoaderLoadFromJSON(t *testing.T) {
	loader := mysql.NewConfigLoader()

	jsonStr := `{"type":"dml","operation":"update","table_select":"ordered","count":"5"}`

	config, err := loader.LoadDependencyConfigFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("LoadDependencyConfigFromJSON() error = %v", err)
	}

	dmlConfig, ok := config.(*mysql.DMLConfig)
	if !ok {
		t.Fatalf("Expected *DMLConfig, got %T", config)
	}

	if dmlConfig.Operation != "update" {
		t.Errorf("Operation = %v, want update", dmlConfig.Operation)
	}
}

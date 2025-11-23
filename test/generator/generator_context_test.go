package generator

import (
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/generator/mysql"
)

func TestDMLConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *mysql.DMLConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &mysql.DMLConfig{
				Operation:   "insert",
				TableSelect: "random",
				Count:       "10",
			},
			wantErr: false,
		},
		{
			name: "missing operation",
			config: &mysql.DMLConfig{
				TableSelect: "random",
				Count:       "10",
			},
			wantErr: true,
		},
		{
			name: "empty table select (should default)",
			config: &mysql.DMLConfig{
				Operation: "update",
				Count:     "5",
			},
			wantErr: false,
		},
		{
			name: "empty count (should default)",
			config: &mysql.DMLConfig{
				Operation:   "delete",
				TableSelect: "ordered",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DMLConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证Type方法
			if tt.config.Type() != "dml" {
				t.Errorf("DMLConfig.Type() = %v, want dml", tt.config.Type())
			}
		})
	}
}

func TestTransactionConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *mysql.TransactionConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &mysql.TransactionConfig{
				Operations: map[string]mysql.TransactionOp{
					"users": {
						Operation: "insert",
						Count:     5,
						WriteType: "insert",
					},
					"orders": {
						Operation: "update",
						Count:     3,
						WriteType: "update",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty operations",
			config: &mysql.TransactionConfig{
				Operations: map[string]mysql.TransactionOp{},
			},
			wantErr: true,
		},
		{
			name: "zero count",
			config: &mysql.TransactionConfig{
				Operations: map[string]mysql.TransactionOp{
					"users": {
						Operation: "insert",
						Count:     0,
						WriteType: "insert",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TransactionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证Type方法
			if tt.config.Type() != "transaction" {
				t.Errorf("TransactionConfig.Type() = %v, want transaction", tt.config.Type())
			}
		})
	}
}

func TestDDLConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *mysql.DDLConfig
		wantErr bool
	}{
		{
			name: "valid alter config",
			config: &mysql.DDLConfig{
				DDLType: "ALTER",
				Columns: []mysql.ColumnChange{
					{
						Name:   "new_col",
						Type:   "INT",
						Change: "ADD",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing ddl type",
			config: &mysql.DDLConfig{
				Columns: []mysql.ColumnChange{},
			},
			wantErr: true,
		},
		{
			name: "valid create config",
			config: &mysql.DDLConfig{
				DDLType: "CREATE",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DDLConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证Type方法
			if tt.config.Type() != "ddl" {
				t.Errorf("DDLConfig.Type() = %v, want ddl", tt.config.Type())
			}
		})
	}
}

func TestDependencyConfigInterface(t *testing.T) {
	// 验证所有Config类型都实现了DependencyConfig接口
	var _ plugin.DependencyConfig = (*mysql.DMLConfig)(nil)
	var _ plugin.DependencyConfig = (*mysql.TransactionConfig)(nil)
	var _ plugin.DependencyConfig = (*mysql.DDLConfig)(nil)
}

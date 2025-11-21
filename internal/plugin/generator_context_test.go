package plugin

import (
	"testing"
)

func TestDMLConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DMLConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &DMLConfig{
				Operation:   "insert",
				TableSelect: "random",
				Count:       "10",
			},
			wantErr: false,
		},
		{
			name: "missing operation",
			config: &DMLConfig{
				TableSelect: "random",
				Count:       "10",
			},
			wantErr: true,
		},
		{
			name: "empty table select (should default)",
			config: &DMLConfig{
				Operation: "update",
				Count:     "5",
			},
			wantErr: false,
		},
		{
			name: "empty count (should default)",
			config: &DMLConfig{
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
		config  *TransactionConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &TransactionConfig{
				Operations: map[string]TransactionOp{
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
			config: &TransactionConfig{
				Operations: map[string]TransactionOp{},
			},
			wantErr: true,
		},
		{
			name: "zero count",
			config: &TransactionConfig{
				Operations: map[string]TransactionOp{
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
		config  *DDLConfig
		wantErr bool
	}{
		{
			name: "valid alter config",
			config: &DDLConfig{
				DDLType: "ALTER",
				Columns: []ColumnChange{
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
			config: &DDLConfig{
				Columns: []ColumnChange{},
			},
			wantErr: true,
		},
		{
			name: "valid create config",
			config: &DDLConfig{
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
	var _ DependencyConfig = (*DMLConfig)(nil)
	var _ DependencyConfig = (*TransactionConfig)(nil)
	var _ DependencyConfig = (*DDLConfig)(nil)
}

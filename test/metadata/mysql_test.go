package metadata

import (
	"context"
	"fmt"
	"testing"

	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/metadata/common"
	"github.com/xuenqlve/kyogre/pkg/metadata/mysql_config"
	"github.com/xuenqlve/kyogre/pkg/metadata/mysql_mock"
)

func TestCreateTableSQL(t *testing.T) {
	tests := []struct {
		name     string
		table    common.Table
		database string
		want     string
	}{
		{
			name: "simple table with primary key",
			table: common.Table{
				Table: "users",
				Columns: []common.Column{
					{
						Column: "id",
						Type:   "primary",
					},
					{
						Column: "username",
						Type:   "string",
					},
					{
						Column: "email",
						Type:   "varchar",
					},
				},
				Indexes: []common.Index{
					{
						Name:      "",
						Columns:   []string{"id"},
						IsPrimary: true,
						IsUnique:  false,
					},
					{
						Name:      "uk_email",
						Columns:   []string{"email"},
						IsPrimary: false,
						IsUnique:  true,
					},
				},
			},
			database: "test_db",
		},
		{
			name: "table with composite index",
			table: common.Table{
				Table: "orders",
				Columns: []common.Column{
					{
						Column: "order_id",
						Type:   "primary",
					},
					{
						Column: "user_id",
						Type:   "bigint_unsigned",
					},
					{
						Column: "status",
						Type:   "enum",
					},
				},
				Indexes: []common.Index{
					{
						Name:      "",
						Columns:   []string{"order_id"},
						IsPrimary: true,
						IsUnique:  false,
					},
					{
						Name:      "idx_user_status",
						Columns:   []string{"user_id", "status"},
						IsPrimary: false,
						IsUnique:  false,
					},
				},
			},
			database: "shop_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.table.CreateTableSQL(tt.database)
			fmt.Printf("Test: %s\n", tt.name)
			fmt.Printf("Generated SQL:\n%s\n\n", result)

			// 验证SQL包含基本的结构
			if !contains(result, "CREATE TABLE IF NOT EXISTS") {
				t.Errorf("Missing CREATE TABLE statement")
			}
			if !contains(result, tt.database) {
				t.Errorf("Missing database name: %s", tt.database)
			}
			if !contains(result, tt.table.Table) {
				t.Errorf("Missing table name: %s", tt.table.Table)
			}
			if !contains(result, "ENGINE=InnoDB") {
				t.Errorf("Missing ENGINE specification")
			}
			if !contains(result, "CHARSET=utf8mb4") {
				t.Errorf("Missing charset specification")
			}
			if !contains(result, "PRIMARY KEY") {
				t.Errorf("Missing PRIMARY KEY definition")
			}
		})
	}
}

func TestColumnDefinition(t *testing.T) {
	tests := []struct {
		name   string
		column common.Column
		want   string
	}{
		{
			name: "primary key with TypeTransform",
			column: common.Column{
				Column: "id",
				Type:   "primary",
			},
			want: "`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'id'",
		},
		{
			name: "simple string type",
			column: common.Column{
				Column: "username",
				Type:   "string",
			},
			want: "`username` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'username'",
		},
		{
			name: "int type with TypeTransform",
			column: common.Column{
				Column: "status",
				Type:   "int",
			},
			want: "`status` INT NOT NULL DEFAULT 0 COMMENT 'status'",
		},
		{
			name: "decimal type",
			column: common.Column{
				Column: "price",
				Type:   "decimal",
			},
			want: "`price` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT 'price'",
		},
		{
			name: "timestamp type",
			column: common.Column{
				Column: "created_at",
				Type:   "timestamp",
			},
			want: "`created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at'",
		},
		{
			name: "timestamp_update type",
			column: common.Column{
				Column: "updated_at",
				Type:   "timestamp_update",
			},
			want: "`updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'updated_at'",
		},
		{
			name: "json type",
			column: common.Column{
				Column: "metadata",
				Type:   "json",
			},
			want: "`metadata` JSON COMMENT 'metadata'",
		},
		{
			name: "boolean type",
			column: common.Column{
				Column: "is_active",
				Type:   "boolean",
			},
			want: "`is_active` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'is_active'",
		},
		{
			name: "varchar_large type",
			column: common.Column{
				Column: "description",
				Type:   "varchar_large",
			},
			want: "`description` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT 'description'",
		},
		{
			name: "text type",
			column: common.Column{
				Column: "content",
				Type:   "text",
			},
			want: "`content` TEXT COMMENT 'content'",
		},
		{
			name: "date type",
			column: common.Column{
				Column: "birth_date",
				Type:   "date",
			},
			want: "`birth_date` DATE NOT NULL DEFAULT '2000-01-01' COMMENT 'birth_date'",
		},
		{
			name: "datetime type",
			column: common.Column{
				Column: "event_time",
				Type:   "datetime",
			},
			want: "`event_time` DATETIME NOT NULL DEFAULT '2000-01-01 00:00:00' COMMENT 'event_time'",
		},
		{
			name: "custom full type (fallback)",
			column: common.Column{
				Column: "custom_col",
				Type:   "DECIMAL(15,4) NOT NULL DEFAULT 0.0000",
			},
			want: "`custom_col` DECIMAL(15,4) NOT NULL DEFAULT 0.0000 COMMENT 'custom_col'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.column.ColumnDefinition()
			fmt.Printf("Test: %s\nResult: %s\nExpected: %s\n\n", tt.name, result, tt.want)
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}

func TestTypeTransform(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected string
	}{
		// 主键类型
		{"primary key", "primary", "BIGINT UNSIGNED NOT NULL AUTO_INCREMENT"},

		// 整数类型
		{"tinyint", "tinyint", "TINYINT NOT NULL DEFAULT 0"},
		{"smallint", "smallint", "SMALLINT NOT NULL DEFAULT 0"},
		{"mediumint", "mediumint", "MEDIUMINT NOT NULL DEFAULT 0"},
		{"int", "int", "INT NOT NULL DEFAULT 0"},
		{"bigint", "bigint", "BIGINT NOT NULL DEFAULT 0"},
		{"bigint_unsigned", "bigint_unsigned", "BIGINT UNSIGNED NOT NULL DEFAULT 0"},

		// 浮点数类型
		{"float", "float", "FLOAT NOT NULL DEFAULT 0.0"},
		{"double", "double", "DOUBLE NOT NULL DEFAULT 0.0"},
		{"decimal", "decimal", "DECIMAL(10,2) NOT NULL DEFAULT 0.00"},

		// 字符串类型
		{"char", "char", "CHAR(255) NOT NULL DEFAULT ''"},
		{"string", "string", "VARCHAR(255) NOT NULL DEFAULT ''"},
		{"varchar", "varchar", "VARCHAR(255) NOT NULL DEFAULT ''"},
		{"varchar_large", "varchar_large", "VARCHAR(1024) NOT NULL DEFAULT ''"},
		{"varchar_xlarge", "varchar_xlarge", "VARCHAR(5000) NOT NULL DEFAULT ''"},
		{"text", "text", "TEXT"},
		{"mediumtext", "mediumtext", "MEDIUMTEXT"},
		{"longtext", "longtext", "LONGTEXT"},
		{"blob", "blob", "BLOB"},

		// 日期时间类型
		{"date", "date", "DATE NOT NULL DEFAULT '2000-01-01'"},
		{"time", "time", "TIME NOT NULL DEFAULT '00:00:00'"},
		{"datetime", "datetime", "DATETIME NOT NULL DEFAULT '2000-01-01 00:00:00'"},
		{"timestamp", "timestamp", "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"},
		{"timestamp_update", "timestamp_update", "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"},
		{"year", "year", "YEAR NOT NULL DEFAULT 2000"},

		// 特殊类型
		{"boolean", "boolean", "TINYINT(1) NOT NULL DEFAULT 0"},
		{"json", "json", "JSON"},
		{"enum", "enum", "ENUM('active','inactive') NOT NULL DEFAULT 'active'"},

		// 自定义类型（回退）
		{"custom type", "DECIMAL(15,4) NOT NULL", "DECIMAL(15,4) NOT NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := common.Column{Column: "test_col", Type: tt.typeStr}
			result := col.TypeTransform()
			fmt.Printf("Type: %s → %s\n", tt.typeStr, result)
			if result != tt.expected {
				t.Errorf("TypeTransform(%q) = %q, want %q", tt.typeStr, result, tt.expected)
			}
		})
	}
}

func TestIndexDefinition(t *testing.T) {
	tests := []struct {
		name  string
		index common.Index
		want  string
	}{
		{
			name: "primary key",
			index: common.Index{
				Columns:   []string{"id"},
				IsPrimary: true,
				IsUnique:  false,
			},
			want: "PRIMARY KEY (`id`)",
		},
		{
			name: "unique index",
			index: common.Index{
				Name:      "uk_email",
				Columns:   []string{"email"},
				IsPrimary: false,
				IsUnique:  true,
			},
			want: "UNIQUE KEY `uk_email` (`email`)",
		},
		{
			name: "normal index",
			index: common.Index{
				Name:      "idx_status",
				Columns:   []string{"status"},
				IsPrimary: false,
				IsUnique:  false,
			},
			want: "KEY `idx_status` (`status`)",
		},
		{
			name: "composite index",
			index: common.Index{
				Name:      "idx_user_status",
				Columns:   []string{"user_id", "status"},
				IsPrimary: false,
				IsUnique:  false,
			},
			want: "KEY `idx_user_status` (`user_id`,`status`)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.index.IndexDefinition()
			fmt.Printf("Test: %s\nResult: %s\nExpected: %s\n\n", tt.name, result, tt.want)
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mockDatabases() map[string]any {
	return map[string]any{
		"kyogre": []map[string]any{
			{
				"table": "orders",
				"columns": []map[string]any{
					{
						"column": "id",
						"type":   "primary",
					},
					{
						"column": "order_id",
						"type":   "bigint",
					},
					{
						"column": "user_id",
						"type":   "bigint",
					},
					{
						"column": "status",
						"type":   "smallint",
					},
					{
						"column": "create_time",
						"type":   "datetime",
					},
					{
						"column": "update_time",
						"type":   "timestamp_update",
					},
				},
				"indexes": []map[string]any{
					{
						"name": "primary",
						"columns": []string{
							"id",
						},
						"is_primary": true,
						"is_unique":  true,
					}, {
						"name":       "idx_order",
						"columns":    []string{"order_id", "user_id"},
						"is_primary": false,
						"is_unique":  true,
					}, {
						"name":       "idx_create_time",
						"columns":    []string{"create_time"},
						"is_primary": false,
						"is_unique":  false,
					},
				},
			},
			{
				"table": "users",
				"columns": []map[string]any{
					{
						"column": "id",
						"type":   "primary",
					}, {
						"column": "name",
						"type":   "string",
					}, {
						"column": "sex",
						"type":   "smallint",
					}, {
						"column": "age",
						"type":   "int",
					}, {
						"column": "email",
						"type":   "string",
					}, {
						"column": "create_time",
						"type":   "datetime",
					}, {
						"column": "update_time",
						"type":   "timestamp_update",
					},
				},
				"indexes": []map[string]any{
					{
						"name": "primary",
						"columns": []string{
							"id",
						},
						"is_primary": true,
						"is_unique":  true,
					}, {
						"name":       "idx_name",
						"columns":    []string{"name", "sex"},
						"is_primary": false,
						"is_unique":  true,
					}, {
						"name":       "idx_create_time",
						"columns":    []string{"create_time"},
						"is_primary": false,
						"is_unique":  false,
					},
				},
			},
		},
	}
}

func mockMySQLConfig() map[string]any {
	return map[string]any{
		"data-source": mysqlDataSource,
		"databases":   mockDatabases(),
	}
}

func TestMySQLConfigMetadata(t *testing.T) {
	metadata, err := plugin.GetMetadata(mysql_config.MySQL, mysql_config.ConfigMode)
	if err != nil {
		t.Errorf("plugin get metadata err:%v", err)
		return
	}
	if err = metadata.Configure(pipeline, mockMySQLConfig()); err != nil {
		t.Errorf("plugin configure err:%v", err)
		return
	}
	ctx := context.Background()
	t.Run("Initialize", func(t *testing.T) {
		if err = metadata.Initialize(ctx); err != nil {
			t.Errorf("plugin init err:%v", err)
			return
		}
	})

	t.Run("SchemaKeys", func(t *testing.T) {
		keys := metadata.SchemaKeys()
		for _, key := range keys {
			t.Logf("id:%v", key.UniqueID())
		}
	})

	t.Run("SchemaStore", func(t *testing.T) {
		store := metadata.SchemaStore()
		keys := metadata.SchemaKeys()
		for _, key := range keys {
			schema, err := store.GetSchema(key)
			if err != nil {
				t.Errorf("plugin get schema err:%v", err)
				return
			}
			tableDef, ok := schema.(*mysql.Table)
			if !ok {
				t.Errorf("schema transformation *mysql.Table fail")
				return
			}
			t.Logf("database:%s,table:%s primary:%v uniqueIndex:%v", tableDef.Database, tableDef.Table, tableDef.PrimaryIndex, tableDef.UniqueIndex)
			for _, column := range tableDef.Columns {
				t.Logf("column:%v", column)
			}
		}

	})

}

func TestMySQLMockMetadata(t *testing.T) {
	metadata, err := plugin.GetMetadata(mysql_config.MySQL, mysql_mock.MockMode)
	if err != nil {
		t.Errorf("plugin get metadata err:%v", err)
		return
	}
	if err = metadata.Configure(pipeline, mockMySQLConfig()); err != nil {
		t.Errorf("plugin configure err:%v", err)
		return
	}
	t.Run("SchemaKeys", func(t *testing.T) {
		keys := metadata.SchemaKeys()
		for _, key := range keys {
			t.Logf("id:%v", key.UniqueID())
		}
	})

	t.Run("SchemaStore", func(t *testing.T) {
		store := metadata.SchemaStore()
		keys := metadata.SchemaKeys()
		for _, key := range keys {
			schema, err := store.GetSchema(key)
			if err != nil {
				t.Errorf("plugin get schema err:%v", err)
				return
			}
			tableDef, ok := schema.(*mysql.Table)
			if !ok {
				t.Errorf("schema transformation *mysql.Table fail")
				return
			}
			t.Logf("database:%s,table:%s primary:%v uniqueIndex:%v", tableDef.Database, tableDef.Table, tableDef.PrimaryIndex, tableDef.UniqueIndex)
			for _, column := range tableDef.Columns {
				t.Logf("column:%v", column)
			}
		}
	})
}

package common

import (
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

// BuildMockTables 根据数据库配置构建mock的MySQL Table对象
// 用于不连接实际数据库的场景
func BuildMockTables(databases Database) map[string]*mysql_schema.Table {
	mockTables := make(map[string]*mysql_schema.Table)

	for database, tables := range databases {
		for _, table := range tables {
			mockTable := &mysql_schema.Table{
				Database:     database,
				Table:        table.Table,
				Columns:      []mysql_schema.Column{},
				ColumnMap:    make(map[string]mysql_schema.Column),
				PrimaryIndex: []string{},
				UniqueIndex:  make(map[string][]string),
			}

			// 构建列信息
			for i, col := range table.Columns {
				rawType := strings.ToLower(col.TypeTransform())
				mockCol := mysql_schema.Column{
					Name:            col.Column,
					Type:            mysql_schema.ExtractColumnType(rawType), // 设置为默认类型
					RawType:         rawType,
					IsNullable:      true,
					OrdinalPosition: i + 1,
				}

				if strings.Contains(rawType, "unsigned") {
					mockCol.IsUnsigned = true
				} else {
					mockCol.IsUnsigned = false
				}

				if strings.Contains(rawType, "VIRTUAL GENERATED") || strings.Contains(rawType, "STORED GENERATED") {
					mockCol.IsGenerated = true
				}

				if col.Type == Primary {
					mockCol.IsPrimaryKey = true
				} else {
					mockCol.IsPrimaryKey = false
				}

				mockTable.Columns = append(mockTable.Columns, mockCol)
				mockTable.ColumnMap[col.Column] = mockCol
			}

			// 构建索引信息
			for _, idx := range table.Indexes {
				if idx.IsPrimary {
					mockTable.PrimaryIndex = idx.Columns
				}
				if idx.IsUnique {
					mockTable.UniqueIndex[idx.Name] = idx.Columns
				}
			}

			// 初始化扫描列（优先级：主键 > 唯一索引 > 全部列）
			if len(mockTable.PrimaryIndex) > 0 {
				mockTable.SetScanColumns(mockTable.PrimaryIndex)
			} else if len(mockTable.UniqueIndex) > 0 {
				for _, cols := range mockTable.UniqueIndex {
					mockTable.SetScanColumns(cols)
					break
				}
			} else {
				// 如果没有主键和唯一索引，使用所有列
				allCols := []string{}
				for _, col := range mockTable.Columns {
					allCols = append(allCols, col.Name)
				}
				mockTable.SetScanColumns(allCols)
			}

			key := database + "." + table.Table
			mockTables[key] = mockTable
		}
	}

	return mockTables
}

// CreateTableSQLs 生成所有CREATE TABLE和CREATE DATABASE语句
func CreateTableSQLs(databases Database) []string {
	sqlSet := []string{}
	for database, tables := range databases {
		sqlSet = append(sqlSet, "CREATE DATABASE IF NOT EXISTS "+database)
		for _, table := range tables {
			sqlSet = append(sqlSet, table.CreateTableSQL(database))
		}
	}
	return sqlSet
}

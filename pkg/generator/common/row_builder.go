package common

import (
	"fmt"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

// RowBuilder 用于构建 MySQL 行数据的通用工具
type RowBuilder struct {
	dataGen *DataGenerator
}

// NewRowBuilder 创建新的 RowBuilder 实例
func NewRowBuilder() *RowBuilder {
	return &RowBuilder{
		dataGen: NewDataGenerator(),
	}
}

// BuildInsertRow 构建单条 INSERT 行数据
func (rb *RowBuilder) BuildInsertRow(table *mysql_schema.Table, index int) *mysql_schema.RowData {
	return &mysql_schema.RowData{
		Key:       rb.generateRowKey(table, index),
		Data:      rb.generateColumnData(table),
		GuideKeys: rb.generateGuideKeys(table),
	}
}

// BuildUpdateRow 构建单条 UPDATE 行数据（包含 Old 字段）
func (rb *RowBuilder) BuildUpdateRow(table *mysql_schema.Table, index int) *mysql_schema.RowData {
	return &mysql_schema.RowData{
		Key:       rb.generateRowKey(table, index),
		Old:       rb.generateColumnData(table),
		Data:      rb.generateColumnData(table),
		GuideKeys: rb.generateGuideKeys(table),
	}
}

// BuildDeleteRow 构建单条 DELETE 行数据
func (rb *RowBuilder) BuildDeleteRow(table *mysql_schema.Table, index int) *mysql_schema.RowData {
	return &mysql_schema.RowData{
		Key:       rb.generateRowKey(table, index),
		GuideKeys: rb.generateGuideKeys(table),
	}
}

// BuildReplaceRow 构建单条 REPLACE 行数据
func (rb *RowBuilder) BuildReplaceRow(table *mysql_schema.Table, index int) *mysql_schema.RowData {
	return &mysql_schema.RowData{
		Key:       rb.generateRowKey(table, index),
		Data:      rb.generateColumnData(table),
		GuideKeys: rb.generateGuideKeys(table),
	}
}

// BuildInsertIgnoreRow 构建单条 INSERT IGNORE 行数据
func (rb *RowBuilder) BuildInsertIgnoreRow(table *mysql_schema.Table, index int) *mysql_schema.RowData {
	return &mysql_schema.RowData{
		Key:       rb.generateRowKey(table, index),
		Data:      rb.generateColumnData(table),
		GuideKeys: rb.generateGuideKeys(table),
	}
}

// BuildInsertOnDuplicateKeyRow 构建单条 INSERT ON DUPLICATE KEY UPDATE 行数据
func (rb *RowBuilder) BuildInsertOnDuplicateKeyRow(table *mysql_schema.Table, index int) *mysql_schema.RowData {
	return &mysql_schema.RowData{
		Key:       rb.generateRowKey(table, index),
		Data:      rb.generateColumnData(table),
		GuideKeys: rb.generateGuideKeys(table),
	}
}

// BuildRows 批量构建行数据
func (rb *RowBuilder) BuildRows(table *mysql_schema.Table, operation string, count int) []mysql_schema.RowData {
	var rows []mysql_schema.RowData

	for i := 0; i < count; i++ {
		var row *mysql_schema.RowData

		switch operation {
		case "insert":
			row = rb.BuildInsertRow(table, i)
		case "update":
			row = rb.BuildUpdateRow(table, i)
		case "delete":
			row = rb.BuildDeleteRow(table, i)
		case "replace":
			row = rb.BuildReplaceRow(table, i)
		case "insert_ignore":
			row = rb.BuildInsertIgnoreRow(table, i)
		case "insert_on_duplicate_key":
			row = rb.BuildInsertOnDuplicateKeyRow(table, i)
		default:
			row = rb.BuildInsertRow(table, i)
		}

		if row != nil {
			rows = append(rows, *row)
		}
	}

	return rows
}

// generateRowKey 生成行的唯一 Key
func (rb *RowBuilder) generateRowKey(table *mysql_schema.Table, index int) string {
	pkValue := rb.generatePrimaryKeyValue(table, index)
	db, tableName := table.Schema()
	return fmt.Sprintf("%s.%s.%v", db, tableName, pkValue)
}

// generatePrimaryKeyValue 生成主键值
func (rb *RowBuilder) generatePrimaryKeyValue(table *mysql_schema.Table, index int) string {
	if len(table.PrimaryIndex) == 0 {
		return fmt.Sprintf("row_%d", index)
	}
	return fmt.Sprintf("pk_%d", index)
}

// generateColumnData 为表的所有列生成随机数据
func (rb *RowBuilder) generateColumnData(table *mysql_schema.Table) map[string]any {
	data := make(map[string]any)

	for _, col := range table.Columns {
		data[col.Name] = rb.dataGen.GenerateValue(col)
	}

	return data
}

// generateGuideKeys 生成用于 WHERE 条件的主键/唯一键
func (rb *RowBuilder) generateGuideKeys(table *mysql_schema.Table) map[string]any {
	guideKeys := make(map[string]any)

	// 优先使用主键
	if len(table.PrimaryIndex) > 0 {
		for _, pkCol := range table.PrimaryIndex {
			guideKeys[pkCol] = rb.generatePrimaryKeyValue(table, 0)
		}
		return guideKeys
	}

	// 如果没有主键，尝试使用唯一键
	for _, col := range table.Columns {
		if col.ColumnKey == "UNI" || col.ColumnKey == "PRI" {
			guideKeys[col.Name] = rb.dataGen.GenerateValue(col)
			break
		}
	}

	// 如果都没有，使用第一列
	if len(guideKeys) == 0 && len(table.Columns) > 0 {
		col := table.Columns[0]
		guideKeys[col.Name] = rb.dataGen.GenerateValue(col)
	}

	return guideKeys
}

// SetSeed 设置随机数种子（便于测试复现）
func (rb *RowBuilder) SetSeed(seed uint64) {
	rb.dataGen.SetSeed(seed)
}

// GetDataGenerator 获取内部的数据生成器，便于定制
func (rb *RowBuilder) GetDataGenerator() *DataGenerator {
	return rb.dataGen
}

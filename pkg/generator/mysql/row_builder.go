package mysql

import (
	"fmt"
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/pkg/message"
)

// RowBuilder 用于构建 MySQL 行数据的通用工具。
type RowBuilder struct {
	dataGen *DataGenerator
}

// NewRowBuilder 创建新的 RowBuilder 实例。
func NewRowBuilder() *RowBuilder {
	return &RowBuilder{
		dataGen: NewDataGenerator(),
	}
}

// BuildInsertRow 构建单条 INSERT 行数据。
func (rb *RowBuilder) BuildInsertRow(table *mysql_schema.Table, columns []string) *mysql_schema.RowData {
	data := rb.generateColumnData(table, columns)
	key, guideKeys := rb.generateGuideKeysByData(table, data)
	return &mysql_schema.RowData{
		Key:       key,
		Data:      data,
		GuideKeys: guideKeys,
	}
}

// BuildUpdateRow 构建单条 UPDATE 行数据（包含 Old 字段）。
func (rb *RowBuilder) BuildUpdateRow(table *mysql_schema.Table, columns []string) *mysql_schema.RowData {
	data := rb.generateColumnData(table, nil)
	newData := rb.generateNewColumnData(table, data, columns)
	key, guideKeys := rb.generateGuideKeysByData(table, data)
	return &mysql_schema.RowData{
		Key:       key,
		Old:       data,
		Data:      newData,
		GuideKeys: guideKeys,
	}
}

// BuildDeleteRow 构建单条 DELETE 行数据。
func (rb *RowBuilder) BuildDeleteRow(table *mysql_schema.Table) *mysql_schema.RowData {
	key, guideKeys := rb.generateGuideKeys(table)
	return &mysql_schema.RowData{
		Key:       key,
		GuideKeys: guideKeys,
	}
}

// BuildRows 批量构建行数据。
func (rb *RowBuilder) BuildRows(table *mysql_schema.Table, operation string, count int, columns []string) []mysql_schema.RowData {
	var rows []mysql_schema.RowData
	for i := 0; i < count; i++ {
		var row *mysql_schema.RowData
		switch operation {
		case message.Insert, message.InsertIgnore, message.Replace, message.InsertOnDuplicateKey:
			row = rb.BuildInsertRow(table, columns)
		case message.Update:
			row = rb.BuildUpdateRow(table, columns)
		case message.Delete:
			row = rb.BuildDeleteRow(table)
		default:
			row = rb.BuildInsertRow(table, columns)
		}
		if row != nil {
			rows = append(rows, *row)
		}
	}
	return rows
}

// generateColumnData 为表的列生成随机数据。
// columns 为空表示全量列；未包含的列设置 DEFAULT。
func (rb *RowBuilder) generateColumnData(table *mysql_schema.Table, columns []string) map[string]any {
	data := make(map[string]any, len(table.Columns))
	columnSet := make(map[string]struct{}, len(columns))
	for _, name := range columns {
		if name != "" {
			columnSet[name] = struct{}{}
		}
	}
	for _, col := range table.Columns {
		if len(columnSet) > 0 {
			if _, ok := columnSet[col.Name]; !ok {
				data[col.Name] = mysql_schema.DefaultStruct{}
				continue
			}
		}
		data[col.Name] = rb.dataGen.GenerateValue(col)
	}
	return data
}

func (rb *RowBuilder) generateNewColumnData(table *mysql_schema.Table, old map[string]any, columns []string) map[string]any {
	newData := make(map[string]any, len(old))
	for k, v := range old {
		newData[k] = v
	}
	columnSet := make(map[string]struct{}, len(columns))
	for _, name := range columns {
		if name != "" {
			columnSet[name] = struct{}{}
		}
	}
	scanColumns := map[string]struct{}{}
	for _, column := range table.ScanColumns() {
		scanColumns[column] = struct{}{}
	}
	for _, column := range table.Columns {
		if _, isScan := scanColumns[column.Name]; isScan {
			continue
		}
		if len(columnSet) > 0 {
			if _, ok := columnSet[column.Name]; !ok {
				continue
			}
		}
		newData[column.Name] = rb.dataGen.GenerateValue(column)
	}
	return newData
}

func (rb *RowBuilder) generateGuideKeysByData(table *mysql_schema.Table, data map[string]any) (string, map[string]any) {
	scanColumns := table.ScanColumns()
	guideKeys := map[string]any{}
	pksList := make([]string, 0, len(scanColumns))
	for _, column := range scanColumns {
		value, ok := data[column]
		if !ok {
			return "", nil
		}
		guideKeys[column] = value
		pksList = append(pksList, fmt.Sprintf("%s.%v", column, value))
	}
	key := mysql_schema.MakeRowKey(table.Database, table.Table, strings.Join(pksList, ","))
	return key, guideKeys
}

// generateGuideKeys 生成用于 WHERE 条件的主键/唯一键。
func (rb *RowBuilder) generateGuideKeys(table *mysql_schema.Table) (string, map[string]any) {
	guideKeys := make(map[string]any)
	pksList := make([]string, 0, len(table.ScanColumns()))
	for _, column := range table.ScanColumns() {
		col := table.ColumnMap[column]
		value := rb.dataGen.GenerateValue(col)
		guideKeys[column] = value
		pksList = append(pksList, fmt.Sprintf("%s.%v", column, value))
	}
	return mysql_schema.MakeRowKey(table.Database, table.Table, strings.Join(pksList, ",")), guideKeys
}

// SetSeed 设置随机数种子（便于测试复现）。
func (rb *RowBuilder) SetSeed(seed uint64) {
	rb.dataGen.SetSeed(seed)
}

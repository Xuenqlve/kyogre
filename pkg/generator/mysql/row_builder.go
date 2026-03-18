package mysql

import (
	"fmt"
	"reflect"
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/message"
)

// RowBuilder 用于构建 MySQL 行数据的通用工具。
type RowBuilder struct {
	dataGen   *DataGenerator
	snapshot  *generator.StrategySnapshot
	providers map[string]iquery.Provider
}

// NewRowBuilder 创建新的 RowBuilder 实例。
func NewRowBuilder(snapshot *generator.StrategySnapshot, providers map[string]iquery.Provider) *RowBuilder {
	return &RowBuilder{
		dataGen:   NewDataGenerator(),
		snapshot:  snapshot,
		providers: providers,
	}
}

// BuildInsertRow 构建单条 INSERT 行数据。
func (rb *RowBuilder) BuildInsertRow(table *mysql_schema.Table, columns []string, providerRow map[string]any) (*mysql_schema.RowData, error) {
	data := rb.generateColumnData(table, columns)
	if err := rb.mergeProviderRow(table, data, providerRow); err != nil {
		return nil, err
	}
	rb.applyStrategy(data)
	key, guideKeys, err := rb.generateGuideKeysByData(table, data)
	if err != nil {
		return nil, err
	}
	return &mysql_schema.RowData{
		Key:       key,
		Data:      data,
		GuideKeys: guideKeys,
	}, nil
}

// BuildUpdateRow 构建单条 UPDATE 行数据（包含 Old 字段）。
func (rb *RowBuilder) BuildUpdateRow(table *mysql_schema.Table, columns []string, providerRow map[string]any, updateGuideKeys bool) (*mysql_schema.RowData, error) {
	oldData := rb.generateColumnData(table, nil)
	if err := rb.mergeProviderRow(table, oldData, providerRow); err != nil {
		return nil, err
	}
	newData := rb.generateNewColumnData(table, oldData, columns)
	rb.applyStrategy(newData)
	if !updateGuideKeys {
		if err := rb.ensureGuideKeysUnchanged(table, oldData, newData); err != nil {
			return nil, err
		}
	}
	keySource := oldData
	if providerRow != nil {
		keySource = providerRow
	} else if updateGuideKeys {
		keySource = newData
	}
	key, guideKeys, err := rb.generateGuideKeysByData(table, keySource)
	if err != nil {
		return nil, err
	}
	return &mysql_schema.RowData{
		Key:       key,
		Old:       oldData,
		Data:      newData,
		GuideKeys: guideKeys,
	}, nil
}

// BuildDeleteRow 构建单条 DELETE 行数据。
func (rb *RowBuilder) BuildDeleteRow(table *mysql_schema.Table, providerRow map[string]any) (*mysql_schema.RowData, error) {
	if providerRow != nil {
		key, guideKeys, err := rb.generateGuideKeysByData(table, providerRow)
		if err != nil {
			return nil, err
		}
		return &mysql_schema.RowData{
			Key:       key,
			GuideKeys: guideKeys,
		}, nil
	}
	key, guideKeys, err := rb.generateGuideKeys(table)
	if err != nil {
		return nil, err
	}
	return &mysql_schema.RowData{
		Key:       key,
		GuideKeys: guideKeys,
	}, nil
}

// BuildRows 批量构建行数据。
func (rb *RowBuilder) BuildRows(table *mysql_schema.Table, operation string, count int, columns []string) ([]mysql_schema.RowData, error) {
	provider, err := rb.providerForTable(table)
	if err != nil {
		return nil, err
	}
	var providerRows []map[string]any
	if provider != nil {
		if err := rb.validateProviderColumns(table, provider); err != nil {
			return nil, err
		}
		providerRows, err = rb.providerRows(provider, count)
		if err != nil {
			return nil, err
		}
	}
	rows := make([]mysql_schema.RowData, 0, count)
	for i := 0; i < count; i++ {
		var providerRow map[string]any
		if providerRows != nil {
			providerRow = providerRows[i]
		}
		var row *mysql_schema.RowData
		var rowErr error
		switch operation {
		case message.Insert, message.InsertIgnore, message.Replace, message.InsertOnDuplicateKey:
			row, rowErr = rb.BuildInsertRow(table, columns, providerRow)
		case message.Update:
			row, rowErr = rb.BuildUpdateRow(table, columns, providerRow, false)
		case message.Delete:
			row, rowErr = rb.BuildDeleteRow(table, providerRow)
		default:
			return nil, fmt.Errorf("unsupported operation: %s", operation)
		}
		if rowErr != nil {
			return nil, rowErr
		}
		if row != nil {
			rows = append(rows, *row)
		}
	}
	return rows, nil
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

func (rb *RowBuilder) generateGuideKeysByData(table *mysql_schema.Table, data map[string]any) (string, map[string]any, error) {
	scanColumns := table.ScanColumns()
	if len(scanColumns) == 0 {
		return "", nil, fmt.Errorf("scan columns are empty")
	}
	guideKeys := map[string]any{}
	pksList := make([]string, 0, len(scanColumns))
	for _, column := range scanColumns {
		value, ok := data[column]
		if !ok {
			return "", nil, fmt.Errorf("missing scan column %q in data", column)
		}
		guideKeys[column] = value
		pksList = append(pksList, fmt.Sprintf("%s.%v", column, value))
	}
	key := mysql_schema.MakeRowKey(table.Database, table.Table, strings.Join(pksList, ","))
	return key, guideKeys, nil
}

// generateGuideKeys 生成用于 WHERE 条件的主键/唯一键。
func (rb *RowBuilder) generateGuideKeys(table *mysql_schema.Table) (string, map[string]any, error) {
	if table == nil {
		return "", nil, fmt.Errorf("table is nil")
	}
	guideKeys := make(map[string]any)
	scanColumns := table.ScanColumns()
	if len(scanColumns) == 0 {
		return "", nil, fmt.Errorf("scan columns are empty")
	}
	pksList := make([]string, 0, len(scanColumns))
	for _, column := range scanColumns {
		col, ok := table.Column(column)
		if !ok {
			return "", nil, fmt.Errorf("scan column %q not found in table", column)
		}
		value := rb.dataGen.GenerateValue(col)
		guideKeys[column] = value
		pksList = append(pksList, fmt.Sprintf("%s.%v", column, value))
	}
	return mysql_schema.MakeRowKey(table.Database, table.Table, strings.Join(pksList, ",")), guideKeys, nil
}

//// SetSeed 设置随机数种子（便于测试复现）。
//func (rb *RowBuilder) SetSeed(seed uint64) {
//	rb.dataGen.SetSeed(seed)
//}

func (rb *RowBuilder) providerForTable(table *mysql_schema.Table) (iquery.Provider, error) {
	if table == nil {
		return nil, fmt.Errorf("table is nil")
	}
	if rb.providers == nil {
		return nil, nil
	}
	index := table.Index()
	key := index.UniqueID()
	provider, ok := rb.providers[key]
	if ok && provider == nil {
		return nil, fmt.Errorf("provider is nil for table %s", key)
	}
	return provider, nil
}

func (rb *RowBuilder) providerRows(provider iquery.Provider, count int) ([]map[string]any, error) {
	rows := provider.Rows()
	if len(rows) == 0 {
		return nil, fmt.Errorf("provider rows are empty")
	}
	if count > len(rows) {
		return nil, fmt.Errorf("provider rows not enough: need %d, got %d", count, len(rows))
	}
	return rows, nil
}

func (rb *RowBuilder) validateProviderColumns(table *mysql_schema.Table, provider iquery.Provider) error {
	if table == nil {
		return fmt.Errorf("table is nil")
	}
	scanColumns := table.ScanColumns()
	if len(scanColumns) == 0 {
		return fmt.Errorf("scan columns are empty")
	}
	pCols := provider.Columns()
	if len(pCols) == 0 {
		return fmt.Errorf("provider columns are empty")
	}
	scanSet := make(map[string]struct{}, len(scanColumns))
	for _, col := range scanColumns {
		if col == "" {
			continue
		}
		scanSet[col] = struct{}{}
	}
	providerSet := make(map[string]struct{}, len(pCols))
	for _, col := range pCols {
		if col == "" {
			continue
		}
		providerSet[col] = struct{}{}
	}
	if len(scanSet) != len(providerSet) {
		return fmt.Errorf("provider columns mismatch scan columns")
	}
	for col := range scanSet {
		if _, ok := providerSet[col]; !ok {
			return fmt.Errorf("provider columns missing scan column %q", col)
		}
	}
	return nil
}

func (rb *RowBuilder) mergeProviderRow(table *mysql_schema.Table, data map[string]any, providerRow map[string]any) error {
	if providerRow == nil {
		return nil
	}
	scanColumns := table.ScanColumns()
	if len(scanColumns) == 0 {
		return fmt.Errorf("scan columns are empty")
	}
	for _, col := range scanColumns {
		val, ok := providerRow[col]
		if !ok {
			return fmt.Errorf("provider row missing scan column %q", col)
		}
		data[col] = val
	}
	return nil
}

func (rb *RowBuilder) applyStrategy(data map[string]any) {
	if rb.snapshot == nil || data == nil {
		return
	}
	filtered := rb.snapshot.FilterFields(mapKeys(data))
	if len(filtered) == 0 && len(data) > 0 {
		for key := range data {
			delete(data, key)
		}
		return
	}
	keep := make(map[string]struct{}, len(filtered))
	for _, key := range filtered {
		keep[key] = struct{}{}
	}
	for key := range data {
		if _, ok := keep[key]; !ok {
			delete(data, key)
		}
	}
	rb.snapshot.ApplyAll(data)
}

func mapKeys(row map[string]any) []string {
	if len(row) == 0 {
		return nil
	}
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	return keys
}

func (rb *RowBuilder) ensureGuideKeysUnchanged(table *mysql_schema.Table, oldData, newData map[string]any) error {
	scanColumns := table.ScanColumns()
	if len(scanColumns) == 0 {
		return fmt.Errorf("scan columns are empty")
	}
	for _, col := range scanColumns {
		oldVal, ok := oldData[col]
		if !ok {
			return fmt.Errorf("old data missing scan column %q", col)
		}
		newVal, ok := newData[col]
		if !ok {
			return fmt.Errorf("new data missing scan column %q", col)
		}
		if !reflect.DeepEqual(oldVal, newVal) {
			return fmt.Errorf("scan column %q updated while updateGuideKeys disabled", col)
		}
	}
	return nil
}

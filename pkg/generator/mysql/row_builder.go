package mysql

//// RowBuilder 用于构建 MySQL 行数据的通用工具
//type RowBuilder struct {
//	dataGen *DataGenerator
//}
//
//// NewRowBuilder 创建新的 RowBuilder 实例
//func NewRowBuilder() *RowBuilder {
//	return &RowBuilder{
//		dataGen: NewDataGenerator(),
//	}
//}
//
//// BuildInsertRow 构建单条 INSERT 行数据
//func (rb *RowBuilder) BuildInsertRow(table *mysql_schema.Table) *mysql_schema.RowData {
//	data := rb.generateColumnData(table)
//	key, guideKeys := rb.generateGuideKeysByData(table, data)
//	return &mysql_schema.RowData{
//		Key:       key,
//		Data:      data,
//		GuideKeys: guideKeys,
//	}
//}
//
//// BuildUpdateRow 构建单条 UPDATE 行数据（包含 Old 字段）
//func (rb *RowBuilder) BuildUpdateRow(table *mysql_schema.Table) *mysql_schema.RowData {
//	data := rb.generateColumnData(table)
//	newData := rb.generateNewColumnData(table, data, rb.dataGen.GenerateRandomInt(len(table.Columns)))
//	key, guideKeys := rb.generateGuideKeysByData(table, data)
//	return &mysql_schema.RowData{
//		Key:       key,
//		Old:       data,
//		Data:      newData,
//		GuideKeys: guideKeys,
//	}
//}
//
//// BuildDeleteRow 构建单条 DELETE 行数据
//func (rb *RowBuilder) BuildDeleteRow(table *mysql_schema.Table) *mysql_schema.RowData {
//	key, guideKeys := rb.generateGuideKeys(table)
//	return &mysql_schema.RowData{
//		Key:       key,
//		GuideKeys: guideKeys,
//	}
//}
//
//// BuildRows 批量构建行数据
//func (rb *RowBuilder) BuildRows(table *mysql_schema.Table, operation string, count int) []mysql_schema.RowData {
//	var rows []mysql_schema.RowData
//
//	for i := 0; i < count; i++ {
//		var row *mysql_schema.RowData
//
//		switch operation {
//		case message.Insert, message.InsertIgnore, message.Replace, message.InsertOnDuplicateKey:
//			row = rb.BuildInsertRow(table)
//		case message.Update:
//			row = rb.BuildUpdateRow(table)
//		case message.Delete:
//			row = rb.BuildDeleteRow(table)
//		default:
//			row = rb.BuildInsertRow(table)
//		}
//		if row != nil {
//			rows = append(rows, *row)
//		}
//	}
//	return rows
//}
//
//// generateColumnData 为表的所有列生成随机数据
//func (rb *RowBuilder) generateColumnData(table *mysql_schema.Table) map[string]any {
//	data := make(map[string]any)
//	for _, col := range table.Columns {
//		data[col.Name] = rb.dataGen.GenerateValue(col)
//	}
//
//	return data
//}
//
//func (rb *RowBuilder) generateNewColumnData(table *mysql_schema.Table, old map[string]any, updateCount int) map[string]any {
//	// 复制旧数据到新的 map
//	newData := make(map[string]any)
//	for k, v := range old {
//		newData[k] = v
//	}
//	scanColumns := map[string]struct{}{}
//	for _, column := range table.ScanColumns() {
//		scanColumns[column] = struct{}{}
//	}
//
//	// 确保 updateCount 不超过列数
//	if updateCount > len(table.Columns)-len(table.ScanColumns()) {
//		updateCount = len(table.Columns) - len(table.ScanColumns())
//	}
//
//	// 用于追踪已更新的列，避免重复
//	updatedColumns := make(map[int]bool)
//
//	// 随机选择 updateCount 列进行更新
//	for i := 0; i < updateCount; i++ {
//		var index int
//		// 确保不会更新同一列两次
//		for {
//			index = rb.dataGen.GenerateRandomInt(len(table.Columns))
//			if !updatedColumns[index] {
//				updatedColumns[index] = true
//				break
//			}
//		}
//
//		column := table.Columns[index]
//		if _, exist := scanColumns[column.Name]; exist {
//			continue
//		}
//		newValue := rb.dataGen.GenerateValue(column)
//		newData[column.Name] = newValue
//	}
//
//	return newData
//}
//
//func (rb *RowBuilder) generateGuideKeysByData(table *mysql_schema.Table, data map[string]any) (string, map[string]any) {
//	scanColumns := table.ScanColumns()
//	guideKeys := map[string]any{}
//	pksList := make([]string, 0, len(scanColumns))
//
//	for _, column := range scanColumns {
//		value := data[column]
//		guideKeys[column] = value
//		pksList = append(pksList, fmt.Sprintf("%s.%v", column, value))
//	}
//	key := mysql_schema.MakeRowKey(table.Database, table.Table, strings.Join(pksList, ","))
//	return key, guideKeys
//}
//
//// generateGuideKeys 生成用于 WHERE 条件的主键/唯一键
//func (rb *RowBuilder) generateGuideKeys(table *mysql_schema.Table) (string, map[string]any) {
//	guideKeys := make(map[string]any)
//	pksList := make([]string, 0, len(table.ScanColumns()))
//	for _, column := range table.ScanColumns() {
//		col := table.ColumnMap[column]
//		value := rb.dataGen.GenerateValue(col)
//		guideKeys[column] = value
//		pksList = append(pksList, fmt.Sprintf("%s.%v", column, value))
//	}
//	return mysql_schema.MakeRowKey(table.Database, table.Table, strings.Join(pksList, ",")), guideKeys
//}
//
//// SetSeed 设置随机数种子（便于测试复现）
//func (rb *RowBuilder) SetSeed(seed uint64) {
//	rb.dataGen.SetSeed(seed)
//}

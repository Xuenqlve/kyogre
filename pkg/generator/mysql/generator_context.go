package mysql

//const (
//	ModeDMLRow      generator.GenerationMode = "dml_row"      // 单表行级DML
//	ModeDMLBatch    generator.GenerationMode = "dml_batch"    // 单表批量DML
//	ModeDMLSequence generator.GenerationMode = "dml_sequence" // 单表序列化DML
//	ModeDDL         generator.GenerationMode = "ddl"          // DDL操作
//	ModeTransaction generator.GenerationMode = "transaction"  // 多表事务
//)
//
//const (
//	DML         string = "dml"
//	Transaction string = "transaction"
//	DDL         string = "ddl"
//)
//
//// DMLConfig DML操作的配置
//type DMLConfig struct {
//	Operation   string // insert/update/delete/select
//	TableSelect string // random/ordered/same_with_last/diff_from_last
//	Count       string // 生成行数，可以是数字或"random"
//	WriteType   string // insert/replace/insert_ignore/insert_on_duplicate_key (可选)
//}
//
//func (c *DMLConfig) Type() string {
//	return DML
//}
//
//func (c *DMLConfig) Validate() error {
//	if c.Operation == "" {
//		return fmt.Errorf("operation is required")
//	}
//	if c.TableSelect == "" {
//		c.TableSelect = generator.RandomTableSelect
//	}
//	if c.Count == "" {
//		c.Count = "1"
//	}
//	return nil
//}
//
//// DMLDependency DML操作的依赖条件
//type DMLDependency struct {
//	Mode        generator.GenerationMode // 生成模式
//	Table       *mysql.Table             // 目标表
//	Operation   string                   // 操作类型：insert/update/delete/select
//	WriteType   string                   // 写入类型：insert/replace/insert_ignore/insert_on_duplicate_key
//	Count       int                      // 生成数据条数
//	FieldFilter []string                 // 只生成指定字段（nil表示全部）
//}
//
//func (d *DMLDependency) DependencyType() string {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (d *DMLDependency) GetTables() []any {
//	return []any{d.Table}
//}
//
//func (d *DMLDependency) GetFields() []string {
//	if len(d.FieldFilter) > 0 {
//		return d.FieldFilter
//	}
//	// 返回表的所有字段
//	if d.Table == nil {
//		return []string{}
//	}
//	// 假设Table有这个方法，如果没有可以改为返回空列表
//	// 实际实现时需要根据Table的API来获取列
//	return []string{}
//}
//
//// 获取此生成所需的表列表
//func (d *DMLDependency) GetSchemas() []schema_store.SchemaKey {
//	return nil
//}
//
//// 获取生成模式
//
//func (d *DMLDependency) GetMode() generator.GenerationMode {
//	return d.Mode
//}
//
//func (d *DMLDependency) Validate() error {
//	if d.Table == nil {
//		return fmt.Errorf("table is required")
//	}
//	if d.Count <= 0 {
//		return fmt.Errorf("count must be greater than 0")
//	}
//	if d.Operation == "" {
//		return fmt.Errorf("operation is required")
//	}
//	return nil
//}
//
//func (d *DMLDependency) Type() string {
//	return DML
//}
//
//// TransactionConfig 事务操作的配置
//type TransactionConfig struct {
//	Operations map[string]TransactionOp // key: table_name
//}
//
//type TransactionOp struct {
//	Operation string // insert/update/delete
//	Count     int    // 每张表生成多少行
//	WriteType string
//}
//
//func (c *TransactionConfig) Type() string {
//	return Transaction
//}
//
//func (c *TransactionConfig) Validate() error {
//	if len(c.Operations) == 0 {
//		return fmt.Errorf("operations is required")
//	}
//	for tableName, op := range c.Operations {
//		if op.Count <= 0 {
//			return fmt.Errorf("operation count for table %s must be greater than 0", tableName)
//		}
//	}
//	return nil
//}
//
//// TransactionDependency 事务操作的依赖条件
//type TransactionDependency struct {
//	Tables     []*mysql.Table           // 涉及的多张表
//	Operations map[string]TransactionOp // key: table_name, value: 操作
//}
//
//func (d *TransactionDependency) GetSchemas() []schema_store.SchemaKey {
//	return nil
//}
//
//func (d *TransactionDependency) GetFields() []string {
//	// 事务可能涉及多张表的不同字段组合
//	return []string{}
//}
//
//func (d *TransactionDependency) GetMode() generator.GenerationMode {
//	return ModeTransaction
//}
//
//func (d *TransactionDependency) Validate() error {
//	if len(d.Tables) == 0 {
//		return fmt.Errorf("at least one table is required")
//	}
//	if len(d.Operations) == 0 {
//		return fmt.Errorf("at least one operation is required")
//	}
//	for _, op := range d.Operations {
//		if op.Count <= 0 {
//			return fmt.Errorf("count must be greater than 0")
//		}
//	}
//	return nil
//}
//
//func (d *TransactionDependency) DependencyType() string {
//	return Transaction
//}
//
//// DDLConfig DDL操作的配置
//type DDLConfig struct {
//	DDLType string // ALTER/RENAME/CREATE/DROP
//	Columns []ColumnChange
//}
//
//func (c *DDLConfig) Type() string {
//	return DDL
//}
//
//func (c *DDLConfig) Validate() error {
//	if c.DDLType == "" {
//		return fmt.Errorf("ddl type is required")
//	}
//	return nil
//}
//
//// DDLDependency DDL操作的依赖条件
//type DDLDependency struct {
//	Table   *mysql.Table   // 目标表
//	DDLType string         // ALTER/RENAME/CREATE/DROP
//	Columns []ColumnChange // 列修改信息
//}
//
//type ColumnChange struct {
//	Name   string
//	Type   string
//	Change string // 修改类型
//}
//
//func (d *DDLDependency) GetSchemas() []schema_store.SchemaKey {
//	return nil
//}
//
//func (d *DDLDependency) GetFields() []string {
//	return []string{}
//}
//
//func (d *DDLDependency) GetMode() generator.GenerationMode {
//	return ModeDDL
//}
//
//func (d *DDLDependency) Validate() error {
//	if d.Table == nil {
//		return fmt.Errorf("table is required")
//	}
//	if d.DDLType == "" {
//		return fmt.Errorf("ddl type is required")
//	}
//	return nil
//}
//
//func (d *DDLDependency) DependencyType() string {
//	return DDL
//}
//
//// ParseDependency 将参数解析为具体的Dependency
//func ParseDependency(depType string, data map[string]any) (generator.GenerationDependency, error) {
//	switch depType {
//	case "dml":
//		return parseDMLDependency(data)
//	case "transaction":
//		return parseTransactionDependency(data)
//	case "ddl":
//		return parseDDLDependency(data)
//	default:
//		return nil, fmt.Errorf("unknown dependency type: %s", depType)
//	}
//}
//
//func parseDMLDependency(data map[string]any) (generator.GenerationDependency, error) {
//	// 这里需要根据data反序列化为DMLDependency
//	// 简化实现，实际使用时可以用mapstructure
//	dep := &DMLDependency{}
//
//	if mode, ok := data["mode"].(string); ok {
//		dep.Mode = generator.GenerationMode(mode)
//	}
//
//	if operation, ok := data["operation"].(string); ok {
//		dep.Operation = operation
//	}
//
//	if writeType, ok := data["write_type"].(string); ok {
//		dep.WriteType = writeType
//	}
//
//	if count, ok := data["count"].(int); ok {
//		dep.Count = count
//	} else if count, ok := data["count"].(float64); ok {
//		dep.Count = int(count)
//	}
//
//	return dep, nil
//}
//
//func parseTransactionDependency(data map[string]any) (generator.GenerationDependency, error) {
//	dep := &TransactionDependency{}
//	dep.Operations = make(map[string]TransactionOp)
//	return dep, nil
//}
//
//func parseDDLDependency(data map[string]any) (generator.GenerationDependency, error) {
//	dep := &DDLDependency{}
//
//	if ddlType, ok := data["ddl_type"].(string); ok {
//		dep.DDLType = ddlType
//	}
//
//	return dep, nil
//}

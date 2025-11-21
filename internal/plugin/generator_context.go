package plugin

import (
	"fmt"

	"github.com/xuenqlve/common/relational_database/mysql"
)

// DependencyConfig 依赖配置接口
// 所有依赖配置必须实现此接口，确保可验证性和类型安全性
// 这个接口规范了Scenario(生产者)和Generator(消费者)之间的契约
type DependencyConfig interface {
	// Validate 验证配置的有效性
	Validate() error

	// Type 返回配置类型标识（用于类型断言和日志记录）
	Type() string
}

// DMLConfig DML操作的配置
type DMLConfig struct {
	Operation   string // insert/update/delete/select
	TableSelect string // random/ordered/same_with_last/diff_from_last
	Count       string // 生成行数，可以是数字或"random"
	WriteType   string // insert/replace/insert_ignore/insert_on_duplicate_key (可选)
}

func (c *DMLConfig) Type() string {
	return "dml"
}

func (c *DMLConfig) Validate() error {
	if c.Operation == "" {
		return fmt.Errorf("operation is required")
	}
	if c.TableSelect == "" {
		c.TableSelect = RandomTableSelect
	}
	if c.Count == "" {
		c.Count = "1"
	}
	return nil
}

// TransactionConfig 事务操作的配置
type TransactionConfig struct {
	Operations map[string]TransactionOp // key: table_name
}

type TransactionOp struct {
	Operation string   // insert/update/delete
	Count     int      // 每张表生成多少行
	WriteType string
}

func (c *TransactionConfig) Type() string {
	return "transaction"
}

func (c *TransactionConfig) Validate() error {
	if len(c.Operations) == 0 {
		return fmt.Errorf("operations is required")
	}
	for tableName, op := range c.Operations {
		if op.Count <= 0 {
			return fmt.Errorf("operation count for table %s must be greater than 0", tableName)
		}
	}
	return nil
}

// DDLConfig DDL操作的配置
type DDLConfig struct {
	DDLType string        // ALTER/RENAME/CREATE/DROP
	Columns []ColumnChange
}

func (c *DDLConfig) Type() string {
	return "ddl"
}

func (c *DDLConfig) Validate() error {
	if c.DDLType == "" {
		return fmt.Errorf("ddl type is required")
	}
	return nil
}

// GenerationMode 生成模式
type GenerationMode string

const (
	ModeDMLRow        GenerationMode = "dml_row"        // 单表行级DML
	ModeDMLBatch      GenerationMode = "dml_batch"      // 单表批量DML
	ModeDMLSequence   GenerationMode = "dml_sequence"   // 单表序列化DML
	ModeDDL           GenerationMode = "ddl"            // DDL操作
	ModeTransaction   GenerationMode = "transaction"    // 多表事务
)

// GenerationDependency 生成数据的依赖条件接口
type GenerationDependency interface {
	// 获取此生成所需的表列表
	GetTables() []*mysql.Table

	// 获取此生成所需的字段列表
	GetFields() []string

	// 获取生成模式
	GetMode() GenerationMode

	// 验证依赖条件是否完整
	Validate() error

	// 获取依赖类型名称（用于序列化）
	DependencyType() string
}

// DMLDependency DML操作的依赖条件
type DMLDependency struct {
	Mode        GenerationMode  // 生成模式
	Table       *mysql.Table    // 目标表
	Operation   string          // 操作类型：insert/update/delete/select
	WriteType   string          // 写入类型：insert/replace/insert_ignore/insert_on_duplicate_key
	Count       int             // 生成数据条数
	FieldFilter []string        // 只生成指定字段（nil表示全部）
}

func (d *DMLDependency) GetTables() []*mysql.Table {
	return []*mysql.Table{d.Table}
}

func (d *DMLDependency) GetFields() []string {
	if len(d.FieldFilter) > 0 {
		return d.FieldFilter
	}
	// 返回表的所有字段
	if d.Table == nil {
		return []string{}
	}
	// 假设Table有这个方法，如果没有可以改为返回空列表
	// 实际实现时需要根据Table的API来获取列
	return []string{}
}

func (d *DMLDependency) GetMode() GenerationMode {
	return d.Mode
}

func (d *DMLDependency) Validate() error {
	if d.Table == nil {
		return fmt.Errorf("table is required")
	}
	if d.Count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}
	if d.Operation == "" {
		return fmt.Errorf("operation is required")
	}
	return nil
}

func (d *DMLDependency) DependencyType() string {
	return "dml"
}

// TransactionDependency 事务操作的依赖条件
type TransactionDependency struct {
	Tables     []*mysql.Table              // 涉及的多张表
	Operations map[string]TransactionOp    // key: table_name, value: 操作
}

func (d *TransactionDependency) GetTables() []*mysql.Table {
	return d.Tables
}

func (d *TransactionDependency) GetFields() []string {
	// 事务可能涉及多张表的不同字段组合
	return []string{}
}

func (d *TransactionDependency) GetMode() GenerationMode {
	return ModeTransaction
}

func (d *TransactionDependency) Validate() error {
	if len(d.Tables) == 0 {
		return fmt.Errorf("at least one table is required")
	}
	if len(d.Operations) == 0 {
		return fmt.Errorf("at least one operation is required")
	}
	for _, op := range d.Operations {
		if op.Count <= 0 {
			return fmt.Errorf("count must be greater than 0")
		}
	}
	return nil
}

func (d *TransactionDependency) DependencyType() string {
	return "transaction"
}

// DDLDependency DDL操作的依赖条件
type DDLDependency struct {
	Table      *mysql.Table    // 目标表
	DDLType    string          // ALTER/RENAME/CREATE/DROP
	Columns    []ColumnChange  // 列修改信息
}

type ColumnChange struct {
	Name   string
	Type   string
	Change string // 修改类型
}

func (d *DDLDependency) GetTables() []*mysql.Table {
	return []*mysql.Table{d.Table}
}

func (d *DDLDependency) GetFields() []string {
	return []string{}
}

func (d *DDLDependency) GetMode() GenerationMode {
	return ModeDDL
}

func (d *DDLDependency) Validate() error {
	if d.Table == nil {
		return fmt.Errorf("table is required")
	}
	if d.DDLType == "" {
		return fmt.Errorf("ddl type is required")
	}
	return nil
}

func (d *DDLDependency) DependencyType() string {
	return "ddl"
}

// GenerationStrategy 生成策略配置
type GenerationStrategy struct {
	SequenceConfig  *SequenceConfig   `json:"sequence_config,omitempty"`   // 序列化配置
	RandomConfig    *RandomConfig     `json:"random_config,omitempty"`     // 随机配置
	TemplateConfig  *TemplateConfig   `json:"template_config,omitempty"`   // 模板配置
	CustomConfig    map[string]any    `json:"custom_config,omitempty"`     // 自定义配置
}

// SequenceConfig 序列化生成配置
type SequenceConfig struct {
	Enabled      bool   `json:"enabled"`
	Field        string `json:"field"`          // 递增字段
	StartValue   int64  `json:"start_value"`
	EndValue     int64  `json:"end_value"`
	CurrentValue int64  `json:"current_value"`  // 由Worker维护
	Step         int64  `json:"step"`           // 递增步长，默认1
}

// RandomConfig 随机生成配置
type RandomConfig struct {
	Seed int64  `json:"seed,omitempty"`
}

// TemplateConfig 模板生成配置（基于预设模板）
type TemplateConfig struct {
	TemplateName string         `json:"template_name"`
	TemplateData map[string]any `json:"template_data"`
}

// ParseDependency 将参数解析为具体的Dependency
func ParseDependency(depType string, data map[string]any) (GenerationDependency, error) {
	switch depType {
	case "dml":
		return parseDMLDependency(data)
	case "transaction":
		return parseTransactionDependency(data)
	case "ddl":
		return parseDDLDependency(data)
	default:
		return nil, fmt.Errorf("unknown dependency type: %s", depType)
	}
}

func parseDMLDependency(data map[string]any) (GenerationDependency, error) {
	// 这里需要根据data反序列化为DMLDependency
	// 简化实现，实际使用时可以用mapstructure
	dep := &DMLDependency{}

	if mode, ok := data["mode"].(string); ok {
		dep.Mode = GenerationMode(mode)
	}

	if operation, ok := data["operation"].(string); ok {
		dep.Operation = operation
	}

	if writeType, ok := data["write_type"].(string); ok {
		dep.WriteType = writeType
	}

	if count, ok := data["count"].(int); ok {
		dep.Count = count
	} else if count, ok := data["count"].(float64); ok {
		dep.Count = int(count)
	}

	return dep, nil
}

func parseTransactionDependency(data map[string]any) (GenerationDependency, error) {
	dep := &TransactionDependency{}
	dep.Operations = make(map[string]TransactionOp)
	return dep, nil
}

func parseDDLDependency(data map[string]any) (GenerationDependency, error) {
	dep := &DDLDependency{}

	if ddlType, ok := data["ddl_type"].(string); ok {
		dep.DDLType = ddlType
	}

	return dep, nil
}

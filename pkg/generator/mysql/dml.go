package mysql

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

const MySQLDML generator.Type = "mysql-dml"

func init() {
	generator.RegisterGenerator(MySQLDML, &DMLGenerator{}, false)
}

type Config struct {
	Metadata string `mapstructure:"metadata" json:"metadata"`
}

type DMLGenerator struct {
	pipeline   string
	metadata   metadata.Metadata
	config     Config
	hitIndex   int
	rowBuilder *RowBuilder
	// 用于支持序列化生成时维护当前值
	sequenceStates map[string]int64
	seqMutex       sync.RWMutex
}

func (g *DMLGenerator) Configure(pipeline string, data map[string]any) (err error) {
	g.pipeline = pipeline
	if err = mapstructure.Decode(data, g.config); err != nil {
		return err
	}
	g.hitIndex = 0
	g.rowBuilder = NewRowBuilder()
	g.sequenceStates = make(map[string]int64)
	return nil
}

func (g *DMLGenerator) RegisterMetadata(metadata metadata.Metadata) {
	g.metadata = metadata
}

// CollectDependencies 第一阶段：收集生成所需的依赖条件
func (g *DMLGenerator) CollectDependencies(req *generator.DependencyRequest) (generator.GenerationDependency, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if req.Config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	// 验证配置
	if err := req.Config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 根据配置类型处理不同的依赖
	switch config := req.Config.(type) {
	case *DMLConfig:
		return g.collectDMLDependency(config)
	case *TransactionConfig:
		return g.collectTransactionDependency(config)
	case *DDLConfig:
		return g.collectDDLDependency(config)
	default:
		return nil, fmt.Errorf("unsupported config type: %T", req.Config)
	}
}

// collectDMLDependency 收集DML操作的依赖条件
func (g *DMLGenerator) collectDMLDependency(dmlConfig *DMLConfig) (generator.GenerationDependency, error) {
	// 解析行数
	count, err := ParseCount(dmlConfig.Count)
	if err != nil {
		count = 1
	}

	// 获取要操作的表定义
	tableDef, err := g.getTableDef(dmlConfig.TableSelect)
	if err != nil {
		return nil, err
	}

	// 构建DMLDependency
	dep := &DMLDependency{
		Mode:      ModeDMLRow,
		Table:     tableDef,
		Operation: dmlConfig.Operation,
		Count:     count,
		WriteType: dmlConfig.WriteType,
	}

	// 验证依赖条件
	if err := dep.Validate(); err != nil {
		return nil, err
	}

	return dep, nil
}

// collectTransactionDependency 收集事务操作的依赖条件
func (g *DMLGenerator) collectTransactionDependency(transConfig *TransactionConfig) (generator.GenerationDependency, error) {
	// 获取所有涉及的表
	tables := make([]*mysql.Table, 0, len(transConfig.Operations))
	for tableName := range transConfig.Operations {
		// 通过表名查询表定义
		// 这里假设所有表都在当前metadata中
		for _, key := range g.metadata.SchemaKeys() {
			schema, err := g.metadata.SchemaStore().GetSchema(key)
			if err != nil {
				continue
			}
			table, ok := schema.(*mysql.Table)
			if !ok {
				continue
			}
			_, name := table.Schema()
			if name == tableName {
				tables = append(tables, table)
				break
			}
		}
	}

	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables found for transaction operations")
	}

	// 构建TransactionDependency
	dep := &TransactionDependency{
		Tables:     tables,
		Operations: transConfig.Operations,
	}

	// 验证依赖条件
	if err := dep.Validate(); err != nil {
		return nil, err
	}

	return dep, nil
}

// collectDDLDependency 收集DDL操作的依赖条件
func (g *DMLGenerator) collectDDLDependency(ddlConfig *DDLConfig) (generator.GenerationDependency, error) {
	// 获取第一个表作为DDL操作的目标表
	// 实际场景中，用户应该在DDLConfig中指定具体的表名
	var table *mysql.Table
	if len(g.metadata.SchemaKeys()) > 0 {
		schema, err := g.metadata.SchemaStore().GetSchema(g.metadata.SchemaKeys()[0])
		if err != nil {
			return nil, fmt.Errorf("failed to get schema: %w", err)
		}
		var ok bool
		table, ok = schema.(*mysql.Table)
		if !ok {
			return nil, fmt.Errorf("schema is not a mysql table")
		}
	}

	if table == nil {
		return nil, fmt.Errorf("no table available for DDL operation")
	}

	// 构建DDLDependency
	dep := &DDLDependency{
		Table:   table,
		DDLType: ddlConfig.DDLType,
		Columns: ddlConfig.Columns,
	}

	// 验证依赖条件
	if err := dep.Validate(); err != nil {
		return nil, err
	}

	return dep, nil
}

func (g *DMLGenerator) getNextIndex() int {
	g.hitIndex++
	if g.hitIndex > len(g.metadata.SchemaKeys()) {
		g.hitIndex = 0
	}
	return g.hitIndex
}

func (g *DMLGenerator) getTableDef(tableSelect string) (*mysql.Table, error) {
	var index int
	switch tableSelect {
	case generator.RandomTableSelect:
		index = rand.IntN(len(g.metadata.SchemaKeys()))
	case generator.OrderedTableSelect:
		index = g.getNextIndex()
	case generator.SameWithLastTableSelect:
		index = g.hitIndex
	case generator.DiffFromLastTableSelect:
		index = g.getNextIndex()
	}
	key := g.metadata.SchemaKeys()[index]
	schema, err := g.metadata.SchemaStore().GetSchema(key)
	if err != nil {
		return nil, err
	}
	table, ok := schema.(*mysql.Table)
	if !ok {
		err = fmt.Errorf("table %s is not a mysql table", key)
		return nil, err
	}
	return table, nil
}

// MockMessage 第二阶段：生成消息（接收反查结果）
func (g *DMLGenerator) MockMessage(req *generator.MessageGenerationRequest) (message.Message, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if req.Dependency == nil {
		return nil, fmt.Errorf("dependency is required")
	}

	// 根据依赖类型处理不同的消息生成
	switch dep := req.Dependency.(type) {
	case *DMLDependency:
		return g.generateDMLMessage(dep, req)
	case *TransactionDependency:
		return g.generateTransactionMessage(dep, req)
	case *DDLDependency:
		return g.generateDDLMessage(dep, req)
	default:
		return nil, fmt.Errorf("unsupported dependency type: %T", req.Dependency)
	}
}

// generateDMLMessage 生成单表DML消息
func (g *DMLGenerator) generateDMLMessage(dmlDep *DMLDependency, req *generator.MessageGenerationRequest) (message.Message, error) {
	if dmlDep.Table == nil {
		return nil, fmt.Errorf("table is nil")
	}

	// 根据生成策略生成数据
	var rowDataList []mysql.RowData

	rowDataList = g.rowBuilder.BuildRows(dmlDep.Table, dmlDep.Operation, dmlDep.Count)

	if len(rowDataList) == 0 {
		return nil, nil
	}

	// 创建单表行消息
	db, table := dmlDep.Table.Schema()
	rowMsg := &message.MySQLRowMessage{
		SQLRows: message.SQLRows{
			Metadata: message.Metadata{
				Database:  db,
				Table:     table,
				Operation: dmlDep.Operation,
				WriteType: dmlDep.WriteType,
				StartTime: g.getCurrentTime(),
			},
			Contents: rowDataList,
		},
	}
	return rowMsg, nil
}

// generateTransactionMessage 生成多表事务消息
func (g *DMLGenerator) generateTransactionMessage(transDep *TransactionDependency, req *generator.MessageGenerationRequest) (message.Message, error) {
	if len(transDep.Tables) == 0 {
		return nil, fmt.Errorf("no tables in transaction dependency")
	}

	// 为每个表生成对应操作的行数据
	sqlRowsList := make([]message.SQLRows, 0, len(transDep.Tables))

	for _, table := range transDep.Tables {
		_, tableName := table.Schema()

		// 获取该表的操作配置
		op, ok := transDep.Operations[tableName]
		if !ok {
			continue
		}

		// 生成该表的行数据
		var rowDataList []mysql.RowData
		switch op.Operation {
		case message.Insert, message.InsertIgnore, message.Replace, message.InsertOnDuplicateKey:
			rowDataList = g.rowBuilder.BuildRows(table, op.Operation, op.Count)
		case message.Update:
			rowDataList = g.rowBuilder.BuildRows(table, op.Operation, op.Count)
		case message.Delete:
			rowDataList = g.rowBuilder.BuildRows(table, op.Operation, op.Count)
		default:
			rowDataList = g.rowBuilder.BuildRows(table, message.Insert, op.Count)
		}

		if len(rowDataList) == 0 {
			continue
		}

		db, tbl := table.Schema()
		sqlRows := message.SQLRows{
			Metadata: message.Metadata{
				Database:  db,
				Table:     tbl,
				Operation: op.Operation,
				WriteType: op.WriteType,
				StartTime: g.getCurrentTime(),
			},
			Contents: rowDataList,
		}
		sqlRowsList = append(sqlRowsList, sqlRows)
	}

	if len(sqlRowsList) == 0 {
		return nil, nil
	}

	// 创建事务消息
	transMsg := &message.MySQLTransactionMessage{
		GTID:    fmt.Sprintf("transaction_%d", time.Now().UnixNano()),
		SQLRows: sqlRowsList,
	}
	return transMsg, nil
}

// generateDDLMessage 生成DDL操作消息
func (g *DMLGenerator) generateDDLMessage(ddlDep *DDLDependency, req *generator.MessageGenerationRequest) (message.Message, error) {
	if ddlDep.Table == nil {
		return nil, fmt.Errorf("table is nil")
	}

	db, table := ddlDep.Table.Schema()

	// 为简化实现，生成一个DDL消息
	// 实际应用中，可以根据DDLType和Columns生成具体的DDL语句
	ddlMsg := &message.MySQLDDLMessage{
		Metadata: message.Metadata{
			Database:  db,
			Table:     table,
			Operation: ddlDep.DDLType,
			WriteType: ddlDep.DDLType,
			StartTime: g.getCurrentTime(),
		},
		// DDLStatement 是一个复杂的结构，这里暂时使用空值
		// 在实际应用中，应该根据DDLType和Columns构造具体的DDL语句
		DDLStatement: mysql.DDLStatement{},
	}

	return ddlMsg, nil
}

// buildSequenceRows 序列化行生成逻辑
func (g *DMLGenerator) buildSequenceRows(dep *DMLDependency, seqConfig *iquery.SequenceConfig) []mysql.RowData {
	rowDataList := make([]mysql.RowData, 0, dep.Count)

	if seqConfig == nil {
		return rowDataList
	}

	db, tableName := dep.Table.Schema()
	stateKey := fmt.Sprintf("%s.%s:%s", db, tableName, seqConfig.Field)

	// 保护对CurrentValue的并发访问
	g.seqMutex.Lock()
	currentValue := seqConfig.CurrentValue
	g.seqMutex.Unlock()

	for i := 0; i < dep.Count; i++ {
		// 检查是否超出分配范围
		if currentValue > seqConfig.EndValue {
			break
		}

		// 使用RowBuilder生成一行基础数据
		var rowData *mysql.RowData
		switch dep.Operation {
		case message.Insert, message.InsertIgnore, message.Replace, message.InsertOnDuplicateKey:
			rowData = g.rowBuilder.BuildInsertRow(dep.Table)
		case message.Update:
			rowData = g.rowBuilder.BuildUpdateRow(dep.Table)
		case message.Delete:
			rowData = g.rowBuilder.BuildDeleteRow(dep.Table)
		default:
			rowData = g.rowBuilder.BuildInsertRow(dep.Table)
		}

		// 将序列字段值设为CurrentValue
		if rowData != nil && rowData.Data != nil {
			rowData.Data[seqConfig.Field] = currentValue
		}

		currentValue += seqConfig.Step
		if rowData != nil {
			rowDataList = append(rowDataList, *rowData)
		}
	}

	// 更新seqConfig的CurrentValue
	g.seqMutex.Lock()
	seqConfig.CurrentValue = currentValue
	g.sequenceStates[stateKey] = currentValue
	g.seqMutex.Unlock()

	return rowDataList
}

// getCurrentTime 获取当前时间
func (g *DMLGenerator) getCurrentTime() time.Time {
	return time.Now()
}

func (g *DMLGenerator) Close() {

}

func ParseCount(countStr string) (int, error) {
	if countStr == "" {
		return 1, nil
	}

	// 处理 "random" 格式
	if countStr == "random" {
		return rand.IntN(100) + 1, nil
	}
	// 尝试直接解析为整数
	return strconv.Atoi(countStr)
}

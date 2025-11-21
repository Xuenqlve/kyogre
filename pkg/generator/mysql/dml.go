package mysql

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin"
)

type Config struct {
	Metadata string `mapstructure:"metadata" json:"metadata"`
}

type DMLGenerator struct {
	pipeline   string
	metadata   metadata.Metadata
	config     Config
	hitIndex   int
	rowBuilder *RowBuilder
}

func (g *DMLGenerator) Configure(pipeline string, data map[string]any) (err error) {
	g.pipeline = pipeline
	if err = mapstructure.Decode(data, g.config); err != nil {
		return err
	}
	g.hitIndex = 0
	g.rowBuilder = NewRowBuilder()
	return nil
}

func (g *DMLGenerator) RegisterMetadata(metadata metadata.Metadata) {
	g.metadata = metadata
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
	case plugin.RandomTableSelect:
		index = rand.IntN(len(g.metadata.SchemaKeys()))
	case plugin.OrderedTableSelect:
		index = g.getNextIndex()
	case plugin.SameWithLastTableSelect:
		index = g.hitIndex
	case plugin.DiffFromLastTableSelect:
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

func (g *DMLGenerator) MockMessage(param plugin.MockParam) message.Message {
	tableDef, err := g.getTableDef(param.TableSelect)
	if err != nil {
		return nil
	}

	// 解析行数
	count, err := ParseCount(param.Count)
	if err != nil {
		count = 1
	}

	// 使用 RowBuilder 生成行数据
	rowDataList := g.rowBuilder.BuildRows(tableDef, param.Operation, count)
	if len(rowDataList) == 0 {
		return nil
	}

	// 创建消息
	db, table := tableDef.Schema()
	rowMsg := &message.MySQLRowMessage{
		SQLRows: message.SQLRows{
			Metadata: message.Metadata{
				Database:  db,
				Table:     table,
				Operation: param.Operation,
				WriteType: param.Operation,
				StartTime: g.getCurrentTime(),
			},
			Contents: rowDataList,
		},
	}
	return rowMsg
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

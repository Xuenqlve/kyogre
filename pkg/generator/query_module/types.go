package query_module

import (
	"context"
	"sync"
	"time"

	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/metadata"
)

// QueryModuleConfig 反查模块的配置
type QueryModuleConfig struct {
	Enabled     bool   `json:"enabled"`      // 是否启用
	QuerySource string `json:"query_source"` // "database" 或 "memory" 或 "composite"
	CacheExpire int    `json:"cache_expire"` // 缓存过期时间（秒）
	BatchSize   int    `json:"batch_size"`   // 批查询大小
}

// QueryResult 反查结果，统一返回格式
type QueryResult struct {
	TableKey        string         // 表的唯一标识符 "db.table"
	Field           string         // 反查的字段
	MaxValue        int64          // 字段的最大值
	CurrentRowCount int64          // 表的当前行数
	Metadata        map[string]any // 其他元数据
	QueryTime       time.Time      // 查询时间
}

// IQueryModule 反查模块接口
type IQueryModule interface {
	// 根据表和字段查询最大值
	QueryMaxValue(ctx context.Context, table *mysql.Table, field string) (int64, error)

	// 查询表的行数
	QueryRowCount(ctx context.Context, table *mysql.Table) (int64, error)

	// 批量查询多张表的信息
	BatchQuery(ctx context.Context, metadata metadata.Metadata) ([]*QueryResult, error)

	// 获取完整的查询结果（包含所有元信息）
	GetQueryResult(ctx context.Context, table *mysql.Table, field string) (*QueryResult, error)

	Close() error
}

// DatabaseQueryModule 从数据库查询的实现
type DatabaseQueryModule struct {
	dataSource  any
	cacheMap    map[string]*QueryResult
	cacheMutex  sync.RWMutex
	cacheExpire time.Duration
}

// NewDatabaseQueryModule 创建数据库查询模块
func NewDatabaseQueryModule(dataSource any, cfg *QueryModuleConfig) (*DatabaseQueryModule, error) {
	expireDuration := time.Duration(cfg.CacheExpire) * time.Second
	if cfg.CacheExpire <= 0 {
		expireDuration = 5 * time.Minute // 默认5分钟过期
	}

	return &DatabaseQueryModule{
		dataSource:  dataSource,
		cacheMap:    make(map[string]*QueryResult),
		cacheExpire: expireDuration,
	}, nil
}

// QueryMaxValue 查询字段最大值
func (d *DatabaseQueryModule) QueryMaxValue(ctx context.Context, table *mysql.Table, field string) (int64, error) {
	cacheKey := d.makeCacheKey(table, field, "max")

	// 先查缓存
	d.cacheMutex.RLock()
	if result, ok := d.cacheMap[cacheKey]; ok {
		if time.Since(result.QueryTime) < d.cacheExpire {
			d.cacheMutex.RUnlock()
			return result.MaxValue, nil
		}
	}
	d.cacheMutex.RUnlock()

	// 从数据库查询
	// 这里的实现需要根据实际的dataSource类型来处理
	// 暂时返回0，实际实现时需要执行SQL查询
	maxValue := int64(0)

	// 缓存结果
	d.cacheMutex.Lock()
	d.cacheMap[cacheKey] = &QueryResult{
		TableKey:        d.makeTableKey(table),
		Field:           field,
		MaxValue:        maxValue,
		CurrentRowCount: 0,
		Metadata:        make(map[string]any),
		QueryTime:       time.Now(),
	}
	d.cacheMutex.Unlock()

	return maxValue, nil
}

// QueryRowCount 查询表行数
func (d *DatabaseQueryModule) QueryRowCount(ctx context.Context, table *mysql.Table) (int64, error) {
	cacheKey := d.makeCacheKey(table, "", "count")

	// 先查缓存
	d.cacheMutex.RLock()
	if result, ok := d.cacheMap[cacheKey]; ok {
		if time.Since(result.QueryTime) < d.cacheExpire {
			d.cacheMutex.RUnlock()
			return result.CurrentRowCount, nil
		}
	}
	d.cacheMutex.RUnlock()

	// 从数据库查询
	rowCount := int64(0)

	// 缓存结果
	d.cacheMutex.Lock()
	d.cacheMap[cacheKey] = &QueryResult{
		TableKey:        d.makeTableKey(table),
		Field:           "",
		MaxValue:        0,
		CurrentRowCount: rowCount,
		Metadata:        make(map[string]any),
		QueryTime:       time.Now(),
	}
	d.cacheMutex.Unlock()

	return rowCount, nil
}

// BatchQuery 批量查询
func (d *DatabaseQueryModule) BatchQuery(ctx context.Context, tables []*mysql.Table, fields []string) ([]*QueryResult, error) {
	results := make([]*QueryResult, 0, len(tables)*len(fields))

	for _, table := range tables {
		for _, field := range fields {
			result, err := d.GetQueryResult(ctx, table, field)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// GetQueryResult 获取完整的查询结果
func (d *DatabaseQueryModule) GetQueryResult(ctx context.Context, table *mysql.Table, field string) (*QueryResult, error) {
	cacheKey := d.makeCacheKey(table, field, "full")

	d.cacheMutex.RLock()
	if result, ok := d.cacheMap[cacheKey]; ok {
		if time.Since(result.QueryTime) < d.cacheExpire {
			d.cacheMutex.RUnlock()
			return result, nil
		}
	}
	d.cacheMutex.RUnlock()

	maxValue, err := d.QueryMaxValue(ctx, table, field)
	if err != nil {
		return nil, err
	}

	rowCount, err := d.QueryRowCount(ctx, table)
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		TableKey:        d.makeTableKey(table),
		Field:           field,
		MaxValue:        maxValue,
		CurrentRowCount: rowCount,
		Metadata:        make(map[string]any),
		QueryTime:       time.Now(),
	}

	d.cacheMutex.Lock()
	d.cacheMap[cacheKey] = result
	d.cacheMutex.Unlock()

	return result, nil
}

// Close 关闭模块
func (d *DatabaseQueryModule) Close() error {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()
	d.cacheMap = nil
	return nil
}

// MemoryQueryModule 从内存缓存查询的实现
type MemoryQueryModule struct {
	data      map[string]*QueryResult
	dataMutex sync.RWMutex
}

// NewMemoryQueryModule 创建内存查询模块
func NewMemoryQueryModule(cfg *QueryModuleConfig) (*MemoryQueryModule, error) {
	return &MemoryQueryModule{
		data: make(map[string]*QueryResult),
	}, nil
}

// QueryMaxValue 从内存查询字段最大值
func (m *MemoryQueryModule) QueryMaxValue(ctx context.Context, table *mysql.Table, field string) (int64, error) {
	m.dataMutex.RLock()
	defer m.dataMutex.RUnlock()

	key := makeQueryKey(table, field)
	if result, ok := m.data[key]; ok {
		return result.MaxValue, nil
	}
	return 0, nil
}

// QueryRowCount 从内存查询表行数
func (m *MemoryQueryModule) QueryRowCount(ctx context.Context, table *mysql.Table) (int64, error) {
	m.dataMutex.RLock()
	defer m.dataMutex.RUnlock()

	key := makeQueryKey(table, "")
	if result, ok := m.data[key]; ok {
		return result.CurrentRowCount, nil
	}
	return 0, nil
}

// BatchQuery 批量查询
func (m *MemoryQueryModule) BatchQuery(ctx context.Context, tables []*mysql.Table, fields []string) ([]*QueryResult, error) {
	results := make([]*QueryResult, 0, len(tables)*len(fields))

	m.dataMutex.RLock()
	for _, table := range tables {
		for _, field := range fields {
			key := makeQueryKey(table, field)
			if result, ok := m.data[key]; ok {
				results = append(results, result)
			}
		}
	}
	m.dataMutex.RUnlock()

	return results, nil
}

// GetQueryResult 从内存获取完整的查询结果
func (m *MemoryQueryModule) GetQueryResult(ctx context.Context, table *mysql.Table, field string) (*QueryResult, error) {
	m.dataMutex.RLock()
	defer m.dataMutex.RUnlock()

	key := makeQueryKey(table, field)
	if result, ok := m.data[key]; ok {
		return result, nil
	}
	return nil, nil
}

// SetQueryResult 设置查询结果（用于初始化）
func (m *MemoryQueryModule) SetQueryResult(result *QueryResult) {
	m.dataMutex.Lock()
	defer m.dataMutex.Unlock()
	key := result.TableKey + ":" + result.Field
	m.data[key] = result
}

// Close 关闭模块
func (m *MemoryQueryModule) Close() error {
	m.dataMutex.Lock()
	defer m.dataMutex.Unlock()
	m.data = nil
	return nil
}

// CompositeQueryModule 混合策略（先查内存，再查数据库）
type CompositeQueryModule struct {
	memory   *MemoryQueryModule
	database *DatabaseQueryModule
}

// NewCompositeQueryModule 创建混合查询模块
func NewCompositeQueryModule(dataSource any, cfg *QueryModuleConfig) (*CompositeQueryModule, error) {
	memory, err := NewMemoryQueryModule(cfg)
	if err != nil {
		return nil, err
	}

	database, err := NewDatabaseQueryModule(dataSource, cfg)
	if err != nil {
		return nil, err
	}

	return &CompositeQueryModule{
		memory:   memory,
		database: database,
	}, nil
}

// QueryMaxValue 查询字段最大值（先查内存再查DB）
func (c *CompositeQueryModule) QueryMaxValue(ctx context.Context, table *mysql.Table, field string) (int64, error) {
	// 先查内存
	val, err := c.memory.QueryMaxValue(ctx, table, field)
	if err == nil && val > 0 {
		return val, nil
	}

	// 再查数据库
	return c.database.QueryMaxValue(ctx, table, field)
}

// QueryRowCount 查询表行数
func (c *CompositeQueryModule) QueryRowCount(ctx context.Context, table *mysql.Table) (int64, error) {
	// 先查内存
	count, err := c.memory.QueryRowCount(ctx, table)
	if err == nil && count > 0 {
		return count, nil
	}

	// 再查数据库
	return c.database.QueryRowCount(ctx, table)
}

// BatchQuery 批量查询
func (c *CompositeQueryModule) BatchQuery(ctx context.Context, tables []*mysql.Table, fields []string) ([]*QueryResult, error) {
	// 先从内存查询
	results, err := c.memory.BatchQuery(ctx, tables, fields)
	if err != nil {
		return nil, err
	}

	// 对于内存中没有的，从数据库查询
	dbResults, err := c.database.BatchQuery(ctx, tables, fields)
	if err != nil {
		return nil, err
	}

	// 合并结果
	return append(results, dbResults...), nil
}

// GetQueryResult 获取完整的查询结果
func (c *CompositeQueryModule) GetQueryResult(ctx context.Context, table *mysql.Table, field string) (*QueryResult, error) {
	// 先查内存
	result, err := c.memory.GetQueryResult(ctx, table, field)
	if err == nil && result != nil {
		return result, nil
	}

	// 再查数据库
	return c.database.GetQueryResult(ctx, table, field)
}

// Close 关闭模块
func (c *CompositeQueryModule) Close() error {
	if err := c.memory.Close(); err != nil {
		return err
	}
	return c.database.Close()
}

// 辅助函数

func (d *DatabaseQueryModule) makeCacheKey(table *mysql.Table, field string, suffix string) string {
	tableKey := d.makeTableKey(table)
	return tableKey + ":" + field + ":" + suffix
}

func (d *DatabaseQueryModule) makeTableKey(table *mysql.Table) string {
	db, tbl := table.Schema()
	return db + "." + tbl
}

func makeQueryKey(table *mysql.Table, field string) string {
	db, tbl := table.Schema()
	return db + "." + tbl + ":" + field
}

package mysql

import (
	"context"
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/generator/query_module"
)

type Scenario struct {
	pipeline   string
	cfg        *Config
	ctx        context.Context
	metadata   metadata.Metadata
	generator  []plugin.Generator
	workers    []*Worker

	// 【新增】反查模块（可选）
	queryModule query_module.IQueryModule

	// 【新增】预分段的映射，支持多表多字段
	// key: "db.table:field", value: *SequenceSegment
	segmentMap map[string]*SequenceSegment
	segMutex   sync.RWMutex
}

// SequenceSegment 分段信息
type SequenceSegment struct {
	Ranges []*SegmentRange
}

type SegmentRange struct {
	WorkerID   int
	StartValue int64
	EndValue   int64
}

func NewScenario() *Scenario {
	return &Scenario{
		generator:  make([]plugin.Generator, 0),
		workers:    make([]*Worker, 0),
		segmentMap: make(map[string]*SequenceSegment),
	}
}

func (s *Scenario) Configure(pipeline string, data map[string]any) (err error) {
	s.pipeline = pipeline
	s.cfg = &Config{}
	if err = mapstructure.Decode(data, s.cfg); err != nil {
		return
	}
	if err = s.cfg.Validate(); err != nil {
		return
	}
	return
}

func (s *Scenario) Preparation(ctx context.Context) error {
	s.ctx = ctx

	// 1. 初始化metadata
	// TODO: 从配置中初始化metadata
	// metadata, err := initMetadata(...)
	// if err != nil {
	//     return err
	// }
	// s.metadata = metadata

	// 2. 初始化Generator（每个worker一个）
	// TODO: 根据配置创建generator实例
	// s.generator = make([]plugin.Generator, s.cfg.WorkerCount)
	// for i := 0; i < s.cfg.WorkerCount; i++ {
	//     gen := NewDMLGenerator()
	//     gen.RegisterMetadata(metadata)
	//     gen.Configure(s.pipeline, ...)
	//     s.generator[i] = gen
	// }

	// 3. 【关键】条件判断：是否启用反查模块
	if s.cfg.QueryModule != nil && s.cfg.QueryModule.Enabled {
		// 初始化反查模块
		qm, err := s.initQueryModule()
		if err != nil {
			return err
		}
		s.queryModule = qm

		// 4. 执行预分段（仅当启用反查且配置了序列化时）
		if s.cfg.GenerationStrategy != nil &&
			s.cfg.GenerationStrategy.SequenceConfig != nil &&
			s.cfg.GenerationStrategy.SequenceConfig.Enabled {
			if err := s.allocateSequenceSegments(); err != nil {
				return err
			}
		}
	}

	return nil
}

// 初始化反查模块
func (s *Scenario) initQueryModule() (query_module.IQueryModule, error) {
	cfg := s.cfg.QueryModule

	switch cfg.QuerySource {
	case "database":
		// TODO: 需要从dataSource获取真实的数据库连接
		return query_module.NewDatabaseQueryModule(nil, cfg)
	case "memory":
		return query_module.NewMemoryQueryModule(cfg)
	case "composite":
		// TODO: 需要从dataSource获取真实的数据库连接
		return query_module.NewCompositeQueryModule(nil, cfg)
	default:
		return nil, fmt.Errorf("unsupported query source: %s", cfg.QuerySource)
	}
}

// 预分段逻辑
func (s *Scenario) allocateSequenceSegments() error {
	if s.queryModule == nil {
		return fmt.Errorf("query module is not initialized")
	}

	if s.metadata == nil {
		return fmt.Errorf("metadata is not initialized")
	}

	seqConfig := s.cfg.GenerationStrategy.SequenceConfig

	schemaKeys := s.metadata.SchemaKeys()
	if len(schemaKeys) == 0 {
		return fmt.Errorf("no tables in metadata")
	}

	// 为每张表的指定字段进行分段
	for _, schemaKey := range schemaKeys {
		schema, err := s.metadata.SchemaStore().GetSchema(schemaKey)
		if err != nil {
			return err
		}

		table, ok := schema.(*mysql.Table)
		if !ok {
			return fmt.Errorf("schema %v is not a mysql table", schemaKey)
		}

		// 查询当前最大值
		maxValue, err := s.queryModule.QueryMaxValue(s.ctx, table, seqConfig.Field)
		if err != nil {
			return err
		}

		// 进行分段
		db, tableName := table.Schema()
		key := fmt.Sprintf("%s.%s:%s", db, tableName, seqConfig.Field)
		s.segmentMap[key] = s.divideSegments(maxValue, s.cfg.WorkerCount)
	}

	return nil
}

// 分段分配算法
func (s *Scenario) divideSegments(maxValue int64, workerCount int) *SequenceSegment {
	seg := &SequenceSegment{
		Ranges: make([]*SegmentRange, workerCount),
	}

	// 避免分割大小为0
	if maxValue <= 0 {
		maxValue = 1
	}

	rangeSize := (maxValue / int64(workerCount)) + 1

	for i := 0; i < workerCount; i++ {
		seg.Ranges[i] = &SegmentRange{
			WorkerID:   i,
			StartValue: int64(i)*rangeSize + maxValue + 1,
			EndValue:   int64(i+1)*rangeSize + maxValue,
		}
	}

	return seg
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	if len(s.generator) == 0 {
		return fmt.Errorf("no generators initialized")
	}

	// 从配置中获取依赖配置
	dependencyConfig, err := s.cfg.DependencyConfig.GetDependencyConfig()
	if err != nil {
		return fmt.Errorf("failed to get dependency config: %w", err)
	}

	for i := 0; i < s.cfg.WorkerCount; i++ {
		worker := NewWorker(
			s.ctx,
			msgChan,
			s.generator[i],
			s.queryModule,           // 传入反查模块（可能为nil）
			s.segmentMap,            // 传入分段信息
			i,                       // worker ID
			dependencyConfig,        // 传入依赖配置
			s.cfg.GenerationStrategy,// 传入生成策略
		)
		worker.Start()
		s.workers = append(s.workers, worker)
	}
	return nil
}

func (s *Scenario) Close() error {
	for _, w := range s.workers {
		w.Close()
	}

	if s.queryModule != nil {
		if err := s.queryModule.Close(); err != nil {
			return err
		}
	}

	return nil
}

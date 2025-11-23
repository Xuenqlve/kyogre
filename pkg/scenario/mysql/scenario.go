package mysql

import (
	"context"
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin"
)

type Scenario struct {
	pipeline  string
	cfg       *Config
	ctx       context.Context
	metadata  metadata.Metadata
	generator []plugin.Generator
	workers   []*Worker

	// 【新增】反查模块（可选）
	queryModule plugin.IQuery

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
	// 3. 【关键】条件判断：是否启用反查模块
	if s.cfg.EnableIQuery {
		// 初始化反查模块
		qm, err := plugin.GetIQueryModule(plugin.IQueryType(s.cfg.IQueryModule.Type))
		if err != nil {
			return err
		}
		if err = qm.Configure(s.pipeline, s.cfg.IQueryModule.Config); err != nil {
			return err
		}
		s.queryModule = qm

		// 4. 执行预分段（仅当启用反查且配置了序列化时）
		if s.cfg.GenerationStrategy != nil &&
			s.cfg.GenerationStrategy.SequenceConfig != nil &&
			s.cfg.GenerationStrategy.SequenceConfig.Enabled {
			if err = s.allocateSequenceSegments(); err != nil {
				return err
			}
		}
	}

	return nil
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
		// 查询当前最大值
		maxValue, err := s.queryModule.QueryMaxValue(s.ctx, schemaKey, seqConfig.Field)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("%s:%s", schemaKey.UniqueID(), seqConfig.Field)
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
			s.queryModule,            // 传入反查模块（可能为nil）
			s.segmentMap,             // 传入分段信息
			i,                        // worker ID
			dependencyConfig,         // 传入依赖配置
			s.cfg.GenerationStrategy, // 传入生成策略
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

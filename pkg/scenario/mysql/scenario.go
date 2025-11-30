package mysql

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/iquery"
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
	queryModule iquery.IQuery

	// 【新增】分段管理器，负责管理序列化字段的分段分配和缓存
	segmentManager *iquery.SegmentManager
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
		generator: make([]plugin.Generator, 0),
		workers:   make([]*Worker, 0),
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
		qm, err := iquery.GetIQueryModule(iquery.IQueryType(s.cfg.IQueryModule.Type))
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
			// 使用 SegmentManager 进行分段分配
			s.segmentManager = iquery.NewSegmentManager(s.queryModule, s.metadata, s.cfg.WorkerCount)
			if err = s.segmentManager.AllocateSegments(ctx, s.cfg.GenerationStrategy.SequenceConfig); err != nil {
				return err
			}
		}
	}

	return nil
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
			s.segmentManager,         // 传入分段管理器（可能为nil）
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

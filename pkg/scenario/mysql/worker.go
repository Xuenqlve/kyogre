package mysql

import (
	"context"
	"sync"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/generator/query_module"
)

type Worker struct {
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	msgQueue         message.InPoint
	generator        plugin.Generator
	queryModule      query_module.IQueryModule  // 可能为nil
	segmentMap       map[string]*SequenceSegment
	workerID         int
	once             sync.Once

	// 从Scenario配置中获取的依赖配置和生成策略
	dependencyConfig   plugin.DependencyConfig
	generationStrategy *plugin.GenerationStrategy
}

func NewWorker(
	ctx context.Context,
	msgChan message.InPoint,
	generator plugin.Generator,
	queryModule query_module.IQueryModule,
	segmentMap map[string]*SequenceSegment,
	workerID int,
	dependencyConfig plugin.DependencyConfig,
	generationStrategy *plugin.GenerationStrategy,
) *Worker {
	lctx, cancel := context.WithCancel(ctx)
	return &Worker{
		ctx:                lctx,
		cancel:             cancel,
		wg:                 sync.WaitGroup{},
		msgQueue:           msgChan,
		generator:          generator,
		queryModule:        queryModule,
		segmentMap:         segmentMap,
		workerID:           workerID,
		once:               sync.Once{},
		dependencyConfig:   dependencyConfig,
		generationStrategy: generationStrategy,
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer func() {
			w.wg.Done()
			if r := recover(); r != nil {
				log.Errorf("MySQL Worker.run panicked: %v", r)
			}
		}()

		if err := w.run(); err != nil {
			log.Errorf("Worker.run err:%s", err.Error())
			return
		}
	}()
}

// 三阶段调用的核心逻辑
func (w *Worker) run() error {
	for {
		select {
		case <-w.ctx.Done():
			log.Infof("[worker %d] ctx Done", w.workerID)
			return nil
		default:
		}

		// ========== 阶段1：收集依赖条件 ==========
		depReq := plugin.NewDependencyRequest(
			w.buildDependencyConfig(),
			w.getGenerationStrategy(),
		)

		dep, err := w.generator.CollectDependencies(depReq)
		if err != nil {
			log.Errorf("[worker %d] failed to collect dependencies: %v", w.workerID, err)
			continue
		}

		// ========== 阶段2：执行反查（如果启用） ==========
		var queryResults map[string]*query_module.QueryResult
		if w.queryModule != nil {
			tables := dep.GetTables()
			fields := dep.GetFields()

			if len(tables) > 0 && len(fields) > 0 {
				results, err := w.queryModule.BatchQuery(w.ctx, tables, fields)
				if err != nil {
					log.Errorf("[worker %d] failed to query: %v", w.workerID, err)
					continue
				}

				queryResults = make(map[string]*query_module.QueryResult)
				for _, result := range results {
					queryResults[result.TableKey] = result
				}
			}
		}

		// ========== 阶段3：生成消息 ==========
		msgReq := plugin.NewMessageGenerationRequest(
			dep,
			queryResults,
			w.getGenerationStrategy(),
		)

		mockMessage, err := w.generator.MockMessage(msgReq)
		if err != nil {
			log.Errorf("[worker %d] failed to generate message: %v", w.workerID, err)
			continue
		}

		if mockMessage != nil {
			select {
			case w.msgQueue <- mockMessage:
			case <-w.ctx.Done():
				return nil
			}
		}
	}
}

// buildDependencyConfig 构建依赖条件的配置
// 返回从Scenario传入的配置对象，确保所有Worker使用统一的配置
func (w *Worker) buildDependencyConfig() plugin.DependencyConfig {
	// 直接返回从Scenario传入的配置
	// 这样所有Worker都使用相同的配置策略
	return w.dependencyConfig
}

// getGenerationStrategy 获取该worker的生成策略（包括序列化配置）
func (w *Worker) getGenerationStrategy() *plugin.GenerationStrategy {
	// 复制从Scenario传入的基础生成策略
	strategy := &plugin.GenerationStrategy{
		SequenceConfig: w.generationStrategy.SequenceConfig,
		RandomConfig:   w.generationStrategy.RandomConfig,
		TemplateConfig: w.generationStrategy.TemplateConfig,
		CustomConfig:   w.generationStrategy.CustomConfig,
	}

	// 如果启用了序列化，为该worker分配分段范围
	if strategy.SequenceConfig != nil && strategy.SequenceConfig.Enabled && len(w.segmentMap) > 0 {
		// 假设只处理第一个分段（实际可能需要支持多个）
		for key, segment := range w.segmentMap {
			if w.workerID < len(segment.Ranges) {
				r := segment.Ranges[w.workerID]

				// 从key解析出字段名 "db.table:field" -> "field"
				var field string
				if idx := len(key); idx > 0 {
					for i := idx - 1; i >= 0; i-- {
						if key[i] == ':' {
							field = key[i+1:]
							break
						}
					}
				}

				// 为该worker分配分段范围
				strategy.SequenceConfig.StartValue = r.StartValue
				strategy.SequenceConfig.EndValue = r.EndValue
				strategy.SequenceConfig.CurrentValue = r.StartValue
				strategy.SequenceConfig.Field = field
				// 只处理第一个分段
				break
			}
		}
	}

	return strategy
}

func (w *Worker) Done() {
	w.once.Do(func() {
		w.cancel()
	})
}

func (w *Worker) Close() {
	w.wg.Wait()
}

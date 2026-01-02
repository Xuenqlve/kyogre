package mysql

import (
	"context"
	"sync"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

type Worker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	msgQueue  message.InPoint
	generator generator.Generator
	lookup    iquery.Lookup
	sequencer *iquery.Sequencer
	workerID  int
	once      sync.Once

	// 从Scenario配置中获取的依赖配置和生成策略
	dependencyConfig   generator.DependencyConfig
	generationStrategy *generator.GenerationStrategy
}

func NewWorker(
	ctx context.Context,
	msgChan message.InPoint,
	generator generator.Generator,
	lookup iquery.Lookup,
	sequencer *iquery.Sequencer,
	workerID int,
	dependencyConfig generator.DependencyConfig,
	generationStrategy *generator.GenerationStrategy,
) *Worker {
	lctx, cancel := context.WithCancel(ctx)
	return &Worker{
		ctx:                lctx,
		cancel:             cancel,
		wg:                 sync.WaitGroup{},
		msgQueue:           msgChan,
		generator:          generator,
		lookup:             lookup,
		sequencer:          sequencer,
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

		//if err := w.run(); err != nil {
		//	log.Errorf("Worker.run err:%s", err.Error())
		//	return
		//}
	}()
}

//三阶段调用的核心逻辑
//func (w *Worker) run() error {
//	for {
//		select {
//		case <-w.ctx.Done():
//			log.Infof("[worker %d] ctx Done", w.workerID)
//			return nil
//		default:
//		}
//
//		baseStrategy := w.cloneGenerationStrategy()
//
//		// ========== 阶段1：收集依赖条件 ==========
//		depReq := generator.NewDependencyRequest(
//			w.buildDependencyConfig(),
//			baseStrategy,
//		)
//
//		dep, err := w.generator.CollectDependencies(depReq)
//		if err != nil {
//			log.Errorf("[worker %d] failed to collect dependencies: %v", w.workerID, err)
//			continue
//		}
//
//		// ========== 阶段2：执行反查（如果启用） ==========
//		queryResults, err := w.executeLookup(dep)
//		if err != nil {
//			log.Errorf("[worker %d] failed to execute lookup: %v", w.workerID, err)
//			continue
//		}
//
//		// 根据分段配置调整策略
//		msgStrategy, err := w.applySequenceStrategy(dep, baseStrategy)
//		if err != nil {
//			log.Errorf("[worker %d] failed to apply sequence strategy: %v", w.workerID, err)
//			continue
//		}
//
//		// ========== 阶段3：生成消息 ==========
//		msgReq := generator.NewMessageGenerationRequest(
//			dep,
//			queryResults,
//			msgStrategy,
//		)
//
//		mockMessage, err := w.generator.MockMessage(msgReq)
//		if err != nil {
//			log.Errorf("[worker %d] failed to generate message: %v", w.workerID, err)
//			continue
//		}
//
//		if mockMessage != nil {
//			select {
//			case w.msgQueue <- mockMessage:
//			case <-w.ctx.Done():
//				return nil
//			}
//		}
//	}
//}

// buildDependencyConfig 构建依赖条件的配置
// 返回从Scenario传入的配置对象，确保所有Worker使用统一的配置
func (w *Worker) buildDependencyConfig() generator.DependencyConfig {
	// 直接返回从Scenario传入的配置
	// 这样所有Worker都使用相同的配置策略
	return w.dependencyConfig
}

func (w *Worker) cloneGenerationStrategy() *generator.GenerationStrategy {
	if w.generationStrategy == nil {
		return &generator.GenerationStrategy{}
	}

	strategy := &generator.GenerationStrategy{
		TemplateConfig: w.generationStrategy.TemplateConfig,
		CustomConfig:   w.generationStrategy.CustomConfig,
	}
	return strategy
}

//func (w *Worker) executeLookup(dep generator.GenerationDependency) (map[string]*iquery.LookupResult, error) {
//	if w.lookup == nil || dep == nil {
//		return nil, nil
//	}
//
//	schemas := dep.GetSchemas()
//	if len(schemas) == 0 {
//		return nil, nil
//	}
//
//	items := make([]iquery.LookupRequestItem, 0, len(schemas))
//	for _, key := range schemas {
//		items = append(items, iquery.LookupRequestItem{
//			Schema: key,
//		})
//	}
//
//	res, err := w.lookup.Lookup(w.ctx, iquery.LookupRequest{Items: items})
//	if err != nil {
//		return nil, err
//	}
//
//	if len(res) == 0 {
//		return nil, nil
//	}
//
//	queryResults := make(map[string]*iquery.LookupResult, len(res))
//	for i := range res {
//		item := res[i]
//		result := item
//		queryResults[item.Schema.UniqueID()] = &result
//	}
//
//	return queryResults, nil
//}

func (w *Worker) applySequenceStrategy(
	dep generator.GenerationDependency,
	base *generator.GenerationStrategy,
) (*generator.GenerationStrategy, error) {
	if w.sequencer == nil || dep == nil || base == nil {
		return base, nil
	}

	schemas := dep.GetSchemas()
	if len(schemas) == 0 {
		return base, nil
	}
	return nil, nil
}

func (w *Worker) Done() {
	w.once.Do(func() {
		w.cancel()
	})
}

func (w *Worker) Close() {
	w.wg.Wait()
}

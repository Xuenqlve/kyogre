package iquery

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
)

var IQueryManager = &Manager{
	engines: make(map[string]Lookup),
}

type Manager struct {
	pipeline string
	mux      sync.RWMutex
	engines  map[string]Lookup
}

func (e *Manager) Configure(pipeline string, data map[string]config.ConfigureMold) error {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.pipeline = pipeline
	if e.engines == nil {
		e.engines = make(map[string]Lookup)
	}
	for key, cfg := range data {
		iq, err := GetIQueryModule(LookupType(cfg.Type))
		if err != nil {
			return err
		}
		if err = iq.Configure(pipeline, cfg.Config); err != nil {
			return err
		}
		e.engines[key] = iq
	}
	return nil
}

func (e *Manager) GetIQueryLookup(key string) (Lookup, error) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	m, exist := e.engines[key]
	if !exist {
		return nil, fmt.Errorf("iquery not found by key: %s", key)
	}
	return m, nil
}

//
//type HandlerConfig struct {
//	WorkerCount int
//}
//
//type Handler struct {
//	iquery IQuery
//	cfg    *HandlerConfig
//	segmentMap map[string]map[string]
//}
//
//func NewHandler(iQueryType string, opts ...HandlerOptions) (*Handler, error) {
//	query, err := IQueryManager.GetIQuery(iQueryType)
//	if err != nil {
//		return nil, err
//	}
//	cfg := &HandlerConfig{}
//	for _, option := range opts {
//		option(cfg)
//	}
//	return &Handler{
//		q:   query,
//		cfg: cfg,
//	}, nil
//}
//
//func (h *Handler) Register(key string, field string) {
//	h.iquery.QueryMaxValue()
//}
//
//func (h *Handler) GetSegments(key, field string) SequenceSegment {
//	key := fmt.Sprintf("%s:%s", schemaID, field)
//	sm.segMutex.RLock()
//	defer sm.segMutex.RUnlock()
//	return sm.segmentMap[key]
//}

//
//// SegmentManager 负责管理分段的分配和缓存
//// 将分段逻辑从 Scenario 下沉到 Manager 层，支持多表、多字段和多 worker 的场景
//type SegmentManager struct {
//	queryModule IQuery
//	metadata    metadata.Metadata
//	workerCount int
//
//	// 分段缓存：key: "db.table:field", value: *SequenceSegment
//	segmentMap map[string]SequenceSegment
//	segMutex   sync.RWMutex
//}
//
//// NewSegmentManager 创建一个新的分段管理器
//func NewSegmentManager(queryModule IQuery, metadata metadata.Metadata, workerCount int) *SegmentManager {
//	return &SegmentManager{
//		queryModule: queryModule,
//		metadata:    metadata,
//		workerCount: workerCount,
//		segmentMap:  make(map[string]SequenceSegment),
//	}
//}
//
//// AllocateSegments 为指定的表和字段分配分段
//// 该方法会根据配置，为每张表的指定字段进行分段分配
//// 分段信息会被缓存在内存中供后续使用
//func (sm *SegmentManager) AllocateSegments(ctx context.Context, sequenceConfig *SequenceConfig) error {
//	if sm.queryModule == nil {
//		return fmt.Errorf("query module is not initialized")
//	}
//
//	if sm.metadata == nil {
//		return fmt.Errorf("metadata is not initialized")
//	}
//
//	if sequenceConfig == nil {
//		return fmt.Errorf("sequence config is nil")
//	}
//
//	schemaKeys := sm.metadata.SchemaKeys()
//	if len(schemaKeys) == 0 {
//		return fmt.Errorf("no tables in metadata")
//	}
//
//	// 为每张表的指定字段进行分段
//	for _, schemaKey := range schemaKeys {
//		// 查询当前最大值
//		maxValue, err := sm.queryModule.QueryMaxValue(ctx, schemaKey, sequenceConfig.Field)
//		if err != nil {
//			return err
//		}
//
//		// 构建缓存 key: "db.table:field"
//		key := fmt.Sprintf("%s:%s", schemaKey.UniqueID(), sequenceConfig.Field)
//
//		// 分割分段并存储到缓存中
//		sm.segMutex.Lock()
//		sm.segmentMap[key] = sm.divideSegments(maxValue)
//		sm.segMutex.Unlock()
//	}
//
//	return nil
//}
//
//// GetSegments 获取指定表和字段的分段信息
//func (sm *SegmentManager) GetSegments(schemaID, field string) SequenceSegment {
//	key := fmt.Sprintf("%s:%s", schemaID, field)
//	sm.segMutex.RLock()
//	defer sm.segMutex.RUnlock()
//	return sm.segmentMap[key]
//}
//
//// GetAllSegments 返回所有分段信息的副本
//func (sm *SegmentManager) GetAllSegments() map[string]SequenceSegment {
//	//sm.segMutex.RLock()
//	//defer sm.segMutex.RUnlock()
//	//
//	//// 返回副本以避免外部修改
//	//result := make(map[string]*mysql.SequenceSegment, len(sm.segmentMap))
//	//for k, v := range sm.segmentMap {
//	//	result[k] = v
//	//}
//	//return result
//}
//
//// GetWorkerSegmentRange 获取指定 worker 在指定分段中的范围
//// 返回 (startValue, endValue, found)
//func (sm *SegmentManager) GetWorkerSegmentRange(schemaID, field string, workerID int) (int64, int64, bool) {
//	//key := fmt.Sprintf("%s:%s", schemaID, field)
//	//sm.segMutex.RLock()
//	//segment, ok := sm.segmentMap[key]
//	//sm.segMutex.RUnlock()
//	//
//	//if !ok || segment == nil {
//	//	return 0, 0, false
//	//}
//	//
//	//if workerID >= len(segment.Ranges) {
//	//	return 0, 0, false
//	//}
//	//
//	//r := segment.Ranges[workerID]
//	//return r.StartValue, r.EndValue, true
//}
//
//// divideSegments 分段分配算法
//// 将 maxValue 分割成 workerCount 段，每个 worker 一段
//func (sm *SegmentManager) divideSegments(maxValue int64) SequenceSegment {
//	//seg := &mysql.SequenceSegment{
//	//	Ranges: make([]*mysql.SegmentRange, sm.workerCount),
//	//}
//	//
//	//// 避免分割大小为 0
//	//if maxValue <= 0 {
//	//	maxValue = 1
//	//}
//	//
//	//rangeSize := (maxValue / int64(sm.workerCount)) + 1
//	//
//	//for i := 0; i < sm.workerCount; i++ {
//	//	seg.Ranges[i] = &mysql.SegmentRange{
//	//		WorkerID:   i,
//	//		StartValue: int64(i)*rangeSize + maxValue + 1,
//	//		EndValue:   int64(i+1)*rangeSize + maxValue,
//	//	}
//	//}
//	//
//	//return seg
//}

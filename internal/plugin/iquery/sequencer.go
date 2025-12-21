package iquery

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/xuenqlve/common/schema_store"
)

// SequenceSpec 定义需要预分配的序列信息
// SequenceSpec 定义需要预分配的序列信息
type SequenceSpec struct {
	Schema schema_store.SchemaKey
	Field  string
	Step   int64
	Width  int64
}

type Segment struct {
	WorkerID   int
	StartValue int64
	EndValue   int64
}

// SequencerOptions 允许通过可选参数配置 sequencer
type SequencerOptions func(*SequencerConfig)

// WithWorkerCount 设置 worker 数量
func WithWorkerCount(workerCount int) SequencerOptions {
	return func(cfg *SequencerConfig) {
		cfg.WorkerCount = workerCount
	}
}

// WithLookUpKey 指定用于反查的 lookup key
func WithLookUpKey(lookup string) SequencerOptions {
	return func(cfg *SequencerConfig) {
		cfg.LookupKey = lookup
	}
}

// SequencerConfig 保存 sequencer 的基础配置
type SequencerConfig struct {
	WorkerCount int    `mapstructure:"worker-count" json:"worker-count"`
	LookupKey   string `mapstructure:"lookup-key" json:"lookup-key"`
}

// ValidateAndSetDefault 校验配置并补充默认值
func (cfg *SequencerConfig) ValidateAndSetDefault() error {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	if cfg.LookupKey == "" {
		return fmt.Errorf("missing lookup-key")
	}
	return nil
}

var ErrSegmentExhausted = errors.New("sequence segment exhausted, need reallocate")

// Sequencer 负责向反查模块申请序列区间并按 worker 切段
type Sequencer struct {
	pipeline string
	lookup   Lookup
	cfg      *SequencerConfig

	stateMu sync.RWMutex
	states  map[string]*sequenceState
}

type sequenceState struct {
	spec     SequenceSpec
	segments []*workerSegment
	mu       sync.Mutex
}

// reserve 从 worker 可用段落中取出连续序列
func (st *sequenceState) reserve(workerID int, step, length int64) (int64, int64, error) {
	if workerID < 0 || workerID >= len(st.segments) {
		return 0, 0, fmt.Errorf("invalid worker id %d", workerID)
	}
	if step <= 0 {
		step = 1
	}
	if length <= 0 {
		length = 1
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	seg := st.segments[workerID]
	if seg.next > seg.EndValue {
		return 0, 0, ErrSegmentExhausted
	}

	maxLen := ((seg.EndValue - seg.next) / step) + 1
	if maxLen <= 0 {
		return 0, 0, ErrSegmentExhausted
	}
	if length > maxLen {
		length = maxLen
	}

	start := seg.next
	end := start + (length-1)*step
	seg.next = end + step
	return start, end, nil
}

type workerSegment struct {
	Segment
	next int64
}

// NewSequencer 创建 sequencer 并初始化内部状态
func NewSequencer() (*Sequencer, error) {
	return &Sequencer{
		cfg:    &SequencerConfig{},
		states: make(map[string]*sequenceState),
	}, nil
}

// Configure 结合可选项初始化 lookup 和配置
func (s *Sequencer) Configure(pipeline string, opts ...SequencerOptions) (err error) {
	s.pipeline = pipeline
	for _, option := range opts {
		option(s.cfg)
	}
	if err = s.cfg.ValidateAndSetDefault(); err != nil {
		return err
	}
	s.lookup, err = IQueryManager.GetIQueryLookup(s.cfg.LookupKey)
	if err != nil {
		return err
	}
	return nil
}

// Preallocate 调用反查模块，按照 worker 数拆分区间
func (s *Sequencer) Preallocate(ctx context.Context, specs []SequenceSpec) error {
	specMap := make(map[string]SequenceSpec)
	req := LookupRequest{Items: make([]LookupRequestItem, 0, len(specs))}

	for _, spec := range specs {
		if spec.Schema == nil || spec.Field == "" {
			continue
		}
		if spec.Step <= 0 {
			spec.Step = 1
		}
		key := sequenceKey(spec.Schema, spec.Field)
		// 后写的配置覆盖之前的，保证同 key 使用最新参数
		specMap[key] = spec
	}

	for _, spec := range specMap {
		req.Items = append(req.Items, LookupRequestItem{
			Schema: spec.Schema,
			Field:  spec.Field,
		})
	}

	if len(req.Items) == 0 {
		return fmt.Errorf("no valid sequence specs to allocate")
	}

	results, err := s.lookup.Lookup(ctx, req)
	if err != nil {
		return err
	}

	for _, res := range results {
		key := sequenceKey(res.Schema, res.Field)
		spec, ok := specMap[key]
		if !ok {
			continue
		}
		segs := s.buildSegments(res, spec)
		if len(segs) == 0 {
			continue
		}
		s.setState(key, spec, segs)
	}
	return nil
}

// buildSegments 根据反查结果切割各 worker 的连续区间
func (s *Sequencer) buildSegments(res LookupResult, spec SequenceSpec) []*workerSegment {
	if s.cfg.WorkerCount <= 0 {
		return nil
	}

	rangeSize := spec.Width
	if rangeSize <= 0 {
		rangeSize = (res.Max / int64(s.cfg.WorkerCount)) + 1
		if rangeSize <= 0 {
			rangeSize = 1
		}
	}
	if rangeSize < spec.Step {
		rangeSize = spec.Step
	}

	startBase := res.Max + 1
	segs := make([]*workerSegment, s.cfg.WorkerCount)
	for i := 0; i < s.cfg.WorkerCount; i++ {
		start := startBase + int64(i)*rangeSize
		end := start + rangeSize - 1
		segs[i] = &workerSegment{
			Segment: Segment{
				WorkerID:   i,
				StartValue: start,
				EndValue:   end,
			},
			next: start,
		}
	}
	return segs
}

// setState 将分配的段写入状态表
func (s *Sequencer) setState(key string, spec SequenceSpec, segs []*workerSegment) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.states[key] = &sequenceState{
		spec:     spec,
		segments: segs,
	}
}

// getState 返回序列状态，未命中则 false
func (s *Sequencer) getState(key string) (*sequenceState, bool) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	state, ok := s.states[key]
	return state, ok
}

// Reserve 为指定 worker 预留 length 个序列号，必要时触发再切段
func (s *Sequencer) Reserve(ctx context.Context, workerID int, spec SequenceSpec, length int64) (int64, int64, error) {
	if spec.Schema == nil || spec.Field == "" {
		return 0, 0, fmt.Errorf("invalid sequence spec: missing schema or field")
	}
	key := sequenceKey(spec.Schema, spec.Field)
	state, err := s.ensureState(ctx, key, spec)
	if err != nil {
		return 0, 0, err
	}
	for {
		start, end, err := state.reserve(workerID, spec.Step, length)
		if err == nil {
			return start, end, nil
		}
		if !errors.Is(err, ErrSegmentExhausted) {
			return 0, 0, err
		}
		if err = s.Preallocate(ctx, []SequenceSpec{state.spec}); err != nil {
			return 0, 0, err
		}
		state, err = s.ensureState(ctx, key, spec)
		if err != nil {
			return 0, 0, err
		}
	}
}

// ensureState 确保序列状态存在，不存在则重新申请
func (s *Sequencer) ensureState(ctx context.Context, key string, spec SequenceSpec) (*sequenceState, error) {
	if state, ok := s.getState(key); ok {
		return state, nil
	}
	if err := s.Preallocate(ctx, []SequenceSpec{spec}); err != nil {
		return nil, err
	}
	state, ok := s.getState(key)
	if !ok {
		return nil, fmt.Errorf("sequence spec not initialized: %s", key)
	}
	return state, nil
}

// Close 关闭底层 lookup 资源
func (s *Sequencer) Close() error {
	if s.lookup == nil {
		return nil
	}
	return s.lookup.Close()
}

// sequenceKey 将 schema 与字段组合成唯一 key
func sequenceKey(schema schema_store.SchemaKey, field string) string {
	return fmt.Sprintf("%s:%s", schema.UniqueID(), field)
}

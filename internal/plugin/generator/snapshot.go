package generator

import "math/rand/v2"

// SnapshotOption 用于构建策略快照。
type SnapshotOption func(*StrategySnapshot)

// WithCountFixed 设置固定行数。
func WithCountFixed(fixed int) SnapshotOption {
	return func(s *StrategySnapshot) {
		s.Count = &CountSpec{Mode: CountSpecModeFixed, Fixed: fixed}
	}
}

// WithCountRange 设置范围行数（闭区间）。
func WithCountRange(min, max int) SnapshotOption {
	return func(s *StrategySnapshot) {
		s.Count = &CountSpec{Mode: CountSpecModeRange, Min: min, Max: max}
	}
}

// WithInclude 设置字段包含列表。
func WithInclude(fields ...string) SnapshotOption {
	return func(s *StrategySnapshot) {
		s.Fields.addInclude(fields)
	}
}

// WithExclude 设置字段排除列表。
func WithExclude(fields ...string) SnapshotOption {
	return func(s *StrategySnapshot) {
		s.Fields.addExclude(fields)
	}
}

// WithValues 设置字段值策略。
func WithValues(values map[string]ValueSpec) SnapshotOption {
	return func(s *StrategySnapshot) {
		if len(values) == 0 {
			return
		}
		if s.Values == nil {
			s.Values = map[string]ValueSpec{}
		}
		for k, v := range values {
			s.Values[k] = v
		}
	}
}

// WithValue 设置单个字段值策略。
func WithValue(name string, spec ValueSpec) SnapshotOption {
	return func(s *StrategySnapshot) {
		if name == "" {
			return
		}
		if s.Values == nil {
			s.Values = map[string]ValueSpec{}
		}
		s.Values[name] = spec
	}
}

// WithTemplate 设置模板策略。
func WithTemplate(name string, data map[string]any) SnapshotOption {
	return func(s *StrategySnapshot) {
		s.Template = &TemplateSpec{Name: name, Data: data}
	}
}

// WithCustom 设置自定义参数。
func WithCustom(data map[string]any) SnapshotOption {
	return func(s *StrategySnapshot) {
		if len(data) == 0 {
			return
		}
		if s.Custom == nil {
			s.Custom = map[string]any{}
		}
		for k, v := range data {
			s.Custom[k] = v
		}
	}
}

// NewSnapshot 创建策略快照并应用选项。
func NewSnapshot(opts ...SnapshotOption) *StrategySnapshot {
	snap := &StrategySnapshot{
		Values:     map[string]ValueSpec{},
		Custom:     map[string]any{},
		Extensions: map[string]any{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(snap)
		}
	}
	return snap
}

// StrategySnapshot 是生成阶段消费的策略快照。
type StrategySnapshot struct {
	Count      *CountSpec
	Fields     FieldSpec
	Values     map[string]ValueSpec
	Template   *TemplateSpec
	Custom     map[string]any
	Extensions map[string]any
}

// ResolveCount 返回快照对应的行数（默认 1）。
func (s *StrategySnapshot) ResolveCount() int {
	if s == nil || s.Count == nil {
		return 1
	}
	return s.Count.ResolveCount()
}

// FilterFields 根据 include/exclude 过滤字段列表。
func (s *StrategySnapshot) FilterFields(fields []string) []string {
	if len(fields) == 0 || s == nil {
		return fields
	}
	include := s.Fields.Include
	exclude := s.Fields.Exclude
	if len(include) == 0 && len(exclude) == 0 {
		return fields
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		if len(include) > 0 {
			if _, ok := include[field]; !ok {
				continue
			}
		}
		if len(exclude) > 0 {
			if _, ok := exclude[field]; ok {
				continue
			}
		}
		out = append(out, field)
	}
	return out
}

// ApplyValues 根据 ValueSpec 覆盖行数据。
func (s *StrategySnapshot) ApplyValues(row map[string]any) {
	if s == nil || len(s.Values) == 0 || row == nil {
		return
	}
	for key, spec := range s.Values {
		switch spec.Mode {
		case "", "fixed":
			row[key] = spec.Value
		case "template":
			if spec.Name != "" {
				row[key] = spec.Name
			}
		}
	}
}

// TemplateName 返回模板名。
func (s *StrategySnapshot) TemplateName() string {
	if s == nil || s.Template == nil {
		return ""
	}
	return s.Template.Name
}

// TemplateData 返回模板数据。
func (s *StrategySnapshot) TemplateData() map[string]any {
	if s == nil || s.Template == nil {
		return nil
	}
	return s.Template.Data
}

// CustomValue 返回自定义参数。
func (s *StrategySnapshot) CustomValue(key string) (any, bool) {
	if s == nil || len(s.Custom) == 0 {
		return nil, false
	}
	value, ok := s.Custom[key]
	return value, ok
}

// ApplyTemplate 将模板信息写入行数据（默认字段名为 _template/_template_data）。
func (s *StrategySnapshot) ApplyTemplate(row map[string]any) {
	if s == nil || s.Template == nil || row == nil {
		return
	}
	if s.Template.Name != "" {
		row["_template"] = s.Template.Name
	}
	if len(s.Template.Data) > 0 {
		row["_template_data"] = s.Template.Data
	}
}

// ApplyCustom 将自定义参数写入行数据（默认字段名为 _custom）。
func (s *StrategySnapshot) ApplyCustom(row map[string]any) {
	if s == nil || len(s.Custom) == 0 || row == nil {
		return
	}
	row["_custom"] = s.Custom
}

// ApplyAll 依次应用 Values / Template / Custom。
func (s *StrategySnapshot) ApplyAll(row map[string]any) {
	if s == nil || row == nil {
		return
	}
	s.ApplyValues(row)
	s.ApplyTemplate(row)
	s.ApplyCustom(row)
}

// CountSpec 描述行数策略。
type CountSpec struct {
	Mode  string
	Fixed int
	Min   int
	Max   int
}

const (
	CountSpecModeFixed = "fixed"
	CountSpecModeRange = "range"
)

// ResolveCount 根据规则返回行数，默认返回 1。
// Mode=fixed 使用 Fixed；Mode=range 在 [Min, Max] 中随机选一个。
func (c *CountSpec) ResolveCount() int {
	if c == nil {
		return 1
	}
	switch c.Mode {
	case CountSpecModeFixed:
		if c.Fixed > 0 {
			return c.Fixed
		}
	case CountSpecModeRange:
		if c.Min <= 0 && c.Max <= 0 {
			return 1
		}
		if c.Min <= 0 {
			return c.Max
		}
		if c.Max <= 0 || c.Max < c.Min {
			return c.Min
		}
		return rand.IntN(c.Max-c.Min+1) + c.Min
	default:
		if c.Fixed > 0 {
			return c.Fixed
		}
		if c.Min > 0 || c.Max > 0 {
			if c.Max > 0 && c.Max >= c.Min {
				return rand.IntN(c.Max-c.Min+1) + c.Min
			}
			if c.Min > 0 {
				return c.Min
			}
			return c.Max
		}
	}
	return 1
}

// ValueSpec 描述字段值策略。
type ValueSpec struct {
	Mode  string
	Value any
	Name  string
}

// TemplateSpec 描述模板策略。
type TemplateSpec struct {
	Name string
	Data map[string]any
}

// FieldSpec 表示字段筛选的最终结果。
// Include/Exclude 都是集合，用于避免重复。
type FieldSpec struct {
	Include map[string]struct{}
	Exclude map[string]struct{}
}

func (f *FieldSpec) addInclude(fields []string) {
	if len(fields) == 0 {
		return
	}
	if f.Include == nil {
		f.Include = make(map[string]struct{}, len(fields))
	}
	for _, field := range fields {
		if field == "" {
			continue
		}
		f.Include[field] = struct{}{}
	}
}

func (f *FieldSpec) addExclude(fields []string) {
	if len(fields) == 0 {
		return
	}
	if f.Exclude == nil {
		f.Exclude = make(map[string]struct{}, len(fields))
	}
	for _, field := range fields {
		if field == "" {
			continue
		}
		f.Exclude[field] = struct{}{}
	}
}

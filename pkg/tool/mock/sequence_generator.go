package mock

// SequenceColumn 定义序列生成的列配置。
// Type 仅支持 "int"/"string"；Length 用于字符串长度；Digits 用于数字位数（10^Digits-1）。
// Start/Max 仅对 int 生效，Max<=0 表示按 Digits 推算。
type SequenceColumn struct {
	Name   string
	Type   string
	Length int
	Digits int
	Start  int64
	Max    int64
}

const (
	SequenceTypeString = "string"
	SequenceTypeInt    = "int"
)

type SequenceGeneratorConfig struct {
	StringLength int
	IntDigits    int
	Wrap         bool
}

type SequenceConfigOption func(*SequenceGeneratorConfig)

func defaultSequenceGeneratorConfig() SequenceGeneratorConfig {
	return SequenceGeneratorConfig{
		StringLength: 8,
		IntDigits:    8,
		Wrap:         false,
	}
}

func WithStringLength(length int) SequenceConfigOption {
	return func(cfg *SequenceGeneratorConfig) { cfg.StringLength = length }
}

func WithIntDigits(digits int) SequenceConfigOption {
	return func(cfg *SequenceGeneratorConfig) { cfg.IntDigits = digits }
}

func WithWrap(enabled bool) SequenceConfigOption {
	return func(cfg *SequenceGeneratorConfig) { cfg.Wrap = enabled }
}

func applySequenceOptions(opts ...SequenceConfigOption) SequenceGeneratorConfig {
	cfg := defaultSequenceGeneratorConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.StringLength <= 0 {
		cfg.StringLength = 8
	}
	if cfg.IntDigits <= 0 {
		cfg.IntDigits = 8
	}
	return cfg
}

// SequenceRange 表示闭区间范围。
type SequenceRange struct {
	Start int64
	End   int64
}

func (r SequenceRange) Valid() bool { return r.End >= r.Start }

// RangeSequence 针对单列 int 场景，按需返回连续范围。
// NextRange 的 bool 表示本次调用是否到达序列末尾（wrap=true 时表示已回绕）。
type RangeSequence interface {
	Column() string
	NextRange(need int) (SequenceRange, bool)
	Reset()
}

// RowSequence 针对字符串或联合唯一场景，按需返回行列表。
type RowSequence interface {
	Columns() []string
	NextRows(need int) ([]map[string]any, bool)
	Reset()
}

// ValueGenerator 生成单列标量值。
type ValueGenerator interface {
	Next() (any, bool)
	Reset()
}

type intRangeSequence struct {
	column  string
	start   int64
	max     int64
	cursor  int64
	started bool
	wrap    bool
}

func newIntRangeSequence(column string, start, max int64, wrap bool) *intRangeSequence {
	return &intRangeSequence{
		column: column,
		start:  start,
		max:    max,
		wrap:   wrap,
	}
}

func (s *intRangeSequence) Column() string { return s.column }

func (s *intRangeSequence) Reset() { s.started = false }

func (s *intRangeSequence) NextRange(need int) (SequenceRange, bool) {
	if need <= 0 {
		need = 1
	}
	if !s.started {
		s.cursor = s.start
		s.started = true
	} else {
		s.cursor++
	}

	if s.max > 0 && s.cursor > s.max {
		if s.wrap {
			s.cursor = s.start
		} else {
			return SequenceRange{}, true
		}
	}

	start := s.cursor
	end := start + int64(need) - 1
	if s.max > 0 && end >= s.max {
		end = s.max
		if s.wrap {
			s.cursor = s.start - 1
		} else {
			return SequenceRange{Start: start, End: end}, true
		}
	} else {
		s.cursor = end
	}
	return SequenceRange{Start: start, End: end}, false
}

// NewRangeSequence 从单列 int 配置构造范围生成器。
func NewRangeSequence(column SequenceColumn, opts ...SequenceConfigOption) (RangeSequence, error) {
	cfg := applySequenceOptions(opts...)
	if column.Type != SequenceTypeInt || column.Name == "" {
		return nil, ErrInvalidSequenceColumn
	}
	digits := column.Digits
	if digits <= 0 {
		digits = cfg.IntDigits
	}
	maxV := column.Max
	if maxV <= 0 {
		maxV = intDigitsMax(digits)
	}
	return newIntRangeSequence(column.Name, column.Start, maxV, cfg.Wrap), nil
}

// NewRowSequence 从字符串/联合唯一列配置构造行生成器。
func NewRowSequence(columns []SequenceColumn, opts ...SequenceConfigOption) (RowSequence, error) {
	cfg := applySequenceOptions(opts...)
	if len(columns) == 0 {
		return nil, ErrInvalidSequenceColumn
	}
	colNames := make([]string, 0, len(columns))
	gens := make([]ValueGenerator, 0, len(columns))
	for _, col := range columns {
		if col.Name == "" {
			return nil, ErrInvalidSequenceColumn
		}
		switch col.Type {
		case SequenceTypeString:
			length := col.Length
			if length <= 0 {
				length = cfg.StringLength
			}
			gens = append(gens, NewStringValueGenerator(length))
		case SequenceTypeInt:
			digits := col.Digits
			if digits <= 0 {
				digits = cfg.IntDigits
			}
			gens = append(gens, NewInt64DigitsGenerator(col.Start, digits, cfg.Wrap))
		default:
			return nil, ErrInvalidSequenceColumn
		}
		colNames = append(colNames, col.Name)
	}
	return newRowSequence(colNames, gens), nil
}

// ErrInvalidSequenceColumn 表示列配置非法。
var ErrInvalidSequenceColumn = errInvalidSequenceColumn{}

type errInvalidSequenceColumn struct{}

func (errInvalidSequenceColumn) Error() string { return "invalid sequence column" }

// Int64ValueGenerator 生成连续数值。
type Int64ValueGenerator struct {
	start   int64
	max     int64
	step    int64
	cur     int64
	started bool
	wrap    bool
}

// NewInt64ValueGenerator 创建数值生成器。
// max<=0 表示无上限；wrap=true 表示到达上限后回绕。
func NewInt64ValueGenerator(start, max, step int64, wrap bool) *Int64ValueGenerator {
	if step <= 0 {
		step = 1
	}
	return &Int64ValueGenerator{
		start: start,
		max:   max,
		step:  step,
		wrap:  wrap,
	}
}

func (g *Int64ValueGenerator) Reset() { g.started = false }

func (g *Int64ValueGenerator) Next() (any, bool) {
	if !g.started {
		g.cur = g.start
		g.started = true
		return g.cur, true
	}
	next := g.cur + g.step
	if g.max > 0 && next > g.max {
		if !g.wrap {
			return nil, false
		}
		next = g.start
	}
	g.cur = next
	return g.cur, true
}

// NewInt64ValueGeneratorWithCursor 创建数值生成器并设置当前值。
// 下一次返回将是 current+step（或在 wrap 时回绕）。
func NewInt64ValueGeneratorWithCursor(start, max, step int64, wrap bool, current int64) *Int64ValueGenerator {
	gen := NewInt64ValueGenerator(start, max, step, wrap)
	gen.cur = current
	gen.started = true
	return gen
}

// NewInt64DigitsGenerator 创建固定位数生成器（最大值为 10^digits-1）。
func NewInt64DigitsGenerator(start int64, digits int, wrap bool) *Int64ValueGenerator {
	max := intDigitsMax(digits)
	return NewInt64ValueGenerator(start, max, 1, wrap)
}

// NewInt64DigitsGeneratorWithCursor 创建固定位数生成器并设置当前值。
func NewInt64DigitsGeneratorWithCursor(start int64, digits int, wrap bool, current int64) *Int64ValueGenerator {
	max := intDigitsMax(digits)
	return NewInt64ValueGeneratorWithCursor(start, max, 1, wrap, current)
}

func intDigitsMax(digits int) int64 {
	if digits <= 0 {
		digits = 8
	}
	if digits > 18 {
		digits = 18
	}
	max := int64(1)
	for i := 0; i < digits; i++ {
		max *= 10
	}
	return max - 1
}

// StringValueGenerator 生成 [a..z] 字典序字符串。
type StringValueGenerator struct {
	length  int
	current []byte
	started bool
}

// NewStringValueGenerator 创建固定长度的字典序生成器。
// 起始为 "aaaa..."，结束为 "zzzz..."，结束后返回 false。
func NewStringValueGenerator(length int) *StringValueGenerator {
	if length <= 0 {
		length = 8
	}
	return &StringValueGenerator{
		length:  length,
		current: make([]byte, length),
	}
}

func (g *StringValueGenerator) Reset() { g.started = false }

func (g *StringValueGenerator) Next() (any, bool) {
	if !g.started {
		for i := range g.current {
			g.current[i] = 'a'
		}
		g.started = true
		return string(g.current), true
	}
	for i := len(g.current) - 1; i >= 0; i-- {
		if g.current[i] < 'z' {
			g.current[i]++
			for j := i + 1; j < len(g.current); j++ {
				g.current[j] = 'a'
			}
			return string(g.current), true
		}
	}
	return nil, false
}

// NewStringValueGeneratorWithCursor 创建字符串生成器并设置当前值。
// 下一次返回将是 current 的字典序后继。
func NewStringValueGeneratorWithCursor(length int, current string) (*StringValueGenerator, error) {
	gen := NewStringValueGenerator(length)
	if len(current) != length {
		return nil, ErrInvalidSequenceColumn
	}
	copy(gen.current, current)
	gen.started = true
	return gen, nil
}

type rowSequence struct {
	columns []string
	gens    []ValueGenerator
	values  []any
	started bool
}

func newRowSequence(columns []string, gens []ValueGenerator) *rowSequence {
	return &rowSequence{
		columns: append([]string(nil), columns...),
		gens:    append([]ValueGenerator(nil), gens...),
		values:  make([]any, len(columns)),
	}
}

// NewRowSequenceFromGenerators creates a row sequence from explicit generators.
func NewRowSequenceFromGenerators(columns []string, gens []ValueGenerator) (RowSequence, error) {
	if len(columns) == 0 || len(columns) != len(gens) {
		return nil, ErrInvalidSequenceColumn
	}
	for _, col := range columns {
		if col == "" {
			return nil, ErrInvalidSequenceColumn
		}
	}
	for _, gen := range gens {
		if gen == nil {
			return nil, ErrInvalidSequenceColumn
		}
	}
	return newRowSequence(columns, gens), nil
}

// NewRowSequenceFromGeneratorsWithCursor creates a row sequence seeded at cursor values.
// cursor must match the columns length and each generator must already be at that value.
func NewRowSequenceFromGeneratorsWithCursor(columns []string, gens []ValueGenerator, cursor []any) (RowSequence, error) {
	if len(cursor) != len(columns) {
		return nil, ErrInvalidSequenceColumn
	}
	seq, err := NewRowSequenceFromGenerators(columns, gens)
	if err != nil {
		return nil, err
	}
	rs, ok := seq.(*rowSequence)
	if !ok {
		return nil, ErrInvalidSequenceColumn
	}
	copy(rs.values, cursor)
	rs.started = true
	return rs, nil
}

func (s *rowSequence) Columns() []string { return append([]string(nil), s.columns...) }

func (s *rowSequence) Reset() {
	s.started = false
	for _, gen := range s.gens {
		gen.Reset()
	}
}

func (s *rowSequence) NextRows(need int) ([]map[string]any, bool) {
	if need <= 0 {
		need = 1
	}
	rows := make([]map[string]any, 0, need)
	for i := 0; i < need; i++ {
		row, ok := s.nextRow()
		if !ok {
			return rows, true
		}
		rows = append(rows, row)
	}
	return rows, false
}

func (s *rowSequence) nextRow() (map[string]any, bool) {
	if len(s.columns) == 0 || len(s.columns) != len(s.gens) {
		return nil, false
	}
	if !s.started {
		for i, gen := range s.gens {
			val, ok := gen.Next()
			if !ok {
				return nil, false
			}
			s.values[i] = val
		}
		s.started = true
		return s.valuesMap(), true
	}

	for i := len(s.gens) - 1; i >= 0; i-- {
		val, ok := s.gens[i].Next()
		if !ok {
			continue
		}
		s.values[i] = val
		for j := i + 1; j < len(s.gens); j++ {
			s.gens[j].Reset()
			nextVal, nextOk := s.gens[j].Next()
			if !nextOk {
				return nil, false
			}
			s.values[j] = nextVal
		}
		return s.valuesMap(), true
	}
	return nil, false
}

func (s *rowSequence) valuesMap() map[string]any {
	out := make(map[string]any, len(s.columns))
	for i, col := range s.columns {
		out[col] = s.values[i]
	}
	return out
}

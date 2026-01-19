package lookup

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/transform"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/tool/mock"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

const (
	Memory iquery.LookupType = "memory"
)

func init() {
	iquery.RegisterIQuery(Memory, &MemoryLookup{}, false)
}

type MemoryLookup struct {
	pipeline string
	cfg      MemoryLookupConfig

	mu sync.Mutex
	// maxBySchemaColumn tracks "existing" max value for bounds.
	maxBySchemaColumn map[string]map[string]int64
	rowSeqBySchemaKey map[string]mock.RowSequence
	loopSeqByKey      map[string]mock.RangeSequence
}

type MemoryLookupConfig struct {
	StringLength int  `mapstructure:"string-length" json:"string-length"`
	IntDigits    int  `mapstructure:"int-digits" json:"int-digits"`
	Wrap         bool `mapstructure:"wrap" json:"wrap"`
}

func (q *MemoryLookup) Configure(pipeline string, data map[string]any) error {
	q.pipeline = pipeline
	q.cfg = MemoryLookupConfig{StringLength: 8, IntDigits: 8}
	if len(data) > 0 {
		if err := mapstructure.Decode(data, &q.cfg); err != nil {
			return err
		}
	}
	if q.cfg.StringLength <= 0 {
		q.cfg.StringLength = 8
	}
	if q.cfg.IntDigits <= 0 {
		q.cfg.IntDigits = 8
	}
	q.maxBySchemaColumn = make(map[string]map[string]int64)
	q.rowSeqBySchemaKey = make(map[string]mock.RowSequence)
	q.loopSeqByKey = make(map[string]mock.RangeSequence)
	return nil
}

func (q *MemoryLookup) LookupBounds(ctx context.Context, req iquery.LookupRequest) (iquery.LookupResult, error) {
	_ = ctx
	q.mu.Lock()
	defer q.mu.Unlock()

	sid := req.Schema.UniqueID()
	if _, ok := q.maxBySchemaColumn[sid]; !ok {
		q.maxBySchemaColumn[sid] = make(map[string]int64)
	}
	if len(req.Params) == 0 || req.Params[0].Column == "" {
		return iquery.LookupResult{}, fmt.Errorf("lookup params empty")
	}

	partition := req.Partition
	if partition == "" {
		partition = range_pool.RangePoolLiveName
	}
	need := req.Need
	if need <= 0 {
		need = 1
	}

	col := req.Params[0].Column
	maxV := q.maxBySchemaColumn[sid][col]
	if partition == range_pool.RangePoolFreeName {
		wrapEnabled := req.WrapAt > 0 && maxV >= req.WrapAt
		if wrapEnabled {
			seqKey := sid + ":" + col
			seq := q.loopSeqByKey[seqKey]
			if seq == nil {
				seq, _ = mock.NewRangeSequence(mock.SequenceColumn{
					Name:   col,
					Type:   mock.SequenceTypeInt,
					Start:  1,
					Max:    req.WrapAt,
					Digits: q.cfg.IntDigits,
				}, mock.WithWrap(true))
				q.loopSeqByKey[seqKey] = seq
			}
			win, _ := seq.NextRange(int(need))
			if !win.Valid() {
				return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
			}
			if win.End > maxV {
				q.maxBySchemaColumn[sid][col] = win.End
			}
			return iquery.LookupResult{
				EnableLoop: true,
				Window:     range_pool.IntRange{Start: win.Start, End: win.End},
			}, nil
		}

		seq, err := mock.NewRangeSequence(mock.SequenceColumn{
			Name:   col,
			Type:   mock.SequenceTypeInt,
			Start:  maxV + 1,
			Max:    req.WrapAt,
			Digits: q.cfg.IntDigits,
		})
		if err != nil {
			return iquery.LookupResult{}, err
		}
		win, done := seq.NextRange(int(need))
		if !win.Valid() {
			return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
		}
		if win.End > maxV {
			q.maxBySchemaColumn[sid][col] = win.End
		}
		_ = done
		return iquery.LookupResult{Window: range_pool.IntRange{Start: win.Start, End: win.End}}, nil
	}

	return iquery.LookupResult{Window: range_pool.IntRange{Start: 0, End: maxV}}, nil
}

func (q *MemoryLookup) ScanValues(ctx context.Context, req iquery.ValuesRequest) (iquery.ValuesResult, error) {
	_ = ctx
	if len(req.Columns) == 0 {
		return iquery.ValuesResult{}, nil
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 1000
	}

	seqKey := q.rowSeqKey(req.Schema.UniqueID(), req.Columns)
	q.mu.Lock()
	defer q.mu.Unlock()
	seq := q.rowSeqBySchemaKey[seqKey]
	if seq == nil {
		columns, err := buildSequenceColumns(req.Columns, req.Cursor, q.cfg)
		if err != nil {
			return iquery.ValuesResult{}, err
		}
		seq, err = newRowSequenceWithCursor(columns, req.Cursor, q.cfg)
		if err != nil {
			return iquery.ValuesResult{}, err
		}
		q.rowSeqBySchemaKey[seqKey] = seq
	}
	if req.Cursor == nil {
		seq.Reset()
	}

	rows, done := seq.NextRows(limit)
	if len(rows) == 0 {
		return iquery.ValuesResult{HasMore: !done}, nil
	}

	cols := make([]string, 0, len(req.Columns))
	for _, c := range req.Columns {
		cols = append(cols, c.Column)
	}
	out := make([][]any, 0, len(rows))
	for _, row := range rows {
		vals := make([]any, len(cols))
		for i, col := range cols {
			vals[i] = row[col]
		}
		out = append(out, vals)
	}
	result := iquery.ValuesResult{Rows: out, HasMore: !done}
	if len(out) > 0 {
		last := out[len(out)-1]
		if len(last) == 1 {
			result.NextCursor = last[0]
		} else {
			result.NextCursor = last
		}
	}
	return result, nil
}

func (q *MemoryLookup) Close() error { return nil }

func (q *MemoryLookup) rowSeqKey(schemaID string, cols []iquery.BoundParam) string {
	parts := make([]string, 0, len(cols)+1)
	parts = append(parts, schemaID)
	for _, c := range cols {
		parts = append(parts, c.Column+":"+sequenceTypeFromColumn(c.Type))
	}
	return strings.Join(parts, "|")
}

func buildSequenceColumns(cols []iquery.BoundParam, cursor any, cfg MemoryLookupConfig) ([]mock.SequenceColumn, error) {
	seqCols := make([]mock.SequenceColumn, 0, len(cols))
	var cursorValue int64
	cursorOK := false
	if cursor != nil && isNumericType(cols[0].Type) {
		if v, err := transform.ToInt(cursor); err == nil {
			cursorValue = v
			cursorOK = true
		}
	}
	for i, c := range cols {
		if c.Column == "" {
			return nil, fmt.Errorf("empty column name")
		}
		seqCol := mock.SequenceColumn{
			Name: c.Column,
			Type: sequenceTypeFromColumn(c.Type),
		}
		if seqCol.Type == mock.SequenceTypeString {
			seqCol.Length = cfg.StringLength
		}
		if seqCol.Type == mock.SequenceTypeInt {
			seqCol.Digits = cfg.IntDigits
		}
		if i == 0 && cursorOK {
			seqCol.Start = cursorValue + 1
		}
		seqCols = append(seqCols, seqCol)
	}
	return seqCols, nil
}

func newRowSequenceWithCursor(columns []mock.SequenceColumn, cursor any, cfg MemoryLookupConfig) (mock.RowSequence, error) {
	if cursor == nil {
		return mock.NewRowSequence(columns, mock.WithWrap(cfg.Wrap))
	}
	row, ok := cursor.([]any)
	if !ok || len(row) != len(columns) {
		return mock.NewRowSequence(columns, mock.WithWrap(cfg.Wrap))
	}
	colNames := make([]string, 0, len(columns))
	gens := make([]mock.ValueGenerator, 0, len(columns))
	for i, col := range columns {
		colNames = append(colNames, col.Name)
		switch col.Type {
		case mock.SequenceTypeString:
			val, ok := row[i].(string)
			if !ok {
				return mock.NewRowSequence(columns, mock.WithWrap(cfg.Wrap))
			}
			length := col.Length
			if length <= 0 {
				length = cfg.StringLength
			}
			gen, err := mock.NewStringValueGeneratorWithCursor(length, val)
			if err != nil {
				return mock.NewRowSequence(columns, mock.WithWrap(cfg.Wrap))
			}
			gens = append(gens, gen)
		case mock.SequenceTypeInt:
			val, err := transform.ToInt(row[i])
			if err != nil {
				return mock.NewRowSequence(columns, mock.WithWrap(cfg.Wrap))
			}
			digits := col.Digits
			if digits <= 0 {
				digits = cfg.IntDigits
			}
			gens = append(gens, mock.NewInt64DigitsGeneratorWithCursor(col.Start, digits, cfg.Wrap, val))
		default:
			return mock.NewRowSequence(columns, mock.WithWrap(cfg.Wrap))
		}
	}
	return mock.NewRowSequenceFromGeneratorsWithCursor(colNames, gens, row)
}

func sequenceTypeFromColumn(typ string) string {
	if isNumericType(typ) {
		return mock.SequenceTypeInt
	}
	return mock.SequenceTypeString
}

func isNumericType(typ string) bool {
	if typ == "" {
		return true
	}
	t := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case strings.Contains(t, "int"),
		strings.Contains(t, "decimal"),
		strings.Contains(t, "numeric"),
		strings.Contains(t, "float"),
		strings.Contains(t, "double"):
		return true
	default:
		return false
	}
}

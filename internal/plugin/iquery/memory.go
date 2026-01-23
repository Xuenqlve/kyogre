package iquery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/pkg/tool/mock"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

const (
	Memory LookupType = "memory"
)

func init() {
	RegisterIQuery(Memory, &MemoryLookup{}, false)
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

func (q *MemoryLookup) LookupRange(ctx context.Context, req Request) (RangeResult, error) {
	_ = ctx
	q.mu.Lock()
	defer q.mu.Unlock()

	sid := req.Schema.UniqueID()
	if _, ok := q.maxBySchemaColumn[sid]; !ok {
		q.maxBySchemaColumn[sid] = make(map[string]int64)
	}
	if len(req.Columns) == 0 || req.Columns[0].Column == "" {
		return RangeResult{}, fmt.Errorf("lookup params empty")
	}

	need := req.Need
	if need <= 0 {
		need = 1
	}

	col := req.Columns[0].Column
	maxV := q.maxBySchemaColumn[sid][col]
	if req.Cursor != nil {
		if v, err := toInt64(req.Cursor); err == nil {
			if v > maxV {
				maxV = v
			}
		} else {
			return RangeResult{}, err
		}
	}
	if maxV > q.maxBySchemaColumn[sid][col] {
		q.maxBySchemaColumn[sid][col] = maxV
	}
	wrapAt := hardMaxInt64(req.Columns[0].Type)
	start := maxV + 1
	if wrapAt > 0 && start > wrapAt {
		if !q.cfg.Wrap {
			return RangeResult{}, fmt.Errorf("lookup range exhausted for %s: start=%d wrapAt=%d", req.Schema.UniqueID(), start, wrapAt)
		}
		seqKey := sid + ":" + col
		seq := q.loopSeqByKey[seqKey]
		if seq == nil {
			seq, _ = mock.NewRangeSequence(mock.SequenceColumn{
				Name:   col,
				Type:   mock.SequenceTypeInt,
				Start:  1,
				Max:    wrapAt,
				Digits: q.cfg.IntDigits,
			}, mock.WithWrap(true))
			q.loopSeqByKey[seqKey] = seq
		}
		win, _ := seq.NextRange(int(need))
		if !win.Valid() {
			return RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
		}
		if win.End > maxV {
			q.maxBySchemaColumn[sid][col] = win.End
		}
		return RangeResult{
			EnableLoop: true,
			Window:     range_pool.IntRange{Start: win.Start, End: win.End},
		}, nil
	}

	end := start + need - 1
	if wrapAt > 0 && end > wrapAt {
		end = wrapAt
	}
	if end < start {
		return RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
	}
	if end > maxV {
		q.maxBySchemaColumn[sid][col] = end
	}
	return RangeResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
}

func (q *MemoryLookup) ScanValues(ctx context.Context, req Request) (ValuesResult, error) {
	if len(req.Columns) == 0 {
		return ValuesResult{}, nil
	}
	limit := int(req.Need)
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
			return ValuesResult{}, err
		}
		seq, err = newRowSequenceWithCursor(columns, req.Cursor, q.cfg)
		if err != nil {
			return ValuesResult{}, err
		}
		q.rowSeqBySchemaKey[seqKey] = seq
	}
	if req.Cursor == nil {
		seq.Reset()
	}

	rows, done := seq.NextRows(limit)
	if len(rows) == 0 {
		return ValuesResult{HasMore: !done}, nil
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
	result := ValuesResult{Rows: out, HasMore: !done}
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

func (q *MemoryLookup) rowSeqKey(schemaID string, cols []ColumnParam) string {
	parts := make([]string, 0, len(cols)+1)
	parts = append(parts, schemaID)
	for _, c := range cols {
		parts = append(parts, c.Column+":"+sequenceTypeFromColumn(c.Type))
	}
	return strings.Join(parts, "|")
}

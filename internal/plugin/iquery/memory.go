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

	mu                sync.Mutex
	rowSeqBySchemaKey map[string]mock.RowSequence
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
	q.rowSeqBySchemaKey = make(map[string]mock.RowSequence)
	return nil
}

func (q *MemoryLookup) LookupRange(ctx context.Context, req RangeRequest) (RangeResult, error) {
	_ = ctx

	if len(req.Columns) == 0 || req.Columns[0].Column == "" {
		return RangeResult{}, fmt.Errorf("lookup params empty")
	}

	need := req.Need
	if need <= 0 {
		need = 1
	}

	wrapAt := hardMaxInt64(req.Columns[0].Type)
	if wrapAt > 0 && req.Cursor >= wrapAt {
		if !q.cfg.Wrap {
			return RangeResult{}, fmt.Errorf("lookup range exhausted for %s: start=%d wrapAt=%d", req.Schema.UniqueID(), req.Cursor+1, wrapAt)
		}
		start := int64(1)
		end := start + need - 1
		if end > wrapAt {
			end = wrapAt
		}
		if end < start {
			return RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
		}
		return RangeResult{
			EnableLoop: true,
			Window:     range_pool.IntRange{Start: start, End: end},
		}, nil
	}

	start := req.Cursor + 1
	end := start + need - 1
	if wrapAt > 0 && end > wrapAt {
		end = wrapAt
	}
	if end < start {
		return RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
	}
	return RangeResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
}

func (q *MemoryLookup) ScanValues(ctx context.Context, req ValueRequest) (ValuesResult, error) {
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

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, row)
	}
	result := ValuesResult{Rows: out, HasMore: !done}
	if len(out) > 0 {
		result.NextCursor = out[len(out)-1]
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

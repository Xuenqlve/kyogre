package iquery

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

const (
	Memory iquery.LookupType = "memory"
)

func init() {
	iquery.RegisterIQuery(Memory, &MemoryLookup{}, false)
}

type MemoryLookup struct {
	pipeline string

	mu sync.Mutex
	// maxBySchemaColumn tracks "existing" max value for bounds.
	maxBySchemaColumn map[string]map[string]int64
}

func (q *MemoryLookup) Configure(pipeline string, _ map[string]any) error {
	q.pipeline = pipeline
	q.maxBySchemaColumn = make(map[string]map[string]int64)
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

	out := iquery.LookupResult{Bounds: make([]iquery.Bound, 0, len(req.Params))}
	for _, p := range req.Params {
		if p.Column == "" {
			continue
		}
		maxV := q.maxBySchemaColumn[sid][p.Column]
		out.Bounds = append(out.Bounds, iquery.Bound{
			BoundParam: p,
			MinValue:   int64(0),
			MaxValue:   maxV,
			Count:      int(maxV),
		})
	}
	return out, nil
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

	var cursor int64
	switch v := req.Cursor.(type) {
	case nil:
		cursor = 0
	case int64:
		cursor = v
	case int:
		cursor = int64(v)
	default:
		return iquery.ValuesResult{}, fmt.Errorf("memory ScanValues unsupported cursor type %T", req.Cursor)
	}

	rows := make([][]any, 0, limit)
	for i := 0; i < limit; i++ {
		val := cursor + int64(i) + 1
		row := make([]any, len(req.Columns))
		for j := range row {
			row[j] = val
		}
		rows = append(rows, row)
	}
	return iquery.ValuesResult{
		Rows:       rows,
		NextCursor: cursor + int64(limit),
		HasMore:    true,
	}, nil
}

func (q *MemoryLookup) Close() error { return nil }


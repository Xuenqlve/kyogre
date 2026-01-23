package iquery

//const (
//	Memory iquery.LookupType = "memory"
//)
//
//func init() {
//	iquery.RegisterIQuery(Memory, &MemoryLookup{}, false)
//}

//type MemoryLookup struct {
//	pipeline string
//
//	mu sync.Mutex
//	// maxBySchemaColumn tracks "existing" max value for bounds.
//	maxBySchemaColumn map[string]map[string]int64
//}
//
//func (q *MemoryLookup) Configure(pipeline string, _ map[string]any) error {
//	q.pipeline = pipeline
//	q.maxBySchemaColumn = make(map[string]map[string]int64)
//	return nil
//}
//
//func (q *MemoryLookup) LookupBounds(ctx context.Context, req iquery.LookupRequest) (iquery.LookupResult, error) {
//	_ = ctx
//	q.mu.Lock()
//	defer q.mu.Unlock()
//
//	sid := req.Schema.UniqueID()
//	if _, ok := q.maxBySchemaColumn[sid]; !ok {
//		q.maxBySchemaColumn[sid] = make(map[string]int64)
//	}
//	if len(req.Params) == 0 || req.Params[0].Column == "" {
//		return iquery.LookupResult{}, fmt.Errorf("lookup params empty")
//	}
//
//	partition := req.Partition
//	if partition == "" {
//		partition = range_pool.RangePoolLiveName
//	}
//	need := req.Need
//	if need <= 0 {
//		need = 1
//	}
//
//	maxV := q.maxBySchemaColumn[sid][req.Params[0].Column]
//	if partition == range_pool.RangePoolFreeName {
//		enableLoop := req.WrapAt > 0 && maxV >= req.WrapAt
//		if enableLoop {
//			start := int64(1)
//			end := start + need - 1
//			if req.WrapAt > 0 && end > req.WrapAt {
//				end = req.WrapAt
//			}
//			if end < start {
//				return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
//			}
//			return iquery.LookupResult{
//				EnableLoop: true,
//				Window:     range_pool.IntRange{Start: start, End: end},
//			}, nil
//		}
//		start := maxV + 1
//		if req.WrapAt > 0 && start > req.WrapAt {
//			return iquery.LookupResult{}, fmt.Errorf("insert range exhausted for %s: start=%d wrapAt=%d", req.Schema.UniqueID(), start, req.WrapAt)
//		}
//		end := start + need - 1
//		if req.WrapAt > 0 && end > req.WrapAt {
//			end = req.WrapAt
//		}
//		if end < start {
//			return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
//		}
//		return iquery.LookupResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
//	}
//
//	return iquery.LookupResult{Window: range_pool.IntRange{Start: 0, End: maxV}}, nil
//}
//
//func (q *MemoryLookup) ScanValues(ctx context.Context, req iquery.ValuesRequest) (iquery.ValuesResult, error) {
//	_ = ctx
//	if len(req.Columns) == 0 {
//		return iquery.ValuesResult{}, nil
//	}
//	limit := req.Limit
//	if limit <= 0 {
//		limit = 1000
//	}
//
//	var cursor int64
//	switch v := req.Cursor.(type) {
//	case nil:
//		cursor = 0
//	case int64:
//		cursor = v
//	case int:
//		cursor = int64(v)
//	default:
//		return iquery.ValuesResult{}, fmt.Errorf("memory ScanValues unsupported cursor type %T", req.Cursor)
//	}
//
//	rows := make([][]any, 0, limit)
//	for i := 0; i < limit; i++ {
//		val := cursor + int64(i) + 1
//		row := make([]any, len(req.Columns))
//		for j := range row {
//			row[j] = val
//		}
//		rows = append(rows, row)
//	}
//	return iquery.ValuesResult{
//		Rows:       rows,
//		NextCursor: cursor + int64(limit),
//		HasMore:    true,
//	}, nil
//}
//
//func (q *MemoryLookup) Close() error { return nil }

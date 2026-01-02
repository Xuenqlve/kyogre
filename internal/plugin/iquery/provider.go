package iquery

import "fmt"

// RowProvider yields rows in the form of map[column]value, typically used to build
// WHERE ... IN (...) predicates for update/delete, or fixed key columns for insert.
type RowProvider interface {
	Columns() []string
	NextBatch(n int) ([]map[string]any, bool)
}

type rangeProvider struct {
	columns []string
	current int64
	end     int64
	step    int64
}

func newRangeProvider(column string, start, end, step int64) *rangeProvider {
	if step <= 0 {
		step = 1
	}
	return &rangeProvider{
		columns: []string{column},
		current: start,
		end:     end,
		step:    step,
	}
}

func (p *rangeProvider) Columns() []string { return append([]string(nil), p.columns...) }

func (p *rangeProvider) NextBatch(n int) ([]map[string]any, bool) {
	if n <= 0 {
		n = 1
	}
	if p.current > p.end {
		return nil, false
	}
	out := make([]map[string]any, 0, n)
	for i := 0; i < n && p.current <= p.end; i++ {
		row := map[string]any{p.columns[0]: p.current}
		out = append(out, row)
		p.current += p.step
	}
	return out, len(out) > 0
}

type tupleProvider struct {
	columns []string
	rows    [][]any
	idx     int
}

func newTupleProvider(columns []string, rows [][]any) (*tupleProvider, error) {
	cols := make([]string, 0, len(columns))
	for _, c := range columns {
		if c != "" {
			cols = append(cols, c)
		}
	}
	for i := range rows {
		if len(rows[i]) != len(cols) {
			return nil, fmt.Errorf("tuple width mismatch: got %d want %d", len(rows[i]), len(cols))
		}
	}
	return &tupleProvider{columns: cols, rows: rows}, nil
}

func (p *tupleProvider) Columns() []string { return append([]string(nil), p.columns...) }

func (p *tupleProvider) NextBatch(n int) ([]map[string]any, bool) {
	if n <= 0 {
		n = 1
	}
	if p.idx >= len(p.rows) {
		return nil, false
	}
	end := p.idx + n
	if end > len(p.rows) {
		end = len(p.rows)
	}
	out := make([]map[string]any, 0, end-p.idx)
	for ; p.idx < end; p.idx++ {
		rowMap := make(map[string]any, len(p.columns))
		for j, col := range p.columns {
			rowMap[col] = p.rows[p.idx][j]
		}
		out = append(out, rowMap)
	}
	return out, len(out) > 0
}


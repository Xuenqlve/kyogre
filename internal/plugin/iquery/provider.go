package iquery

import "fmt"

type Provider interface {
	Columns() []string
	Rows() []map[string]any
}

// Provider yields rows in the form of map[column]value, typically used to build
// WHERE ... IN (...) predicates for update/delete, or fixed key columns for insert.
type Provider interface {
	Columns() []string
	Rows() []map[string]any
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

func (p *rangeProvider) Rows() []map[string]any {
	if p.current > p.end {
		return nil
	}
	step := p.step
	if step <= 0 {
		step = 1
	}
	total := int((p.end-p.current)/step) + 1
	if total < 0 {
		total = 0
	}
	out := make([]map[string]any, 0, total)
	for p.current <= p.end {
		out = append(out, map[string]any{p.columns[0]: p.current})
		p.current += step
	}
	return out
}

type tupleProvider struct {
	columns []string
	rows    [][]any
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

func (p *tupleProvider) Rows() []map[string]any {
	if len(p.rows) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(p.rows))
	for i := range p.rows {
		rowMap := make(map[string]any, len(p.columns))
		for j, col := range p.columns {
			rowMap[col] = p.rows[i][j]
		}
		out = append(out, rowMap)
	}
	return out
}

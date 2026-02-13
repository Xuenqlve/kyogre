package iquery

import "fmt"

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
	rows    []map[string]any
}

func newTupleProvider(columns []string, rows []map[string]any) (*tupleProvider, error) {
	cols := make([]string, 0, len(columns))
	for _, c := range columns {
		if c != "" {
			cols = append(cols, c)
		}
	}
	for i := range rows {
		for _, col := range cols {
			if _, ok := rows[i][col]; !ok {
				return nil, fmt.Errorf("tuple missing column %q in row", col)
			}
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
		rowMap := make(map[string]any, len(p.rows[i]))
		for k, v := range p.rows[i] {
			rowMap[k] = v
		}
		out = append(out, rowMap)
	}
	return out
}

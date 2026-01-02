package iquery

import (
	"context"
	"fmt"

	"github.com/xuenqlve/common/schema_store"
)

type aliveSet struct {
	capacity int
	batch    int

	columns []BoundParam
	cursor  any

	// rows with a moving offset to avoid O(n) shifts.
	rows   [][]any
	offset int
}

func newAliveSet(columns []BoundParam, capacity, batch int) *aliveSet {
	if capacity <= 0 {
		capacity = 10000
	}
	if batch <= 0 {
		batch = 1000
	}
	cols := make([]BoundParam, 0, len(columns))
	for _, c := range columns {
		if c.Column != "" {
			cols = append(cols, c)
		}
	}
	return &aliveSet{
		capacity: capacity,
		batch:    batch,
		columns:  cols,
	}
}

func (s *aliveSet) available() int {
	if s.offset >= len(s.rows) {
		return 0
	}
	return len(s.rows) - s.offset
}

func (s *aliveSet) compactIfNeeded() {
	if s.offset == 0 {
		return
	}
	// Compact when offset becomes significant.
	if s.offset < 1024 && s.offset*2 < len(s.rows) {
		return
	}
	s.rows = append([][]any(nil), s.rows[s.offset:]...)
	s.offset = 0
}

func (s *aliveSet) dropOldest(extra int) {
	if extra <= 0 {
		return
	}
	s.offset += extra
	if s.offset > len(s.rows) {
		s.offset = len(s.rows)
	}
	s.compactIfNeeded()
}

func (s *aliveSet) appendRows(rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	width := len(s.columns)
	for i := range rows {
		if len(rows[i]) != width {
			return fmt.Errorf("aliveSet tuple width mismatch: got %d want %d", len(rows[i]), width)
		}
		s.rows = append(s.rows, rows[i])
	}
	avail := s.available()
	if avail > s.capacity {
		s.dropOldest(avail - s.capacity)
	}
	return nil
}

func (s *aliveSet) refill(ctx context.Context, lookup Lookup, schema schema_store.SchemaKey, minNeed int) error {
	for s.available() < minNeed {
		res, err := lookup.ScanValues(ctx, ValuesRequest{
			Schema:  schema,
			Columns: s.columns,
			Cursor:  s.cursor,
			Limit:   s.batch,
		})
		if err != nil {
			return err
		}
		if err := s.appendRows(res.Rows); err != nil {
			return err
		}
		s.cursor = res.NextCursor
		if !res.HasMore {
			// Wrap-around scan for pressure scenario.
			s.cursor = nil
			// If still empty after exhausting, break to avoid infinite loop.
			if len(res.Rows) == 0 {
				break
			}
		}
		// If scan returns empty repeatedly, stop.
		if len(res.Rows) == 0 {
			break
		}
	}
	return nil
}

func (s *aliveSet) take(ctx context.Context, lookup Lookup, schema schema_store.SchemaKey, n int) ([][]any, error) {
	if n <= 0 {
		n = 1
	}
	if err := s.refill(ctx, lookup, schema, n); err != nil {
		return nil, err
	}
	if s.available() == 0 {
		return nil, nil
	}
	if n > s.available() {
		n = s.available()
	}
	out := append([][]any(nil), s.rows[s.offset:s.offset+n]...)
	s.offset += n
	s.compactIfNeeded()
	return out, nil
}

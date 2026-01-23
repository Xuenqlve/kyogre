package iquery

import (
	"fmt"
	"math"
	"strings"

	"github.com/xuenqlve/common/transform"
	"github.com/xuenqlve/kyogre/pkg/tool/mock"
)

func sequenceTypeFromColumn(typ string) string {
	if isNumericType(typ) {
		return mock.SequenceTypeInt
	}
	return mock.SequenceTypeString
}

func hardMaxInt64(typ string) int64 {
	if typ == "" {
		return math.MaxInt64
	}
	t := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case strings.Contains(t, "bigint unsigned"):
		return math.MaxInt64 // cannot represent full uint64 in int64, cap for safety
	case strings.Contains(t, "bigint"):
		return math.MaxInt64
	case strings.Contains(t, "int unsigned"):
		return math.MaxInt32
	case strings.Contains(t, "int"):
		return math.MaxInt32
	case strings.Contains(t, "smallint unsigned"):
		return math.MaxInt16
	case strings.Contains(t, "smallint"):
		return math.MaxInt16
	case strings.Contains(t, "tinyint unsigned"):
		return math.MaxInt8
	case strings.Contains(t, "tinyint"):
		return math.MaxInt8
	default:
		return math.MaxInt64
	}
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

func minInt64(a, b int64) int64 {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case nil:
		return 0, nil
	case int:
		return int64(n), nil
	case int8:
		return int64(n), nil
	case int16:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case int64:
		return n, nil
	case uint:
		return int64(n), nil
	case uint8:
		return int64(n), nil
	case uint16:
		return int64(n), nil
	case uint32:
		return int64(n), nil
	case uint64:
		if n > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("uint64 overflow: %d", n)
		}
		return int64(n), nil
	case float32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case string:
		return 0, fmt.Errorf("string cannot convert to int64: %q", n)
	default:
		return 0, fmt.Errorf("unsupported type %T to int64", v)
	}
}

func normalizeSize(need int64, allowed []int64, strict bool) (int64, error) {
	if need <= 0 {
		return 1, nil
	}
	if len(allowed) == 0 {
		return need, nil
	}
	if strict {
		for _, s := range allowed {
			if s == need {
				return need, nil
			}
		}
		return 0, fmt.Errorf("size %d not allowed", need)
	}
	// round up
	best := int64(0)
	for _, s := range allowed {
		if s >= need && (best == 0 || s < best) {
			best = s
		}
	}
	if best == 0 {
		return need, nil
	}
	return best, nil
}

func buildSequenceColumns(cols []ColumnParam, cursor any, cfg MemoryLookupConfig) ([]mock.SequenceColumn, error) {
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

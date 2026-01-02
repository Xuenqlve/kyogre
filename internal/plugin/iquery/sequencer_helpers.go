package iquery

import (
	"fmt"
)

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


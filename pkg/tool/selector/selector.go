package selector

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Strategy string

const (
	RoundRobin Strategy = "round-robin"
	Random     Strategy = "random"
	Weighted   Strategy = "weighted"
)

type Intn interface {
	Intn(n int) int
}

type Selector[T any] interface {
	Pick() (T, error)
	Len() int
	Strategy() Strategy
}

type Option[T any] struct {
	Value  T
	Weight int
}

type Config[T any] struct {
	Strategy Strategy
	Items    []Option[T]
	Rand     Intn
}

func OptionsFromValues[T any](values ...T) []Option[T] {
	items := make([]Option[T], 0, len(values))
	for _, value := range values {
		items = append(items, Option[T]{Value: value, Weight: 1})
	}
	return items
}

func New[T any](cfg Config[T]) (Selector[T], error) {
	if len(cfg.Items) == 0 {
		return nil, fmt.Errorf("selector items are empty")
	}
	switch cfg.Strategy {
	case "", RoundRobin:
		return &roundRobinSelector[T]{items: extractValues(cfg.Items)}, nil
	case Random:
		return &randomSelector[T]{
			items: extractValues(cfg.Items),
			rng:   normalizeRand(cfg.Rand),
		}, nil
	case Weighted:
		return newWeightedSelector(cfg.Items, cfg.Rand)
	default:
		return nil, fmt.Errorf("unsupported selector strategy: %s", cfg.Strategy)
	}
}

func extractValues[T any](items []Option[T]) []T {
	values := make([]T, 0, len(items))
	for _, item := range items {
		values = append(values, item.Value)
	}
	return values
}

func normalizeRand(r Intn) Intn {
	if r != nil {
		return r
	}
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

type roundRobinSelector[T any] struct {
	items  []T
	cursor atomic.Uint64
}

func (s *roundRobinSelector[T]) Pick() (T, error) {
	var zero T
	if len(s.items) == 0 {
		return zero, fmt.Errorf("selector items are empty")
	}
	idx := int(s.cursor.Add(1)-1) % len(s.items)
	return s.items[idx], nil
}

func (s *roundRobinSelector[T]) Len() int { return len(s.items) }

func (s *roundRobinSelector[T]) Strategy() Strategy { return RoundRobin }

type randomSelector[T any] struct {
	items []T
	rng   Intn
	mu    sync.Mutex
}

func (s *randomSelector[T]) Pick() (T, error) {
	var zero T
	if len(s.items) == 0 {
		return zero, fmt.Errorf("selector items are empty")
	}
	s.mu.Lock()
	idx := s.rng.Intn(len(s.items))
	s.mu.Unlock()
	return s.items[idx], nil
}

func (s *randomSelector[T]) Len() int { return len(s.items) }

func (s *randomSelector[T]) Strategy() Strategy { return Random }

type weightedSelector[T any] struct {
	items  []Option[T]
	prefix []int
	total  int
	rng    Intn
	randMu sync.Mutex
}

func newWeightedSelector[T any](items []Option[T], rng Intn) (*weightedSelector[T], error) {
	prefix := make([]int, 0, len(items))
	total := 0
	for i := range items {
		if items[i].Weight <= 0 {
			return nil, fmt.Errorf("selector item[%d] weight must be greater than zero", i)
		}
		total += items[i].Weight
		prefix = append(prefix, total)
	}
	if total <= 0 {
		return nil, fmt.Errorf("selector total weight must be greater than zero")
	}
	return &weightedSelector[T]{
		items:  append([]Option[T](nil), items...),
		prefix: prefix,
		total:  total,
		rng:    normalizeRand(rng),
	}, nil
}

func (s *weightedSelector[T]) Pick() (T, error) {
	var zero T
	if len(s.items) == 0 {
		return zero, fmt.Errorf("selector items are empty")
	}
	s.randMu.Lock()
	pick := s.rng.Intn(s.total)
	s.randMu.Unlock()
	for i, boundary := range s.prefix {
		if pick < boundary {
			return s.items[i].Value, nil
		}
	}
	return zero, fmt.Errorf("selector weighted pick out of range: %d", pick)
}

func (s *weightedSelector[T]) Len() int { return len(s.items) }

func (s *weightedSelector[T]) Strategy() Strategy { return Weighted }

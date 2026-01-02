package iquery

import "fmt"

type SequencerOptions func(*SequencerConfig)

type SequencerConfig struct {
	// LookUpKey is a logical name of the lookup instance (mainly for debugging/metrics).
	LookUpKey string

	// InsertWindowSize controls how many new (high-water) ids are reserved in memory
	// before forcing another lookup refresh.
	InsertWindowSize int64

	// StrictSizes enforces that each Reserve* call uses one of AllowedSizes.
	StrictSizes bool

	// AllowedSizes defines the "ladder" sizes (e.g. 1,5,10,20 or 100,1000,10000).
	// When StrictSizes is false, it is used to round up to the nearest size.
	AllowedSizes []int64

	// WrapAt, when >0, enables cyclic reuse for inserts. Inserts will try to wrap
	// into the already-deleted low region [BaseMin, ExistMin).
	WrapAt int64
}

func WithLookUpKey(key string) SequencerOptions {
	return func(cfg *SequencerConfig) { cfg.LookUpKey = key }
}

func WithInsertWindowSize(size int64) SequencerOptions {
	return func(cfg *SequencerConfig) { cfg.InsertWindowSize = size }
}

func WithAllowedSizes(sizes ...int64) SequencerOptions {
	return func(cfg *SequencerConfig) { cfg.AllowedSizes = append([]int64(nil), sizes...) }
}

func WithStrictSizes(strict bool) SequencerOptions {
	return func(cfg *SequencerConfig) { cfg.StrictSizes = strict }
}

func WithWrapAt(max int64) SequencerOptions {
	return func(cfg *SequencerConfig) { cfg.WrapAt = max }
}

func (cfg *SequencerConfig) ValidateAndSetDefault() error {
	if cfg.InsertWindowSize <= 0 {
		cfg.InsertWindowSize = 10000
	}
	for _, s := range cfg.AllowedSizes {
		if s <= 0 {
			return fmt.Errorf("invalid allowed size %d", s)
		}
	}
	return nil
}


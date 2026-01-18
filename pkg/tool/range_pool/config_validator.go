package range_pool

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// defaultRangePoolConfig returns the default tier ladder used for both partitions.
func defaultRangePoolConfig() RangePoolConfig {
	tiers := []TierConfig{
		{Size: 1, Threshold: 5, MaxCount: 10},
		{Size: 5, Threshold: 6, MaxCount: 10},
		{Size: 10, Threshold: 6, MaxCount: 10},
		{Size: 20, Threshold: 5, MaxCount: 10},
		{Size: 100, Threshold: 5, MaxCount: 10},
		{Size: 500, Threshold: 2, MaxCount: 5},
	}
	return RangePoolConfig{LiveTiers: tiers, FreeTiers: tiers}
}

// rand returns a *rand.Rand constructed from cfg.RandSource, defaulting to a time-based seed.
func (cfg *RangePoolConfig) rand() *rand.Rand {
	src := cfg.RandSource
	if src == nil {
		src = rand.NewSource(time.Now().UnixNano())
	}
	return rand.New(src)
}

// ValidateAndSetDefault ensures tiers are ready for use.
func (cfg *RangePoolConfig) ValidateAndSetDefault() error {
	if len(cfg.LiveTiers) == 0 {
		cfg.LiveTiers = defaultRangePoolConfig().LiveTiers
	}
	if len(cfg.FreeTiers) == 0 {
		cfg.FreeTiers = defaultRangePoolConfig().FreeTiers
	}
	if cfg.Refill == nil {
		return fmt.Errorf("missing register RangePool refill func")
	}
	_, err := validateTierConfigs(RangePoolLiveName, cfg.LiveTiers)
	if err != nil {
		return err
	}
	_, err = validateTierConfigs(RangePoolFreeName, cfg.FreeTiers)
	if err != nil {
		return err
	}
	if err = validateTierDivisible(RangePoolLiveName, cfg.LiveTiers); err != nil {
		return err
	}
	if err = validateTierDivisible(RangePoolFreeName, cfg.FreeTiers); err != nil {
		return err
	}
	return nil
}

// validateTierConfigs performs lightweight validation that is independent of partition construction.
func validateTierConfigs(name string, tiers []TierConfig) (map[int64]struct{}, error) {
	if len(tiers) == 0 {
		return nil, fmt.Errorf("%s partition requires at least one tier", name)
	}
	sizes := make(map[int64]struct{}, len(tiers))
	for _, tier := range tiers {
		if tier.Size <= 0 {
			return nil, fmt.Errorf("%s tier %d has invalid size", name, tier.Size)
		}
		if tier.Threshold < 0 {
			return nil, fmt.Errorf("%s tier %d threshold must be >= 0", name, tier.Size)
		}
		if tier.MaxCount < 0 {
			return nil, fmt.Errorf("%s tier %d max count must be >= 0", name, tier.Size)
		}
		if tier.MaxCount > 0 && tier.MaxCount <= tier.Threshold {
			return nil, fmt.Errorf("%s tier %d max count must be > threshold", name, tier.Size)
		}
		if _, exists := sizes[tier.Size]; exists {
			return nil, fmt.Errorf("%s tier size %d duplicated", name, tier.Size)
		}
		sizes[tier.Size] = struct{}{}
	}
	return sizes, nil
}

func validateTierDivisible(name string, tiers []TierConfig) error {
	if len(tiers) == 0 {
		return fmt.Errorf("%s partition requires at least one tier", name)
	}
	sorted := append([]TierConfig(nil), tiers...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Size < sorted[j].Size })
	for i := 1; i < len(sorted); i++ {
		lo := sorted[i-1].Size
		hi := sorted[i].Size
		if lo == 0 || hi%lo != 0 {
			return fmt.Errorf("%s tier size %d must be divisible by %d", name, hi, lo)
		}
	}
	return nil
}

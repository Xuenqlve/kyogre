package base

import (
	"fmt"

	"github.com/xuenqlve/kyogre/pkg/tool/selector"
)

type SelectorFactory struct {
	rand selector.Intn
}

func NewSelectorFactory(rng selector.Intn) *SelectorFactory {
	return &SelectorFactory{rand: rng}
}

func (f *SelectorFactory) Target(cfg TargetSelectorConfig, targets []Target) (selector.Selector[Target], error) {
	options := make([]selector.Option[Target], 0, len(targets))
	allowed := make(map[string]struct{}, len(cfg.Schemas))
	for _, name := range cfg.Schemas {
		if name != "" {
			allowed[name] = struct{}{}
		}
	}
	weights := make(map[string]int, len(cfg.Items)+len(cfg.Weights))
	for _, item := range cfg.Items {
		weights[item.Value] = item.Weight
	}
	for key, weight := range cfg.Weights {
		weights[key] = weight
	}
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("invalid target %s: %w", target.Key, err)
		}
		if len(allowed) > 0 {
			if _, ok := allowed[target.Key]; !ok {
				continue
			}
		}
		weight := 1
		if cfg.Strategy == string(selector.Weighted) {
			if v, ok := weights[target.Key]; ok && v > 0 {
				weight = v
			}
		}
		options = append(options, selector.Option[Target]{Value: target.Clone(), Weight: weight})
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("target selector has no available targets")
	}
	return selector.New(selector.Config[Target]{
		Strategy: selector.Strategy(cfg.Strategy),
		Items:    options,
		Rand:     f.rand,
	})
}

func (f *SelectorFactory) Operation(cfg ValueSelectorConfig) (selector.Selector[string], error) {
	options, err := buildStringOptions(cfg)
	if err != nil {
		return nil, err
	}
	return selector.New(selector.Config[string]{
		Strategy: selector.Strategy(cfg.Strategy),
		Items:    options,
		Rand:     f.rand,
	})
}

func (f *SelectorFactory) Int(cfg IntSelectorConfig) (selector.Selector[int], error) {
	options, err := buildIntOptions(cfg)
	if err != nil {
		return nil, err
	}
	return selector.New(selector.Config[int]{
		Strategy: selector.Strategy(cfg.Strategy),
		Items:    options,
		Rand:     f.rand,
	})
}

func buildStringOptions(cfg ValueSelectorConfig) ([]selector.Option[string], error) {
	options := make([]selector.Option[string], 0, len(cfg.Items)+len(cfg.Values)+len(cfg.Weights))
	if len(cfg.Items) > 0 {
		for _, item := range cfg.Items {
			weight := item.Weight
			if weight <= 0 {
				weight = 1
			}
			options = append(options, selector.Option[string]{Value: item.Value, Weight: weight})
		}
		return options, nil
	}
	if len(cfg.Weights) > 0 {
		for value, weight := range cfg.Weights {
			options = append(options, selector.Option[string]{Value: value, Weight: weight})
		}
		return options, nil
	}
	for _, value := range cfg.Values {
		options = append(options, selector.Option[string]{Value: value, Weight: 1})
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("string selector options are empty")
	}
	return options, nil
}

func buildIntOptions(cfg IntSelectorConfig) ([]selector.Option[int], error) {
	options := make([]selector.Option[int], 0, len(cfg.Items)+len(cfg.Values))
	if len(cfg.Items) > 0 {
		for _, item := range cfg.Items {
			weight := item.Weight
			if weight <= 0 {
				weight = 1
			}
			options = append(options, selector.Option[int]{Value: item.Value, Weight: weight})
		}
		return options, nil
	}
	for _, value := range cfg.Values {
		options = append(options, selector.Option[int]{Value: value, Weight: 1})
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("int selector options are empty")
	}
	return options, nil
}

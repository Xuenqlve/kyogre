package base

import (
	"fmt"

	"github.com/xuenqlve/kyogre/pkg/tool/selector"
)

type Config struct {
	Builder                 string               `mapstructure:"builder"`
	Mode                    string               `mapstructure:"mode"`
	MessageCount            int                  `mapstructure:"message-count"`
	IntervalMS              int                  `mapstructure:"interval-ms"`
	Columns                 []string             `mapstructure:"columns"`
	Hint                    string               `mapstructure:"hint"`
	WriteType               string               `mapstructure:"write-type"`
	TargetSelector          TargetSelectorConfig `mapstructure:"target-selector"`
	OperationSelector       ValueSelectorConfig  `mapstructure:"operation-selector"`
	RowCountSelector        IntSelectorConfig    `mapstructure:"row-count-selector"`
	TransactionSizeSelector IntSelectorConfig    `mapstructure:"transaction-size-selector"`
	Lookup                  LookupConfig         `mapstructure:"lookup"`
}

type TargetSelectorConfig struct {
	Strategy string          `mapstructure:"strategy"`
	Schemas  []string        `mapstructure:"schemas"`
	Items    []WeightedValue `mapstructure:"items"`
	Weights  map[string]int  `mapstructure:"weights"`
}

type ValueSelectorConfig struct {
	Strategy string          `mapstructure:"strategy"`
	Items    []WeightedValue `mapstructure:"items"`
	Values   []string        `mapstructure:"values"`
	Weights  map[string]int  `mapstructure:"weights"`
}

type IntSelectorConfig struct {
	Strategy string        `mapstructure:"strategy"`
	Items    []WeightedInt `mapstructure:"items"`
	Values   []int         `mapstructure:"values"`
	Fixed    int           `mapstructure:"fixed"`
	Min      int           `mapstructure:"min"`
	Max      int           `mapstructure:"max"`
}

type LookupConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	Operations []string `mapstructure:"operations"`
}

type WeightedValue struct {
	Value  string `mapstructure:"value"`
	Weight int    `mapstructure:"weight"`
}

type WeightedInt struct {
	Value  int `mapstructure:"value"`
	Weight int `mapstructure:"weight"`
}

func (c *Config) Normalize() error {
	if c.Builder == "" {
		return fmt.Errorf("base scenario builder is empty")
	}
	if c.Mode == "" {
		c.Mode = ModeRow
	}
	switch c.Mode {
	case ModeRow, ModeTransaction:
	default:
		return fmt.Errorf("unsupported base scenario mode: %s", c.Mode)
	}
	if c.MessageCount <= 0 {
		c.MessageCount = 1
	}
	if c.IntervalMS < 0 {
		c.IntervalMS = 0
	}
	if err := c.TargetSelector.Normalize(); err != nil {
		return err
	}
	if err := c.OperationSelector.NormalizeWithDefault("insert"); err != nil {
		return err
	}
	if err := c.RowCountSelector.NormalizeAsCount(1, 1, 0); err != nil {
		return err
	}
	if c.Mode == ModeTransaction {
		if err := c.TransactionSizeSelector.NormalizeAsCount(2, 1, 0); err != nil {
			return err
		}
	}
	if err := c.Lookup.Normalize(); err != nil {
		return err
	}
	return nil
}

func (c *TargetSelectorConfig) Normalize() error {
	if c.Strategy == "" {
		c.Strategy = string(selector.RoundRobin)
	}
	switch selector.Strategy(c.Strategy) {
	case selector.RoundRobin, selector.Random, selector.Weighted:
	default:
		return fmt.Errorf("unsupported target selector strategy: %s", c.Strategy)
	}
	if selector.Strategy(c.Strategy) == selector.Weighted && len(c.Items) == 0 && len(c.Weights) == 0 {
		return fmt.Errorf("weighted target selector requires items or weights")
	}
	for i := range c.Items {
		if c.Items[i].Value == "" {
			return fmt.Errorf("target selector item[%d] value is empty", i)
		}
		if selector.Strategy(c.Strategy) == selector.Weighted && c.Items[i].Weight <= 0 {
			return fmt.Errorf("target selector item[%d] weight must be greater than zero", i)
		}
	}
	for key, weight := range c.Weights {
		if key == "" {
			return fmt.Errorf("target selector weights contains empty key")
		}
		if weight <= 0 {
			return fmt.Errorf("target selector weight for %s must be greater than zero", key)
		}
	}
	return nil
}

func (c *ValueSelectorConfig) NormalizeWithDefault(defaultValue string) error {
	if c.Strategy == "" {
		c.Strategy = string(selector.RoundRobin)
	}
	switch selector.Strategy(c.Strategy) {
	case selector.RoundRobin, selector.Random, selector.Weighted:
	default:
		return fmt.Errorf("unsupported value selector strategy: %s", c.Strategy)
	}
	if len(c.Items) == 0 && len(c.Values) == 0 && len(c.Weights) == 0 {
		c.Values = []string{defaultValue}
	}
	if selector.Strategy(c.Strategy) == selector.Weighted {
		if len(c.Items) == 0 && len(c.Weights) == 0 {
			return fmt.Errorf("weighted value selector requires items or weights")
		}
	}
	for i := range c.Items {
		if c.Items[i].Value == "" {
			return fmt.Errorf("value selector item[%d] value is empty", i)
		}
		if selector.Strategy(c.Strategy) == selector.Weighted && c.Items[i].Weight <= 0 {
			return fmt.Errorf("value selector item[%d] weight must be greater than zero", i)
		}
	}
	for i := range c.Values {
		if c.Values[i] == "" {
			return fmt.Errorf("value selector values[%d] is empty", i)
		}
	}
	for value, weight := range c.Weights {
		if value == "" {
			return fmt.Errorf("value selector weights contains empty key")
		}
		if weight <= 0 {
			return fmt.Errorf("value selector weight for %s must be greater than zero", value)
		}
	}
	return nil
}

func (c *IntSelectorConfig) NormalizeAsCount(defaultFixed, minValue, maxValue int) error {
	if c.Strategy == "" {
		if c.Fixed > 0 {
			c.Strategy = string(selector.RoundRobin)
		} else if c.Min > 0 || c.Max > 0 {
			c.Strategy = string(selector.Random)
		} else {
			c.Strategy = string(selector.RoundRobin)
		}
	}
	switch selector.Strategy(c.Strategy) {
	case selector.RoundRobin, selector.Random, selector.Weighted:
	default:
		return fmt.Errorf("unsupported int selector strategy: %s", c.Strategy)
	}
	if c.Fixed <= 0 && len(c.Items) == 0 && len(c.Values) == 0 && c.Min <= 0 && c.Max <= 0 {
		c.Fixed = defaultFixed
	}
	if c.Fixed > 0 && len(c.Values) == 0 && len(c.Items) == 0 {
		c.Values = []int{c.Fixed}
	}
	if c.Min > 0 || c.Max > 0 {
		if c.Min <= 0 {
			c.Min = minValue
		}
		if c.Max <= 0 {
			c.Max = c.Min
		}
		if c.Max < c.Min {
			return fmt.Errorf("int selector max must be greater than or equal to min")
		}
		if len(c.Values) == 0 && len(c.Items) == 0 {
			values := make([]int, 0, c.Max-c.Min+1)
			for value := c.Min; value <= c.Max; value++ {
				values = append(values, value)
			}
			c.Values = values
		}
	}
	if selector.Strategy(c.Strategy) == selector.Weighted && len(c.Items) == 0 {
		return fmt.Errorf("weighted int selector requires items")
	}
	for i := range c.Items {
		if c.Items[i].Value <= 0 {
			return fmt.Errorf("int selector item[%d] value must be greater than zero", i)
		}
		if selector.Strategy(c.Strategy) == selector.Weighted && c.Items[i].Weight <= 0 {
			return fmt.Errorf("int selector item[%d] weight must be greater than zero", i)
		}
	}
	for i := range c.Values {
		if c.Values[i] <= 0 {
			return fmt.Errorf("int selector values[%d] must be greater than zero", i)
		}
	}
	if maxValue > 0 && len(c.Values) > 0 {
		for i := range c.Values {
			if c.Values[i] > maxValue {
				return fmt.Errorf("int selector values[%d] exceeds max value %d", i, maxValue)
			}
		}
	}
	return nil
}

func (c *LookupConfig) Normalize() error {
	if !c.Enabled {
		c.Operations = nil
		return nil
	}
	if len(c.Operations) == 0 {
		c.Operations = []string{"update", "delete"}
	}
	for i := range c.Operations {
		if c.Operations[i] == "" {
			return fmt.Errorf("lookup operations[%d] is empty", i)
		}
	}
	return nil
}

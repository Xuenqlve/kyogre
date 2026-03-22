package base

import (
	"fmt"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

const (
	ModeRow         = "row"
	ModeTransaction = "transaction"
)

// Target 描述一次计划对应的目标对象。
// Key 是统一的逻辑标识，Schema 由具体 builder 解析为数据库专属结构。
type Target struct {
	Key          string
	Schema       any
	SequenceSpec *iquery.SequenceSpec
	Extras       map[string]any
}

func (t Target) Validate() error {
	if t.Key == "" {
		return fmt.Errorf("target key is empty")
	}
	if t.Schema == nil {
		return fmt.Errorf("target schema is nil")
	}
	return nil
}

func (t Target) Clone() Target {
	out := Target{
		Key:    t.Key,
		Schema: t.Schema,
	}
	if t.SequenceSpec != nil {
		spec := *t.SequenceSpec
		if len(spec.Fields) > 0 {
			spec.Fields = append([]iquery.ColumnParam(nil), spec.Fields...)
		}
		out.SequenceSpec = &spec
	}
	if len(t.Extras) > 0 {
		out.Extras = make(map[string]any, len(t.Extras))
		for k, v := range t.Extras {
			out.Extras[k] = v
		}
	}
	return out
}

// Plan 是 base scenario 生成的一次通用执行计划。
// 具体数据库通过 builder 将其转换为对应的 GenerationContext。
type Plan struct {
	Builder         string
	Mode            string
	Operation       string
	Target          Target
	RowsPerMessage  int
	TransactionSize int
	Columns         []string
	Hint            string
	WriteType       string
	Providers       map[string]iquery.Provider
	Extras          map[string]any
}

func (p Plan) Validate() error {
	if p.Builder == "" {
		return fmt.Errorf("plan builder is empty")
	}
	switch p.Mode {
	case "", ModeRow:
	case ModeTransaction:
		if p.TransactionSize <= 0 {
			return fmt.Errorf("plan transaction size must be greater than zero")
		}
	default:
		return fmt.Errorf("unsupported plan mode: %s", p.Mode)
	}
	if p.Operation == "" {
		return fmt.Errorf("plan operation is empty")
	}
	if err := p.Target.Validate(); err != nil {
		return fmt.Errorf("invalid plan target: %w", err)
	}
	if p.RowsPerMessage <= 0 {
		return fmt.Errorf("plan rows per message must be greater than zero")
	}
	return nil
}

func (p Plan) Clone() Plan {
	out := Plan{
		Builder:         p.Builder,
		Mode:            p.Mode,
		Operation:       p.Operation,
		Target:          p.Target.Clone(),
		RowsPerMessage:  p.RowsPerMessage,
		TransactionSize: p.TransactionSize,
		Hint:            p.Hint,
		WriteType:       p.WriteType,
	}
	if len(p.Columns) > 0 {
		out.Columns = append([]string(nil), p.Columns...)
	}
	if len(p.Providers) > 0 {
		out.Providers = make(map[string]iquery.Provider, len(p.Providers))
		for k, v := range p.Providers {
			out.Providers[k] = v
		}
	}
	if len(p.Extras) > 0 {
		out.Extras = make(map[string]any, len(p.Extras))
		for k, v := range p.Extras {
			out.Extras[k] = v
		}
	}
	return out
}

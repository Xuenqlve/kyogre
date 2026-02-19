package mock

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	message2 "github.com/xuenqlve/kyogre/pkg/message"
)

const Mock generator.Type = "mock"

type Config struct {
	Prefix string `mapstructure:"prefix"`
}

type Generator struct {
	pipeline string
	cfg      Config
	seq      atomic.Int64
}

func init() {
	generator.RegisterGenerator(Mock, &Generator{}, false)
}

func (g *Generator) Configure(pipeline string, data map[string]any) error {
	g.pipeline = pipeline
	if err := mapstructure.Decode(data, &g.cfg); err != nil {
		return errors.Trace(err)
	}
	if g.cfg.Prefix == "" {
		g.cfg.Prefix = "mock"
	}
	return nil
}

func (g *Generator) Kinds() []string {
	return []string{genctx.Mock}
}

func (g *Generator) Generate(ctx generator.GenerationContext) (message.Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Validate(); err != nil {
		return nil, err
	}
	snap := ctx.Strategy()
	rows := g.buildRows(ctx, snap)
	return &message2.MockMessage{Rows: rows, CreatedAt: time.Now()}, nil
}

func (g *Generator) Close() {}

func (g *Generator) buildRows(ctx generator.GenerationContext, snap *generator.StrategySnapshot) []message2.MockRow {
	count := 1
	if snap != nil {
		count = snap.ResolveCount()
	}
	if count <= 0 {
		return nil
	}
	rows := make([]message2.MockRow, 0, count)
	for i := 0; i < count; i++ {
		row := cloneRow(baseRow(ctx, g.seq.Add(1)))
		applyFieldSpec(row, snap)
		snap.ApplyAll(row)
		if snap != nil {
			if name := snap.TemplateName(); name != "" {
				row["_template_name"] = name
			}
			if data := snap.TemplateData(); len(data) > 0 {
				row["_template_data_copy"] = data
			}
			if v, ok := snap.CustomValue("suffix"); ok {
				row["_suffix"] = v
			}
		}
		rows = append(rows, message2.MockRow{Value: row})
	}
	return rows
}

func baseRow(ctx generator.GenerationContext, seed int64) map[string]any {
	if typed, ok := ctx.(*genctx.MockContext); ok {
		if len(typed.Value) > 0 {
			return typed.Value
		}
	}
	return map[string]any{
		"   ": seed,
	}
}

func cloneRow(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func applyFieldSpec(row map[string]any, snap *generator.StrategySnapshot) {
	if snap == nil {
		return
	}
	filtered := snap.FilterFields(mapKeys(row))
	if len(filtered) == 0 && len(row) > 0 {
		for key := range row {
			delete(row, key)
		}
		return
	}
	keep := make(map[string]struct{}, len(filtered))
	for _, key := range filtered {
		keep[key] = struct{}{}
	}
	for key := range row {
		if _, ok := keep[key]; !ok {
			delete(row, key)
		}
	}
}

func mapKeys(row map[string]any) []string {
	if len(row) == 0 {
		return nil
	}
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	return keys
}

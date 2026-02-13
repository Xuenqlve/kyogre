package generator_context

import (
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

func NewBaseContext(kind string, opts ...ContextOption) *baseContext {
	ctx := &baseContext{
		kind: kind,
	}
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

type ContextOption func(*baseContext)

func WithIQueryProvider(provider iquery.Provider) ContextOption {
	return func(ctx *baseContext) { ctx.provider = provider }
}

func WithStrategy(snapshot *generator.StrategySnapshot) ContextOption {
	return func(ctx *baseContext) { ctx.strategy = snapshot }
}

func WithExtras(extras map[string]any) ContextOption {
	return func(ctx *baseContext) { ctx.extras = extras }
}

type baseContext struct {
	kind     string
	provider iquery.Provider
	strategy *generator.StrategySnapshot
	extras   map[string]any
}

func (b *baseContext) Kind() string {
	return b.kind
}

func (b *baseContext) Provider() iquery.Provider {
	return b.provider
}

func (b *baseContext) Strategy() *generator.StrategySnapshot {
	return b.strategy
}

func (b *baseContext) Validate() error {
	return nil
}

func (b *baseContext) Extras() map[string]any {
	return b.extras
}

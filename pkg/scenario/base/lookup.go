package base

import (
	"context"
	"fmt"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/message"
)

const TargetExtraSequenceSpec = "sequence_spec"

type ReserveSequencer interface {
	ReserveInsert(ctx context.Context, spec iquery.SequenceSpec, need int64) (iquery.Provider, error)
	ReserveUpdate(ctx context.Context, spec iquery.SequenceSpec, need int64) (iquery.Provider, error)
	ReserveDelete(ctx context.Context, spec iquery.SequenceSpec, need int64) (iquery.Provider, error)
}

type LookupBinder struct {
	cfg LookupConfig
	seq ReserveSequencer
}

func NewLookupBinder(cfg LookupConfig, seq ReserveSequencer) *LookupBinder {
	return &LookupBinder{cfg: cfg, seq: seq}
}

func (b *LookupBinder) Providers(ctx context.Context, plan Plan) (map[string]iquery.Provider, error) {
	if !b.cfg.EnabledFor(plan.Operation) {
		return nil, nil
	}
	if b.seq == nil {
		return nil, fmt.Errorf("lookup binder sequencer is nil for operation %s", plan.Operation)
	}
	spec, err := sequenceSpecFromTarget(plan.Target)
	if err != nil {
		return nil, err
	}
	need := providerNeed(plan)
	var provider iquery.Provider
	switch plan.Operation {
	case message.Insert, message.InsertIgnore, message.InsertOnDuplicateKey, message.Replace:
		provider, err = b.seq.ReserveInsert(ctx, spec, need)
	case message.Update, message.UpdateJoin:
		provider, err = b.seq.ReserveUpdate(ctx, spec, need)
	case message.Delete:
		provider, err = b.seq.ReserveDelete(ctx, spec, need)
	default:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, nil
	}
	return map[string]iquery.Provider{
		spec.Schema.UniqueID(): provider,
	}, nil
}

func providerNeed(plan Plan) int64 {
	need := plan.RowsPerMessage
	if need <= 0 {
		need = 1
	}
	if plan.Mode == ModeTransaction && plan.TransactionSize > 0 {
		need *= plan.TransactionSize
	}
	return int64(need)
}

func sequenceSpecFromTarget(target Target) (iquery.SequenceSpec, error) {
	if len(target.Extras) == 0 {
		return iquery.SequenceSpec{}, fmt.Errorf("target %s missing sequence spec", target.Key)
	}
	value, ok := target.Extras[TargetExtraSequenceSpec]
	if !ok {
		return iquery.SequenceSpec{}, fmt.Errorf("target %s missing sequence spec", target.Key)
	}
	spec, ok := value.(iquery.SequenceSpec)
	if !ok {
		return iquery.SequenceSpec{}, fmt.Errorf("target %s sequence spec type mismatch: %T", target.Key, value)
	}
	if spec.Schema == nil {
		return iquery.SequenceSpec{}, fmt.Errorf("target %s sequence spec schema is nil", target.Key)
	}
	return spec, nil
}

func (c LookupConfig) EnabledFor(operation string) bool {
	if !c.Enabled {
		return false
	}
	for _, item := range c.Operations {
		if item == operation {
			return true
		}
	}
	return false
}

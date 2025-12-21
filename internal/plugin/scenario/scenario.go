package scenario

import (
	"context"

	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

type Scenario interface {
	Configure(pipeline string, data map[string]any) (err error)
	RegisterMetadata(md metadata.Metadata)
	RegisterGenerator(gen generator.Generator)
	Preparation(ctx context.Context) error
	Start(msgChan message.InPoint) error
	Close() error
}

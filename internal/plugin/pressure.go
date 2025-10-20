package plugin

import (
	"context"

	"github.com/xuenqlve/kyogre/internal/message"
)

type Pressure interface {
	Configure(pipeline string, data map[string]any) (err error)
	Start(ctx context.Context) error
	Execute(msg message.Message)
	Close() error
}

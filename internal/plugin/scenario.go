package plugin

import (
	"context"
)

type Scenario interface {
	Configure(pipeline string, data map[string]any) (err error)
	Start(ctx context.Context) error
	Close() error
}

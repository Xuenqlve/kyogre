package plugin

import (
	"context"

	"github.com/xuenqlve/kyogre/internal/message"
)

type Scenario interface {
	Configure(pipeline string, data map[string]any) (err error)
	Preparation(ctx context.Context) error
	Start(msgChan message.InPoint) error
	Close() error
}

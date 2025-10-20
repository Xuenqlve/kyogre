package plugin

import "github.com/xuenqlve/kyogre/internal/message"

type Generator interface {
	Configure(pipelineName string, data map[string]any) error
	MockMessage() message.Message
	Close()
}

type MockCondition struct {
}

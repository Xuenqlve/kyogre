package plugin

import "github.com/xuenqlve/kyogre/internal/message"

type Generator interface {
	Configure(pipeline string, data map[string]any) error
	RegisterMetadata(metadata Metadata)
	MockMessage(param MockParam) message.Message
	Close()
}

var (
	RandomTableSelect       = "random"
	OrderedTableSelect      = "ordered"
	SameWithLastTableSelect = "same_with_last"
	DiffFromLastTableSelect = "diff_from_last"
)

type MockParam struct {
	Operation   string
	Count       string
	TableSelect string
}

package event

type Type int

var (
	PipelineWaitStart     Type = 1000
	PipelineRunning       Type = 2000
	PipelineStop          Type = 3000
	PipelineCompleteExist Type = 3000
)

type Event struct {
	Type  Type
	Key   string
	Value map[string]any
}

type ObserverFunc func(e Event)

type Subject interface {
	Register(observer ObserverFunc)
	Upload(event Event)
}

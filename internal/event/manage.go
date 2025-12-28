package event

import "sync"

var EventAdmin EventManage

type EventManage struct {
	mu       sync.Mutex
	Observer map[Type][]ObserverFunc
}

func (e *EventManage) Init() {
	e.Observer = make(map[Type][]ObserverFunc)
}

func (e *EventManage) Register(et Type, observer ObserverFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.Observer[et]; !ok {
		e.Observer[et] = []ObserverFunc{}
	}
	e.Observer[et] = append(e.Observer[et], observer)
}

func (e *EventManage) Upload(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if observer, ok := e.Observer[event.Type]; ok {
		for _, fn := range observer {
			go fn(event)
		}
	}
}

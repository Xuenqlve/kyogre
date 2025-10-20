package message

import "time"

type Message interface {
	Type() string
	StartTime() time.Time
}

type (
	Point    chan Message
	InPoint  chan<- Message
	OutPoint chan Message
)

func (p Point) InPoint() InPoint {
	return (chan Message)(p)
}
func (p Point) OutPoint() OutPoint {
	return (chan Message)(p)
}
func (p Point) Close() {
	close(p)
}

func (p InPoint) Chan() chan<- Message {
	return p
}

func (p OutPoint) Chan() <-chan Message {
	return p
}

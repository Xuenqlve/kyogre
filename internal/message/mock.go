package message

import "time"

const MockType = "mock"

// MockMessage 是调试链路的简易消息体
type MockMessage struct {
	Value     string
	CreatedAt time.Time
}

func (m *MockMessage) Type() string {
	return MockType
}

func (m *MockMessage) StartTime() time.Time {
	return m.CreatedAt
}

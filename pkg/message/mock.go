package message

import "time"

const MockType = "mock"

// MockMessage 是调试链路的简易消息体
type MockMessage struct {
	Rows      []MockRow
	CreatedAt time.Time
}

type MockRow struct {
	Value map[string]any
}

func (m *MockMessage) Type() string {
	return MockType
}

func (m *MockMessage) StartTime() time.Time {
	return m.CreatedAt
}

package generator_context

import (
	"fmt"
)

// Context 为 mock generator 的上下文实现。
type MockContext struct {
	Value map[string]any
	*baseContext
}

const Mock = "mock"

func NewMockContext(value map[string]any, opts ...ContextOption) *MockContext {
	return &MockContext{
		Value:       value,
		baseContext: NewBaseContext(Mock, opts...),
	}
}

func (c *MockContext) Validate() error {
	if len(c.Value) == 0 {
		return fmt.Errorf("context value is nil")
	}
	return nil
}

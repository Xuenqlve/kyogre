package common

import "fmt"

// GeneratorError 生成器错误的基础类型
type GeneratorError struct {
	Code    string
	Message string
}

func (e *GeneratorError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 错误代码常量
const (
	InvalidOperationCode = "INVALID_OPERATION"
	InvalidCountCode     = "INVALID_COUNT"
	InvalidTableCode     = "INVALID_TABLE"
	GenerationFailCode   = "GENERATION_FAIL"
	ConfigErrorCode      = "CONFIG_ERROR"
)

// NewInvalidOperationError 创建无效操作错误
func NewInvalidOperationError(operation string) error {
	return &GeneratorError{
		Code:    InvalidOperationCode,
		Message: fmt.Sprintf("invalid operation: %s", operation),
	}
}

// NewInvalidCountError 创建无效行数错误
func NewInvalidCountError(count int) error {
	return &GeneratorError{
		Code:    InvalidCountCode,
		Message: fmt.Sprintf("invalid count: %d (must be >= 1)", count),
	}
}

// NewInvalidTableError 创建无效表错误
func NewInvalidTableError(tableName string) error {
	return &GeneratorError{
		Code:    InvalidTableCode,
		Message: fmt.Sprintf("invalid table: %s", tableName),
	}
}

// NewGenerationError 创建生成失败错误
func NewGenerationError(operation string, reason string) error {
	return &GeneratorError{
		Code:    GenerationFailCode,
		Message: fmt.Sprintf("failed to generate %s: %s", operation, reason),
	}
}

// NewConfigError 创建配置错误
func NewConfigError(reason string) error {
	return &GeneratorError{
		Code:    ConfigErrorCode,
		Message: fmt.Sprintf("config error: %s", reason),
	}
}

// IsGeneratorError 检查错误是否为生成器错误
func IsGeneratorError(err error) bool {
	_, ok := err.(*GeneratorError)
	return ok
}

// GetErrorCode 获取错误代码
func GetErrorCode(err error) string {
	if ge, ok := err.(*GeneratorError); ok {
		return ge.Code
	}
	return ""
}

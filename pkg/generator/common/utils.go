package common

import (
	"strconv"

	"math/rand/v2"
)

// ParseCount 解析行数字符串
// 支持格式：
//   - "": 返回 1
//   - "123": 返回 123（数字）
//   - "random": 返回 1-100 之间的随机数
//   - "random(10,100)": 返回 10-100 之间的随机数
func ParseCount(countStr string) (int, error) {
	if countStr == "" {
		return 1, nil
	}

	// 处理 "random" 格式
	if countStr == "random" {
		return rand.IntN(100) + 1, nil
	}

	// 尝试直接解析为整数
	return strconv.Atoi(countStr)
}

// TableSelectStrategy 表选择策略常量
type TableSelectStrategy string

const (
	// RandomSelect 随机选择表
	RandomSelect TableSelectStrategy = "random"
	// OrderedSelect 顺序选择表
	OrderedSelect TableSelectStrategy = "ordered"
	// SameWithLastSelect 继续使用上次的表
	SameWithLastSelect TableSelectStrategy = "same_with_last"
	// DiffFromLastSelect 选择与上次不同的表
	DiffFromLastSelect TableSelectStrategy = "diff_from_last"
)

// OperationType DML 操作类型常量
type OperationType string

const (
	Insert              OperationType = "insert"
	Update              OperationType = "update"
	Delete              OperationType = "delete"
	Replace             OperationType = "replace"
	InsertIgnore        OperationType = "insert_ignore"
	InsertOnDuplicateKey OperationType = "insert_on_duplicate_key"
	UpdateJoin          OperationType = "update_join"
)

// IsValidOperation 检查操作类型是否有效
func IsValidOperation(op string) bool {
	validOps := map[string]bool{
		string(Insert):               true,
		string(Update):               true,
		string(Delete):               true,
		string(Replace):              true,
		string(InsertIgnore):         true,
		string(InsertOnDuplicateKey): true,
		string(UpdateJoin):           true,
	}
	return validOps[op]
}

// IsWriteOperation 检查是否为写操作（INSERT/UPDATE/DELETE/REPLACE）
func IsWriteOperation(op string) bool {
	writeOps := map[string]bool{
		string(Insert):               true,
		string(Update):               true,
		string(Delete):               true,
		string(Replace):              true,
		string(InsertIgnore):         true,
		string(InsertOnDuplicateKey): true,
		string(UpdateJoin):           true,
	}
	return writeOps[op]
}

// GetOperationWriteType 获取操作的写入类型
func GetOperationWriteType(operation string) string {
	switch operation {
	case string(Insert):
		return "insert"
	case string(Update), string(UpdateJoin):
		return "update"
	case string(Delete):
		return "delete"
	case string(Replace):
		return "replace"
	case string(InsertIgnore):
		return "insert_ignore"
	case string(InsertOnDuplicateKey):
		return "insert_on_duplicate_key"
	default:
		return operation
	}
}

// NeedsOldValue 检查操作是否需要 Old 字段
func NeedsOldValue(operation string) bool {
	return operation == string(Update) || operation == string(UpdateJoin)
}

// NeedsGuideKeys 检查操作是否需要 GuideKeys（WHERE 条件）
func NeedsGuideKeys(operation string) bool {
	return operation == string(Update) || operation == string(Delete) || operation == string(UpdateJoin)
}

// BatchOperationConfig 批量操作配置
type BatchOperationConfig struct {
	Operation   string // 操作类型
	Count       int    // 生成的行数
	TableSelect string // 表选择策略
}

// ValidateConfig 验证配置的有效性
func ValidateConfig(config *BatchOperationConfig) error {
	if !IsValidOperation(config.Operation) {
		return NewInvalidOperationError(config.Operation)
	}

	if config.Count < 1 {
		return NewInvalidCountError(config.Count)
	}

	return nil
}

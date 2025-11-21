package plugin

import (
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/metadata"
	"github.com/xuenqlve/kyogre/pkg/generator/query_module"
)

// Generator 生成器接口（两阶段设计）
type Generator interface {
	Configure(pipeline string, data map[string]any) error
	RegisterMetadata(metadata metadata.Metadata)

	// 第一阶段：收集依赖条件
	// 根据配置和生成策略，确定需要哪些信息
	CollectDependencies(req *DependencyRequest) (GenerationDependency, error)

	// 第二阶段：生成消息（接收反查结果）
	// 基于依赖条件、反查结果和生成策略，生成实际数据
	MockMessage(req *MessageGenerationRequest) (message.Message, error)

	Close()
}

var (
	RandomTableSelect       = "random"
	OrderedTableSelect      = "ordered"
	SameWithLastTableSelect = "same_with_last"
	DiffFromLastTableSelect = "diff_from_last"
)

// DependencyRequest 收集依赖时的请求对象
// 使用DependencyConfig接口而不是map，确保类型安全和约束清晰
type DependencyRequest struct {
	// 使用接口而不是map，确保类型安全和约束
	// 接收方可以根据Type()来判断具体类型，然后做类型断言
	Config DependencyConfig `json:"-"`

	// 生成策略（可选）
	GenerationStrategy *GenerationStrategy `json:"generation_strategy"`
}

// MessageGenerationRequest 生成消息时的请求对象
type MessageGenerationRequest struct {
	// 第一阶段收集的依赖条件
	Dependency GenerationDependency `json:"-"`

	// 反查模块查询到的结果
	QueryResults map[string]*query_module.QueryResult `json:"-"`

	// 生成策略配置
	GenerationStrategy *GenerationStrategy `json:"generation_strategy"`
}

// 向后兼容的辅助函数
func NewDependencyRequest(config DependencyConfig, strategy *GenerationStrategy) *DependencyRequest {
	return &DependencyRequest{
		Config:             config,
		GenerationStrategy: strategy,
	}
}

func NewMessageGenerationRequest(
	dep GenerationDependency,
	queryResults map[string]*query_module.QueryResult,
	strategy *GenerationStrategy,
) *MessageGenerationRequest {
	return &MessageGenerationRequest{
		Dependency:         dep,
		QueryResults:       queryResults,
		GenerationStrategy: strategy,
	}
}

// MockParam 已弃用，保留用于向后兼容
// 新代码应使用 DependencyRequest 和 MessageGenerationRequest
type MockParam struct {
	// 核心：生成的依赖条件
	Dependency GenerationDependency `json:"-"`

	// 序列化字段（当从配置/消息反序列化时使用）
	DependencyType string         `json:"dependency_type"`  // "dml"/"transaction"/"ddl"
	DependencyData map[string]any `json:"dependency_data"`  // 具体的依赖条件数据

	// 生成策略配置
	GenerationStrategy *GenerationStrategy `json:"generation_strategy"`

	// 反查结果（在Generator第二阶段接收）
	QueryResults map[string]*query_module.QueryResult `json:"-"`

	// 【向后兼容】旧的参数字段，逐步迁移
	Operation   string
	Count       string
	TableSelect string
}

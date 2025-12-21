package generator

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

type DependencyConfig interface {
	// Validate 验证配置的有效性
	Validate() error

	// Type 返回配置类型标识（用于类型断言和日志记录）
	Type() string
}

// GenerationMode 生成模式
type GenerationMode string

// GenerationDependency 生成数据的依赖条件接口
type GenerationDependency interface {
	// 获取此生成所需的表列表
	GetSchemas() []schema_store.SchemaKey
	// 获取生成模式
	GetMode() GenerationMode

	// 验证依赖条件是否完整
	Validate() error

	// 获取依赖类型名称（用于序列化）
	DependencyType() string
}

// GenerationStrategy 生成策略配置
type GenerationStrategy struct {
	SequenceConfig *iquery.SequenceConfig `json:"sequence_config,omitempty"` // 序列化配置
	RandomConfig   *RandomConfig          `json:"random_config,omitempty"`   // 随机配置
	TemplateConfig *TemplateConfig        `json:"template_config,omitempty"` // 模板配置
	CustomConfig   map[string]any         `json:"custom_config,omitempty"`   // 自定义配置
}

// SequenceConfig 序列化生成配置

// RandomConfig 随机生成配置
type RandomConfig struct {
	Seed int64 `json:"seed,omitempty"`
}

// TemplateConfig 模板生成配置（基于预设模板）
type TemplateConfig struct {
	TemplateName string         `json:"template_name"`
	TemplateData map[string]any `json:"template_data"`
}

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

type (
	Type    string
	Factory func() Generator
)

var (
	_generatorRegistry map[Type]Factory
	_generatorMutex    sync.Mutex
)

// RegisterGenerator 注册生成器实例
func RegisterGenerator(generatorType Type, v Generator, singleton bool) {
	var gf Factory
	if singleton {
		gf = func() Generator { return v }
	} else {
		gf = func() Generator { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Generator) }
	}
	RegisterGeneratorFactory(generatorType, gf)
}

// RegisterGeneratorFactory 注册生成器工厂
func RegisterGeneratorFactory(generatorType Type, factory Factory) {
	_generatorMutex.Lock()
	defer _generatorMutex.Unlock()
	if _generatorRegistry == nil {
		_generatorRegistry = make(map[Type]Factory)
	}
	if _, ok := _generatorRegistry[generatorType]; ok {
		panic(fmt.Sprintf("generator already exists, type:%s", generatorType))
	}
	_generatorRegistry[generatorType] = factory
}

// GetGenerator 根据类型创建生成器实例
func GetGenerator(generatorType Type) (Generator, error) {
	_generatorMutex.Lock()
	defer _generatorMutex.Unlock()
	factory, ok := _generatorRegistry[generatorType]
	if !ok {
		return nil, fmt.Errorf("generator not registered type:%s", generatorType)
	}
	return factory(), nil
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
	QueryResults map[string]*iquery.LookupResult `json:"-"`

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

func NewMessageGenerationRequest(dep GenerationDependency, queryResults map[string]*iquery.LookupResult, strategy *GenerationStrategy) *MessageGenerationRequest {
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
	DependencyType string         `json:"dependency_type"` // "dml"/"transaction"/"ddl"
	DependencyData map[string]any `json:"dependency_data"` // 具体的依赖条件数据

	// 生成策略配置
	GenerationStrategy *GenerationStrategy `json:"generation_strategy"`

	// 反查结果（在Generator第二阶段接收）
	QueryResults map[string]*iquery.LookupResult `json:"-"`

	// 【向后兼容】旧的参数字段，逐步迁移
	Operation   string
	Count       string
	TableSelect string
}

// Package provider defines the interface contracts for AI model providers.
// All external AI service dependencies are abstracted behind these interfaces,
// enabling pluggable implementations (OpenAI, Anthropic, local models, etc.).
//
// 提供者包定义 AI 模型提供者的接口契约。所有外部 AI 服务依赖都被抽象在这些接口之后，
// 支持可插拔的实现（OpenAI、Anthropic、本地模型等）。
package provider

import (
	"context"

	"github.com/y-ai-accompany/server/pkg/types"
)

// ModelConfig is the inference configuration for a single request.
// ModelConfig 单次请求的推理配置。
type ModelConfig struct {
	Model       string  // 模型名称
	Temperature float64 // 温度参数（控制生成随机性）
	MaxTokens   int     // 最大生成 Token 数
	// RoutingTags maps routing tag keys to model names for multi-model routing.
	// Example: {"fast": "gpt-4o-mini", "reasoning": "gpt-4o", "embedding": "text-embedding-3"}
	// RoutingTags 路由标签键到模型名称的映射，用于多模型路由。
	// 示例: {"fast": "gpt-4o-mini", "reasoning": "gpt-4o", "embedding": "text-embedding-3"}
	RoutingTags map[string]string
}

// LLMProvider is the core language model interface.
// Implementations must be safe for concurrent use.
// LLMProvider 是核心语言模型接口。实现必须支持并发安全。
type LLMProvider interface {
	// Chat sends messages and returns the complete reply.
	// modelName can be empty to use the provider default.
	// Chat 发送消息并返回完整回复。modelName 可为空以使用提供者默认模型。
	Chat(ctx context.Context, messages []types.Message, config ModelConfig) (string, error)

	// ChatStream sends messages and returns a channel of content chunks.
	// The returned channel is closed when streaming completes or on error.
	// Callers must read until the channel is closed.
	// ChatStream 发送消息并返回内容块通道。通道在流完成或出错时关闭。
	// 调用者必须读取直到通道关闭。
	ChatStream(ctx context.Context, messages []types.Message, config ModelConfig) (<-chan string, error)
}

// EmbeddingProvider generates vector embeddings for text.
// EmbeddingProvider 生成文本的向量嵌入。
type EmbeddingProvider interface {
	// Embed returns a float32 vector for the given text.
	// Embed 返回给定文本的 float32 向量嵌入。
	Embed(ctx context.Context, text string) ([]float32, error)
}

// GuardProvider performs content safety classification.
// GuardProvider 执行内容安全分类。
type GuardProvider interface {
	// Check returns true if the text is safe, false if it should be blocked.
	// Check 返回 true 表示文本安全，false 表示应被拦截。
	Check(ctx context.Context, text string) (safe bool, err error)
}

// CompositeProvider bundles all AI provider capabilities into one interface.
// This is the recommended pattern when a single service provides all three.
// CompositeProvider 将 AI 提供者的所有能力捆绑到一个接口中。
// 当单个服务提供所有三种能力时推荐使用此模式。
type CompositeProvider interface {
	LLMProvider
	EmbeddingProvider
	GuardProvider
}

// Provider is the unified interface that extends CompositeProvider with lifecycle
// and metadata methods. This is the primary interface consumed by the inference package.
// Provider 是统一接口，扩展 CompositeProvider 并添加生命周期和元数据方法。
// 这是 inference 包消费的主要接口。
type Provider interface {
	CompositeProvider

	// Name returns the provider name (e.g. "openai", "anthropic").
	// Name 返回提供者名称（如 "openai"、"anthropic"）。
	Name() string

	// Models returns the list of model names this provider supports.
	// Models 返回此提供者支持的模型名称列表。
	Models() []string

	// Close releases any resources held by the provider.
	// Close 释放提供者持有的所有资源。
	Close() error
}

// ProviderFactoryConfig holds the configuration for creating a new Provider instance.
// ProviderFactoryConfig 保存创建新 Provider 实例的配置。
type ProviderFactoryConfig struct {
	// ProviderType identifies the provider implementation (e.g. "openai", "anthropic").
	// ProviderType 标识提供者实现类型（如 "openai"、"anthropic"）。
	ProviderType string

	// BaseURL is the API base URL for the provider.
	// BaseURL 提供者的 API 基础 URL。
	BaseURL string

	// APIKey is the authentication key for the provider.
	// APIKey 提供者的认证密钥。
	APIKey string

	// Models maps model roles to model names (e.g. {"chat": "gpt-4o", "embedding": "text-embedding-3"}).
	// Models 模型角色到模型名称的映射（如 {"chat": "gpt-4o", "embedding": "text-embedding-3"}）。
	Models map[string]string

	// Extra holds provider-specific configuration parameters.
	// Extra 保存提供者特定的配置参数。
	Extra map[string]string
}

// ProviderFactory creates Provider instances dynamically based on configuration.
// ProviderFactory 根据配置动态创建 Provider 实例。
type ProviderFactory interface {
	// Create instantiates a new Provider from the given config.
	// Create 从给定配置实例化新的 Provider。
	Create(ctx context.Context, config ProviderFactoryConfig) (Provider, error)
}

package accompany

import (
	"context"
	"log/slog"

	fwtypes "github.com/anrror/y-ai-agent-base/pkg/types"
	"github.com/anrror/y-ai-agent-base/pkg/pipeline"
	"github.com/anrror/y-ai-agent-base/pkg/agent"

	"github.com/y-ai-accompany/server/pkg/personality"
	servertypes "github.com/y-ai-accompany/server/pkg/types"
)

// PersonalityExtension 实现性格演化引擎。
// 每次对话后根据交互特征（情绪、话题新颖度、参与度）微调 OCEAN 人格参数，
// 使 Agent 性格随交互持续演化。
//
// 实现 agent.Extension + agent.MiddlewareProvider：
//   - 前置：将性格参数注入 Agent 的 identity 元数据（通过 context / Metadata）
//   - 后置：根据本次对话特征演化 OCEAN 参数
//
// 注意：当前实现使用内存中的"全局"人格轨迹。生产部署时应为每个
// (userID, agentID) 对独立维护人格状态。
type PersonalityExtension struct {
	evolver func(map[string]float64, servertypes.PersonalityFeatures, float64) map[string]float64
	// 全局人格轨迹（MVP 简化：每个 session / agent 共享一人格）
	// 生产环境应为 (userID × agentID) 维度的 OCEAN 存储
	traits map[string]float64
}

// NewPersonalityExtension 创建 PersonalityExtension 并初始化人格轨迹为默认值。
func NewPersonalityExtension() *PersonalityExtension {
	return &PersonalityExtension{
		evolver: personality.Evolve,
		traits: map[string]float64{
			"openness":          0.6,
			"conscientiousness": 0.7,
			"extraversion":      0.55,
			"agreeableness":     0.8,
			"neuroticism":       0.5,
		},
	}
}

// ID 返回扩展唯一标识。
func (e *PersonalityExtension) ID() string { return "accompany.personality" }

// Close 释放资源。
func (e *PersonalityExtension) Close() error { return nil }

// Middleware 返回管线中间件，在对话后执行人格演化。
func (e *PersonalityExtension) Middleware() pipeline.Middleware {
	return func(next pipeline.Handler) pipeline.Handler {
		return func(ctx context.Context, input *fwtypes.ChatInput, output *fwtypes.ChatOutput) error {
			// 前置：从 context 读取情感结果，构建特征
			features := extractFeatures(ctx, input)

			err := next(ctx, input, output)

			// 后置：演化人格
			if err == nil {
				e.evolver(e.traits, features, 0.97)
				slog.DebugContext(ctx, "personality: evolved",
					"traits", e.traits,
				)
			}

			return err
		}
	}
}

// Traits 返回当前人格特征副本。
func (e *PersonalityExtension) Traits() map[string]float64 {
	result := make(map[string]float64, len(e.traits))
	for k, v := range e.traits {
		result[k] = v
	}
	return result
}

// extractFeatures 从 context 和输入中提取 PersonalityFeatures。
func extractFeatures(ctx context.Context, input *fwtypes.ChatInput) servertypes.PersonalityFeatures {
	features := servertypes.PersonalityFeatures{}

	// 从 context 读取情感结果（emotion extension 写入的）
	if raw, ok := ctx.Value(CtxEmotionResult).(string); ok && raw != "" {
		// 简单启发式：检测到负面情绪 → 负 valence
		features.SentimentValence = inferValence(raw)
	}

	return features
}

// inferValence 从情感结果 JSON 中提取 valence 值。
// 当前是简单的字符串启发式。生产环境应使用 json unmarshal。
func inferValence(raw string) float64 {
	if len(raw) == 0 {
		return 0
	}
	// 简单关键词启发式：包含 sadness/anger/fear → 负向
	for _, kw := range []string{"sadness", "anger", "fear", "anxiety"} {
		if contains(raw, kw) {
			return -0.5
		}
	}
	return 0.3
}

func contains(s, substr string) bool { return len(s) >= len(substr) && containsStr(s, substr) }

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Ensure interface compliance.
var _ agent.Extension = (*PersonalityExtension)(nil)
var _ agent.MiddlewareProvider = (*PersonalityExtension)(nil)

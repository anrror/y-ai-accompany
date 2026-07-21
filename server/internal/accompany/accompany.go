// Package accompany 提供情感、记忆、性格、安全四大自定义能力扩展，
// 以 y-ai-agent-base 框架的 agent.Extension + MiddlewareProvider 接口注入 Agent 管线。
//
// 每个扩展是一个 MiddlewareProvider，其 Middleware() 被 Builder.Build()
// 自动挂载到 Agent 的 pipeline 中，实现对话前/后的自定义处理。
//
// 当前实现使用关键词模式（情感检测）、内存模式（记忆）、
// 透传模式（安全守卫），无需外部 LLM 或数据库依赖。
package accompany

import (
	"log/slog"

	"github.com/anrror/y-ai-agent-base/pkg/agent"

	"github.com/y-ai-accompany/server/pkg/emotion"
	"github.com/y-ai-accompany/server/pkg/memory"
	"github.com/y-ai-accompany/server/pkg/safety"
	servertypes "github.com/y-ai-accompany/server/pkg/types"
)

// Module 是 Accompany 能力聚合模块，持有所有扩展实例。
// 通过 Extensions() 返回 agent.Extension 切片，供 agent.Builder.WithExtensions() 注入。
type Module struct {
	safetyExt    *SafetyExtension
	emotionExt   *EmotionExtension
	memoryExt    *MemoryExtension
	personalityExt *PersonalityExtension
}

// Config 控制各扩展的启用/关闭。
type Config struct {
	Safety   bool // 安全守卫（输入/输出审核）
	Emotion  bool // 情感检测
	Memory   bool // 记忆引擎
	Personality bool // 性格演化
}

// DefaultConfig 返回默认配置：全部启用。
func DefaultConfig() Config {
	return Config{
		Safety:   true,
		Emotion:  true,
		Memory:   true,
		Personality: true,
	}
}

// New 创建 AccompanyModule。所有扩展在创建时完成初始化，
// pkg/ 包在 nil 依赖下自动使用关键词/内存/透传模式。
func New(cfg Config) *Module {
	m := &Module{}

	if cfg.Safety {
		// Safety: 透传守卫（无 GuardProvider 时始终放行）
		sCfg := servertypes.SafetyConfig{
			InputGuardEnabled:  true,
			OutputGuardEnabled: true,
			SafetyNoticeEnabled: true,
		}
		slog.Debug("accompany: safety extension created (passthrough)")
		m.safetyExt = &SafetyExtension{
			guard: safety.New(nil, sCfg),
		}
	}

	if cfg.Emotion {
		// Emotion: 关键词+Emoji 检测（provider=nil → 仅关键词）
		slog.Debug("accompany: emotion extension created (keyword-only)")
		m.emotionExt = &EmotionExtension{
			detector: emotion.New(nil),
		}
	}

	if cfg.Memory {
		// Memory: 内存模式 Engine（无 Store/Embedder/Distiller）
		m.memoryExt = &MemoryExtension{
			engine: memory.NewEngine(memory.EngineConfig{}),
		}
		slog.Debug("accompany: memory extension created (in-memory)")
	}

	if cfg.Personality {
		m.personalityExt = NewPersonalityExtension()
		slog.Debug("accompany: personality extension created")
	}

	return m
}

// Extensions 返回所有启用的 agent.Extension 实例。
// 供 agent.Builder.WithExtensions() 注入 Agent 构建流程。
func (m *Module) Extensions() []agent.Extension {
	var exts []agent.Extension
	if m.safetyExt != nil {
		exts = append(exts, m.safetyExt)
	}
	if m.emotionExt != nil {
		exts = append(exts, m.emotionExt)
	}
	if m.memoryExt != nil {
		exts = append(exts, m.memoryExt)
	}
	if m.personalityExt != nil {
		exts = append(exts, m.personalityExt)
	}
	return exts
}

// Safety 返回 SafetyExtension（用于外部直接调用，如 HTTP handler）。
func (m *Module) Safety() *SafetyExtension { return m.safetyExt }

// Emotion 返回 EmotionExtension。
func (m *Module) Emotion() *EmotionExtension { return m.emotionExt }

// Memory 返回 MemoryExtension。
func (m *Module) Memory() *MemoryExtension { return m.memoryExt }

// Personality 返回 PersonalityExtension。
func (m *Module) Personality() *PersonalityExtension { return m.personalityExt }

package accompany

import (
	"context"
	"encoding/json"
	"log/slog"

	fwtypes "github.com/anrror/y-ai-agent-base/pkg/types"
	"github.com/anrror/y-ai-agent-base/pkg/pipeline"
	"github.com/anrror/y-ai-agent-base/pkg/agent"

	"github.com/y-ai-accompany/server/pkg/emotion"
)

// contextKey 用于在 context 中传递扩展间共享数据。
type contextKey string

func (k contextKey) String() string { return "accompany." + string(k) }

const (
	// CtxEmotionResult 存储情感检测结果（JSON 序列化的 EmotionResult）。
	CtxEmotionResult contextKey = "emotion_result"
)

// EmotionExtension 实现情感检测：从用户输入中检测情绪状态。
// 仅通过关键词+Emoji 快速检测，无需 LLM 调用。
// 实现 agent.Extension + agent.MiddlewareProvider。
type EmotionExtension struct {
	detector *emotion.Detector
}

// ID 返回扩展唯一标识。
func (e *EmotionExtension) ID() string { return "accompany.emotion" }

// Close 释放资源。
func (e *EmotionExtension) Close() error { return nil }

// Middleware 返回管线中间件，在对话前执行情感检测并将结果写入 context。
func (e *EmotionExtension) Middleware() pipeline.Middleware {
	return func(next pipeline.Handler) pipeline.Handler {
		return func(ctx context.Context, input *fwtypes.ChatInput, output *fwtypes.ChatOutput) error {
			// 只在用户消息时检测
			if lastMsg := lastUserMessage(input.Messages); lastMsg != "" {
				result := e.detector.Detect(ctx, lastMsg)
				if result != nil {
					// 序列化后写入 context，供下游 middleware 或 handler 使用
					if data, err := json.Marshal(result); err == nil {
						ctx = context.WithValue(ctx, CtxEmotionResult, string(data))
					}
					slog.DebugContext(ctx, "emotion: detected",
						"emotion", result.Emotion,
						"intensity", result.Intensity,
						"valence", result.Valence,
					)
				}
			}

			return next(ctx, input, output)
		}
	}
}

// Detect 对给定文本执行情感检测（外部调用入口）。
func (e *EmotionExtension) Detect(ctx context.Context, text string) string {
	result := e.detector.Detect(ctx, text)
	if result == nil {
		return "neutral"
	}
	return result.Emotion
}

// Ensure interface compliance.
var _ agent.Extension = (*EmotionExtension)(nil)
var _ agent.MiddlewareProvider = (*EmotionExtension)(nil)

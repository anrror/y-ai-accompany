package accompany

import (
	"context"
	"log/slog"

	fwtypes "github.com/anrror/y-ai-agent-base/pkg/types"
	"github.com/anrror/y-ai-agent-base/pkg/pipeline"
	"github.com/anrror/y-ai-agent-base/pkg/agent"

	"github.com/y-ai-accompany/server/pkg/safety"
)

// SafetyExtension 实现对话安全守卫：输入审核 + 输出审核 + 安全提示注入。
// 实现 agent.Extension + agent.MiddlewareProvider。
type SafetyExtension struct {
	guard *safety.Guard
}

// ID 返回扩展唯一标识。
func (e *SafetyExtension) ID() string { return "accompany.safety" }

// Close 释放资源。
func (e *SafetyExtension) Close() error { return nil }

// Middleware 返回管线中间件，自动注入输入/输出安全审核。
func (e *SafetyExtension) Middleware() pipeline.Middleware {
	return func(next pipeline.Handler) pipeline.Handler {
		return func(ctx context.Context, input *fwtypes.ChatInput, output *fwtypes.ChatOutput) error {
			// ── 输入审核（Layer 2）──
			if lastMsg := lastUserMessage(input.Messages); lastMsg != "" {
				if err := e.guard.CheckInput(ctx, lastMsg); err != nil {
					slog.WarnContext(ctx, "safety: input blocked", "error", err)
					output.Content = "抱歉，您的消息包含不安全内容，无法处理。"
					return nil
				}
			}

			err := next(ctx, input, output)

			// ── 输出审核（Layer 4）──
			if err == nil && output.Content != "" {
				if checkErr := e.guard.CheckOutput(ctx, output.Content); checkErr != nil {
					slog.WarnContext(ctx, "safety: output blocked", "error", checkErr)
					output.Content = "抱歉，我暂时无法回复这个问题。请换个话题聊聊吧。"
				}
			}

			return err
		}
	}
}

// SafetyNotice 返回安全提示声明文本，用于注入 System Prompt。
func SafetyNotice() string { return safety.SafetyNotice() }

// lastUserMessage 从消息列表中提取最后一条用户消息的 Content。
func lastUserMessage(msgs []fwtypes.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

// ensure interface compliance
var _ agent.Extension = (*SafetyExtension)(nil)
var _ agent.MiddlewareProvider = (*SafetyExtension)(nil)

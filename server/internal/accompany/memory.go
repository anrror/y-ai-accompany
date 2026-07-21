package accompany

import (
	"context"
	"fmt"
	"log/slog"

	fwtypes "github.com/anrror/y-ai-agent-base/pkg/types"
	"github.com/anrror/y-ai-agent-base/pkg/pipeline"
	"github.com/anrror/y-ai-agent-base/pkg/agent"

	"github.com/y-ai-accompany/server/pkg/memory"
	servertypes "github.com/y-ai-accompany/server/pkg/types"
)

// MemoryExtension 实现三级记忆管线（工作记忆 → 短期 → 长期）的框架集成。
//
// 管线流程：
//   - 前置（Pre）：根据用户输入检索相关记忆，拼接到最后一条用户消息后
//   - 后置（Post）：将本次对话写入工作记忆，触发异步蒸馏
//
// 当前使用内存模式 Engine（无 pgvector / Redis 依赖）。
// 实现 agent.Extension + agent.MiddlewareProvider。
type MemoryExtension struct {
	engine *memory.Engine
}

// ID 返回扩展唯一标识。
func (e *MemoryExtension) ID() string { return "accompany.memory" }

// Close 释放资源。
func (e *MemoryExtension) Close() error { return nil }

// Middleware 返回管线中间件：前置检索 + 后置存储。
func (e *MemoryExtension) Middleware() pipeline.Middleware {
	return func(next pipeline.Handler) pipeline.Handler {
		return func(ctx context.Context, input *fwtypes.ChatInput, output *fwtypes.ChatOutput) error {
			userID := input.UserID
			agentID := input.AgentID

			slog.InfoContext(ctx, "memory: middleware pre",
				"user_id", userID, "agent_id", agentID,
				"msg_count", len(input.Messages),
			)

			// ── 前置：记忆检索 ──
			if lastMsg := lastUserMessage(input.Messages); lastMsg != "" {
				slog.InfoContext(ctx, "memory: searching",
					"user_id", userID, "agent_id", agentID,
					"query_len", len(lastMsg), "query_prefix", truncate(lastMsg, 50),
				)
				memories := e.engine.BuildContext(ctx, userID, agentID, lastMsg)
				slog.InfoContext(ctx, "memory: BuildContext result",
					"user_id", userID, "agent_id", agentID,
					"memories", memories,
				)
				if memories != "" && memories != "无相关记忆" {
					appendToLastUserMsg(input, fmt.Sprintf("\n\n[相关记忆]\n%s", memories))
					slog.InfoContext(ctx, "memory: context injected into user message",
						"user_id", userID, "agent_id", agentID,
					)
				} else {
					slog.InfoContext(ctx, "memory: no relevant memories found",
						"user_id", userID, "agent_id", agentID,
					)
				}
			} else {
				slog.WarnContext(ctx, "memory: no user message found to search against",
					"user_id", userID, "agent_id", agentID,
				)
			}

			err := next(ctx, input, output)

			// ── 后置：存储工作记忆 + 直接保存可检索记忆条目 ──
			if err == nil {
				lastUser := lastUserMessage(input.Messages)
				if lastUser != "" {
					// Stream 模式下 output.Content 在 handler 返回时是空的，
					// 只在非流式时才有关联回复内容。
					assistantReply := output.Content
					if output.IsStream {
						assistantReply = "[streaming response]"
					}
					slog.InfoContext(ctx, "memory: storing",
						"user_id", userID, "agent_id", agentID,
						"content_len", len(lastUser), "is_stream", output.IsStream,
					)

					e.engine.AddWorking(userID, agentID, "user", lastUser)
					e.engine.AddWorking(userID, agentID, "assistant", assistantReply)
					e.engine.TrimWorking(userID, agentID, 15)

					summary := truncate(lastUser, 100)
					e.engine.Save(userID, agentID, servertypes.MemoryEntry{
						Content:    summary,
						MemoryType: "short_term",
						Importance: 0.5,
					})
					go e.engine.DistillAndSave(context.Background(), userID, agentID, lastUser, assistantReply)

					slog.InfoContext(ctx, "memory: stored",
						"user_id", userID, "agent_id", agentID,
					)
				}
			}

			return err
		}
	}
}

// Engine 返回底层 memory.Engine，供外部直接调用（如 HTTP handler）。
func (e *MemoryExtension) Engine() *memory.Engine { return e.engine }

// appendToLastUserMsg 将 suffix 拼接到 input.Messages 的最后一条用户消息末尾。
// 注意：修改的是 input.Messages 切片的引用，会影响原始切片。
func appendToLastUserMsg(input *fwtypes.ChatInput, suffix string) {
	for i := len(input.Messages) - 1; i >= 0; i-- {
		if input.Messages[i].Role == "user" {
			input.Messages[i].Content += suffix
			return
		}
	}
}

// truncate 截断字符串到 maxLen 长度，保留完整结尾。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// Ensure interface compliance.
var _ agent.Extension = (*MemoryExtension)(nil)
var _ agent.MiddlewareProvider = (*MemoryExtension)(nil)

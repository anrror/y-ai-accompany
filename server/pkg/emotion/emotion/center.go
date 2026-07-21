// Package emotion — 情绪中心（Emotion Center），负责离散情绪类别检测和 VAD 三维建模。
// 回答"用户具体是什么情绪？"这一业务问题。
package emotion

import (
	"context"
	"strings"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// Center is the Emotion Center — analyzes discrete emotion categories and VAD dimensions.
// Center 是情绪中心——分析离散情绪类别（joy/sadness/anger 等）和 VAD 三维维度。
type Center struct {
	llm provider.LLMProvider
}

// New creates an EmotionCenter. llm may be nil (keyword-only mode).
// New 创建情绪中心。llm 可为 nil（仅关键词模式）。
func New(llm provider.LLMProvider) *Center {
	return &Center{llm: llm}
}

// Detect returns emotion detection result. Never returns nil.
// Detect 返回情绪检测结果。永不返回 nil。
// 三级流水线：Emoji（置信度 0.9）→ 关键词（置信度 0.75）→ LLM 兜底（置信度 0.7）→ 默认 fallback（0.3）。
func (c *Center) Detect(ctx context.Context, text string) (types.EmotionResultV2, string, float64) {
	// Tier 1: Emoji-based
	if result, ok := emojiEmotion(text); ok {
		return result, "emoji", 0.9
	}

	// Tier 2: Keyword-based
	if result, ok := keywordEmotion(text); ok {
		return result, "keyword", 0.75
	}

	// Tier 3: LLM fallback with VAD
	if c.llm != nil {
		if result, ok := c.llmEmotion(ctx, text); ok {
			return result, "llm", 0.7
		}
	}

	return types.EmotionResultV2{
		Primary: "neutral",
		VAD:     VADDefaults["neutral"],
	}, "fallback", 0.3
}

// keywordEmotion maps Chinese keywords to emotion categories.
// keywordEmotion 将中文关键词映射到情绪类别。返回匹配到的最高强度情绪及其 VAD 值。
// 如果匹配到多个不同情绪，还会设置次要情绪（Secondary）。
func keywordEmotion(text string) (types.EmotionResultV2, bool) {
	type match struct {
		primary string
		intensity float64
	}

	var matches []match
	for kw, entry := range keywords {
		if strings.Contains(text, kw) {
			matches = append(matches, match{
				primary:   entry.emotion,
				intensity: entry.intensity,
			})
		}
	}

	if len(matches) == 0 {
		return types.EmotionResultV2{}, false
	}

	// Use the highest-intensity match as primary
	best := matches[0]
	for _, m := range matches[1:] {
		if m.intensity > best.intensity {
			best = m
		}
	}

	vad := VADDefaults[best.primary]
	if vad == (types.VAD{}) {
		vad = VADDefaults["neutral"]
	}

	result := types.EmotionResultV2{
		Primary: best.primary,
		VAD:     vad,
	}

	// If multiple different emotions matched, set secondary
	if len(matches) > 1 {
		// Find the second best different from primary
		for _, m := range matches {
			if m.primary != best.primary {
				result.Secondary = m.primary
				break
			}
		}
	}

	return result, true
}

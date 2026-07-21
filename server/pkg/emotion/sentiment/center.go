// Package sentiment provides the Sentiment Center (情感中心) — business algorithm center
// for polarity detection. It evaluates the user's emotional attitude on a positive-negative
// axis, independent of specific emotion categories.
//
// 情感中心——情感极性检测。评估用户情绪态度在积极-消极轴上的位置，独立于具体情绪类别。
// 回答"用户情绪是正面还是负面？"这一业务问题。
//
// Detection pipeline (progressive confidence) / 检测流水线（渐进式置信度）:
//   1. Emoji → fast polarity from emoji presence / Emoji 快速极性判断
//   2. Keyword + modifier → lexicon-based scoring / 关键词 + 修饰词词典评分
//   3. LLM → fallback for ambiguous/fuzzy text / LLM 兜底处理模糊文本
package sentiment

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// Center is the Sentiment Center — analyzes sentiment polarity from text.
// It uses a multi-tier strategy with progressive fallback.
// Center 是情感中心——分析文本的情感极性。使用多层策略渐进式兜底。
type Center struct {
	llm provider.LLMProvider
}

// New creates a SentimentCenter. llm may be nil (keyword-only mode).
// New 创建情感中心。llm 可为 nil（仅关键词模式）。
func New(llm provider.LLMProvider) *Center {
	return &Center{llm: llm}
}

// Detect returns the sentiment analysis result. Never returns nil.
// Detect 返回情感分析结果。永不返回 nil。
// 三级流水线：Emoji（置信度 0.9）→ 关键词+修饰词（动态置信度）→ LLM 兜底（置信度 0.7）→ 默认 fallback（0.3）。
func (c *Center) Detect(ctx context.Context, text string) (types.SentimentResult, string, float64) {
	// Tier 1: Emoji-based
	if result, ok := emojiSentiment(text); ok {
		return result, "emoji", 0.9
	}

	// Tier 2: Keyword + modifier
	if result, ok := keywordSentiment(text); ok {
		return result, "keyword", confidenceFromLength(result.Intensity, text)
	}

	// Tier 3: LLM fallback
	if c.llm != nil {
		if result, ok := c.llmSentiment(ctx, text); ok {
			return result, "llm", 0.7
		}
	}

	return types.SentimentResult{Polarity: "neutral", Score: 0, Intensity: 0}, "fallback", 0.3
}

// confidenceFromLength adjusts confidence based on text length and intensity.
// confidenceFromLength 根据文本长度和强度调整置信度。文本越长、强度越高，置信度越高。
func confidenceFromLength(intensity float64, text string) float64 {
	base := 0.7
	if intensity > 0.8 {
		base = 0.85
	}
	if utf8.RuneCountInString(text) > 20 {
		base += 0.1
	}
	if base > 0.95 {
		base = 0.95
	}
	return base
}

// keywordSentiment performs sentiment analysis via lexicon matching with modifier scaling.
// Returns the result and true if any keyword matched.
// keywordSentiment 通过词典匹配和修饰词缩放执行情感分析。返回聚合后的平均分数和最大强度。
func keywordSentiment(text string) (types.SentimentResult, bool) {
	// Step 1: detect modifier intensity scale
	modifier := detectModifier(text)

	// Step 2: find matching keywords and compute score
	type match struct {
		score     float64 // raw score contribution
		intensity float64
	}
	var matches []match

	for kw, entry := range keywords {
		if strings.Contains(text, kw) {
			matches = append(matches, match{
				score:     entry.score,
				intensity: entry.intensity,
			})
		}
	}

	if len(matches) == 0 {
		return types.SentimentResult{}, false
	}

	// Aggregate: average score, max intensity
	var totalScore float64
	var maxIntensity float64
	for _, m := range matches {
		totalScore += m.score
		if m.intensity > maxIntensity {
			maxIntensity = m.intensity
		}
	}
	avgScore := totalScore / float64(len(matches))

	// Apply modifier scaling
	avgScore *= modifier.factor
	maxIntensity *= modifier.factor
	if maxIntensity > 1.0 {
		maxIntensity = 1.0
	}
	if avgScore > 1.0 {
		avgScore = 1.0
	}
	if avgScore < -1.0 {
		avgScore = -1.0
	}

	// Determine polarity
	polarity := "neutral"
	if avgScore > 0.15 {
		polarity = "positive"
	} else if avgScore < -0.15 {
		polarity = "negative"
	}

	return types.SentimentResult{
		Polarity:  polarity,
		Score:     avgScore,
		Intensity: maxIntensity,
	}, true
}

// modifierDesc holds the scaling factor for intensity modifiers.
// modifierDesc 保存强度修饰词的缩放因子。
type modifierDesc struct {
	factor float64
}

// detectModifier checks for Chinese intensity modifiers before/after the keyword.
// detectModifier 检测中文强度修饰词（如"非常"、"有点"、"一点也不"），返回缩放因子。
// 增强词放大情感强度（1.2~1.8），减弱词缩小强度（0.4~0.6），否定词归零（0.0）。
func detectModifier(text string) modifierDesc {
	type modEntry struct {
		word   string
		factor float64
	}
	modifiers := []modEntry{
		// Amplifiers
		{"非常", 1.5}, {"特别", 1.5}, {"极其", 1.8}, {"超级", 1.6},
		{"十分", 1.4}, {"太", 1.3}, {"好", 1.2}, {"真", 1.2},
		{"很", 1.3}, {"挺", 1.1}, {"蛮", 1.1},
		{"有点", 0.6}, {"有些", 0.6}, {"稍微", 0.5}, {"不太", 0.4},
		{"一点也不", 0.0}, {"根本不", 0.0},
	}
	for _, m := range modifiers {
		if strings.Contains(text, m.word) {
			return modifierDesc{factor: m.factor}
		}
	}
	return modifierDesc{factor: 1.0}
}

// Package emotion provides fast keyword-based emotion detection with optional LLM fallback.
// 情感检测包：提供基于关键词的快速情绪检测，可选 LLM 兜底。
package emotion

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// Detector detects emotion from text using a two-tier strategy:
// 1. Fast path: keyword matching (sub-millisecond, zero model cost)
// 2. Slow path: LLM classification (only when keywords miss)
//
// Deprecated: New code should use AffectiveCenter for richer results.
// Detector is kept for backward compatibility.
// Detector 从文本中检测情绪，使用两层策略：
//   1. 快速路径：关键词匹配（亚毫秒级，零模型成本）
//   2. 慢速路径：LLM 分类（仅在关键词未命中时）
// 已废弃：新代码应使用 AffectiveCenter 获取更丰富的结果。Detector 保留用于向后兼容。
type Detector struct {
	affective *AffectiveCenter
	legacy    *LegacyDetector // pure backward compat for the old behavior
}

// LegacyDetector preserves the original keyword+LLM behavior exactly.
// LegacyDetector 精确保留原始的关键词+LLM 行为，用于向后兼容。
type LegacyDetector struct {
	llm provider.LLMProvider
}

// New creates an emotion detector. llm may be nil for keyword-only mode.
// Internally uses the new AffectiveCenter for detection.
// New 创建情绪检测器。llm 可为 nil（仅关键词模式）。内部使用新的 AffectiveCenter 进行检测。
func New(llm provider.LLMProvider) *Detector {
	return &Detector{
		affective: NewAffectiveCenter(llm),
		legacy:    &LegacyDetector{llm: llm},
	}
}

// Detect returns the emotion result in legacy format. Never returns nil.
// This method uses the legacy detector path for precise backward compatibility.
// Detect 返回旧版格式的情绪结果。永不返回 nil。此方法使用旧版检测路径以确保精确的向后兼容。
func (d *Detector) Detect(ctx context.Context, text string) *types.EmotionResult {
	return d.legacy.Detect(ctx, text)
}

// DetectAffective returns the full affective result with separate sentiment and emotion.
// DetectAffective 返回完整的情感计算结果，包含独立的情感（Sentiment）和情绪（Emotion）分析。
func (d *Detector) DetectAffective(ctx context.Context, text string) *types.AffectiveResult {
	return d.affective.Analyze(ctx, text)
}

// AffectiveCenter returns the underlying AffectiveCenter for direct access.
// AffectiveCenter 返回底层的 AffectiveCenter，用于直接访问。
func (d *Detector) AffectiveCenter() *AffectiveCenter {
	return d.affective
}

// DetectFn is the functional form, usable without a Detector instance.
// DetectFn 是函数式形式，无需 Detector 实例即可使用。
func DetectFn(ctx context.Context, text string, llm provider.LLMProvider) *types.EmotionResult {
	d := &LegacyDetector{llm: llm}
	return d.Detect(ctx, text)
}

// Detect returns the legacy EmotionResult (backward compatible).
// Detect 返回旧版 EmotionResult（向后兼容）。
func (d *LegacyDetector) Detect(ctx context.Context, text string) *types.EmotionResult {
	// Use the new keyword+emoji detection for fast path
	if result := keywordDetect(text); result != nil {
		return result
	}
	if d.llm != nil {
		if result := d.llmDetect(ctx, text); result != nil {
			return result
		}
	}
	return &types.EmotionResult{Emotion: "neutral", Intensity: 0, Valence: 0}
}

// llmDetect 使用 LLM 对文本进行情绪分析，返回 JSON 格式的情绪结果。
func (d *LegacyDetector) llmDetect(ctx context.Context, text string) *types.EmotionResult {
	prompt := `Analyze emotion in this Chinese text. Return JSON only:
{"emotion":"joy|sadness|anger|fear|surprise|neutral","intensity":0.0-1.0,"valence":-1.0-1.0}
Text: ` + text

	reply, err := d.llm.Chat(ctx, []types.Message{{Role: "user", Content: prompt}}, provider.ModelConfig{
		Temperature: 0.1,
		MaxTokens:   100,
	})
	if err != nil {
		return nil
	}

	cleaned := reply
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var result types.EmotionResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil
	}
	if result.Emotion == "" {
		result.Emotion = "neutral"
	}
	return &result
}

// keywordDetect returns a result on keyword match, nil otherwise.
// This is the expanded keyword set (P3) with emoji support.
// keywordDetect 在关键词匹配时返回结果，否则返回 nil。这是扩展关键词集（P3），支持 Emoji。
func keywordDetect(text string) *types.EmotionResult {
	// Try emoji first (P3)
	if result := emojiKeywordDetect(text); result != nil {
		return result
	}

	// Then try keywords
	type entry struct {
		emotion string
		valence float64
	}
	kw := map[string]entry{
		"开心": {"joy", 0.7}, "快乐": {"joy", 0.8}, "高兴": {"joy", 0.7},
		"哈哈": {"joy", 0.8}, "嘻嘻": {"joy", 0.6}, "幸福": {"joy", 0.9},
		"难过": {"sadness", -0.7}, "伤心": {"sadness", -0.8}, "悲伤": {"sadness", -0.8},
		"哭": {"sadness", -0.9}, "哭泣": {"sadness", -0.8}, "心碎": {"sadness", -0.9},
		"生气": {"anger", -0.7}, "烦": {"anger", -0.5}, "愤怒": {"anger", -0.8},
		"害怕": {"fear", -0.8}, "担心": {"fear", -0.6}, "恐惧": {"fear", -0.9},
		"焦虑": {"anxiety", -0.7}, "紧张": {"anxiety", -0.5}, "压力": {"anxiety", -0.5},
	}
	for k, v := range kw {
		if strings.Contains(text, k) {
			return &types.EmotionResult{Emotion: v.emotion, Intensity: 0.7, Valence: v.valence}
		}
	}
	return nil
}

// emojiKeywordDetect does quick emoji→emotion for the legacy path.
// emojiKeywordDetect 为旧版路径执行快速的 Emoji→情绪映射。
func emojiKeywordDetect(text string) *types.EmotionResult {
	emoji := map[string]types.EmotionResult{
		"😊": {Emotion: "joy", Intensity: 0.6, Valence: 0.6}, "😄": {Emotion: "joy", Intensity: 0.7, Valence: 0.8}, "😢": {Emotion: "sadness", Intensity: 0.7, Valence: -0.7},
		"😭": {Emotion: "sadness", Intensity: 0.8, Valence: -0.8}, "😡": {Emotion: "anger", Intensity: 0.8, Valence: -0.8}, "❤️": {Emotion: "love", Intensity: 0.6, Valence: 0.8},
		"😍": {Emotion: "love", Intensity: 0.7, Valence: 0.8}, "😱": {Emotion: "fear", Intensity: 0.8, Valence: -0.7}, "😨": {Emotion: "fear", Intensity: 0.7, Valence: -0.6},
		"😰": {Emotion: "anxiety", Intensity: 0.7, Valence: -0.6},
	}
	for e, r := range emoji {
		if strings.Contains(text, e) {
			return &r
		}
	}
	return nil
}

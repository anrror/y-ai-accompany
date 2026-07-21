// Package emotion — LLM 情绪检测。当 Emoji 和关键词无法匹配时，使用 LLM 进行情绪分类和 VAD 提取。
package emotion

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// llmEmotion uses LLM to detect emotion categories and extract VAD dimensions.
// llmEmotion 使用 LLM 检测情绪类别并提取 VAD 三维值（愉悦度、激活度、控制感）。
// 作为三级流水线的第三级（LLM 兜底），处理复杂/隐晦的情感表达。
func (c *Center) llmEmotion(ctx context.Context, text string) (types.EmotionResultV2, bool) {
	prompt := `Analyze the emotion in this Chinese text. Return JSON only:
{
  "emotion":"joy|sadness|anger|fear|surprise|love|anxiety|gratitude|neutral",
  "valence":0.0-1.0,
  "arousal":0.0-1.0,
  "dominance":0.0-1.0
}
valence: 0=unpleasant, 0.5=neutral, 1.0=pleasant
arousal: 0=calm, 0.5=moderate, 1.0=excited
dominance: 0=submissive, 0.5=neutral, 1.0=in control
Text: ` + text

	reply, err := c.llm.Chat(ctx, []types.Message{{Role: "user", Content: prompt}}, provider.ModelConfig{
		Temperature: 0.1,
		MaxTokens:   120,
	})
	if err != nil {
		return types.EmotionResultV2{}, false
	}

	cleaned := strings.TrimSpace(reply)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var result struct {
		Emotion   string  `json:"emotion"`
		Valence   float64 `json:"valence"`
		Arousal   float64 `json:"arousal"`
		Dominance float64 `json:"dominance"`
	}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return types.EmotionResultV2{}, false
	}
	if result.Emotion == "" {
		return types.EmotionResultV2{}, false
	}

	// Validate emotion category
	validEmotions := map[string]bool{
		"joy": true, "sadness": true, "anger": true, "fear": true,
		"surprise": true, "love": true, "anxiety": true, "gratitude": true, "neutral": true,
	}
	if !validEmotions[result.Emotion] {
		result.Emotion = "neutral"
	}

	// Clamp VAD
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1.0 {
			return 1.0
		}
		return v
	}

	return types.EmotionResultV2{
		Primary: result.Emotion,
		VAD: types.VAD{
			Valence:   clamp(result.Valence),
			Arousal:   clamp(result.Arousal),
			Dominance: clamp(result.Dominance),
		},
	}, true
}

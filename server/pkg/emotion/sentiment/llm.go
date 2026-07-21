// Package sentiment — LLM 情感分析。当 Emoji 和关键词方法不足以判断时，使用 LLM 进行情感极性分析。
package sentiment

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// llmSentiment uses LLM to perform sentiment analysis when keyword/emoji methods are insufficient.
// Returns the result and true if detection succeeded.
// llmSentiment 在关键词/Emoji 方法不足以判断时，使用 LLM 执行情感极性分析。
// 作为三级流水线的第三级（LLM 兜底），处理模糊/复杂文本。
func (c *Center) llmSentiment(ctx context.Context, text string) (types.SentimentResult, bool) {
	prompt := `Analyze the sentiment polarity of this Chinese text. Return JSON only:
{"polarity":"positive|negative|neutral","score":-1.0-1.0,"intensity":0.0-1.0}
score: -1.0=extremely negative, 0=neutral, 1.0=extremely positive
intensity: 0.0=no emotion, 1.0=extremely emotional
Text: ` + text

	reply, err := c.llm.Chat(ctx, []types.Message{{Role: "user", Content: prompt}}, provider.ModelConfig{
		Temperature: 0.1,
		MaxTokens:   80,
	})
	if err != nil {
		return types.SentimentResult{}, false
	}

	cleaned := strings.TrimSpace(reply)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var result struct {
		Polarity  string  `json:"polarity"`
		Score     float64 `json:"score"`
		Intensity float64 `json:"intensity"`
	}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return types.SentimentResult{}, false
	}
	if result.Polarity == "" {
		return types.SentimentResult{}, false
	}

	// Clamp values
	if result.Score < -1.0 {
		result.Score = -1.0
	} else if result.Score > 1.0 {
		result.Score = 1.0
	}
	if result.Intensity < 0 {
		result.Intensity = 0
	} else if result.Intensity > 1.0 {
		result.Intensity = 1.0
	}

	return types.SentimentResult{
		Polarity:  result.Polarity,
		Score:     result.Score,
		Intensity: result.Intensity,
	}, true
}

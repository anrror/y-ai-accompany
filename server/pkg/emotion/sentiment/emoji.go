// Package sentiment — Emoji 情感极性映射。将 50+ 个 Emoji 字符映射到情感极性分数和强度。
// 作为三级流水线的第一级（最高置信度 0.9），毫秒级响应。
package sentiment

import (
	"strings"

	"github.com/y-ai-accompany/server/pkg/types"
)

// emojiSentiment maps emoji characters to sentiment polarity.
// Returns the result and true if any emoji was matched.
// emojiSentiment 将 Emoji 字符映射到情感极性。计算所有匹配 Emoji 的平均分数和最大强度。
func emojiSentiment(text string) (types.SentimentResult, bool) {
	type entry struct {
		score     float64
		intensity float64
	}
	// Emoji → sentiment polarity mapping (50+ entries)
	mapping := map[string]entry{
		// Positive / Happy
		"😊": {0.8, 0.7}, "😄": {0.9, 0.8}, "😁": {0.9, 0.8}, "😂": {0.6, 0.9},
		"🤣": {0.7, 0.9}, "😍": {0.9, 0.8}, "🥰": {0.9, 0.7}, "😘": {0.8, 0.6},
		"😌": {0.5, 0.3}, "😇": {0.8, 0.4}, "🙂": {0.5, 0.3}, "😋": {0.7, 0.6},
		"😎": {0.6, 0.5}, "🤗": {0.7, 0.5}, "🥳": {0.9, 0.8}, "😆": {0.8, 0.7},
		"😅": {0.3, 0.5}, "😉": {0.6, 0.4}, "😃": {0.8, 0.7}, "😀": {0.7, 0.6},
		"☺️": {0.6, 0.3}, "❤️": {0.9, 0.7}, "🧡": {0.8, 0.6}, "💛": {0.7, 0.5},
		"💚": {0.7, 0.5}, "💙": {0.7, 0.5}, "💜": {0.7, 0.5}, "💖": {0.9, 0.8},
		"💕": {0.8, 0.6}, "✨": {0.5, 0.4}, "🌟": {0.6, 0.5}, "🎉": {0.8, 0.7},
		"🎊": {0.8, 0.6}, "🌸": {0.5, 0.3}, "🌹": {0.4, 0.3}, "☀️": {0.5, 0.3},
		"👍": {0.6, 0.4}, "🙌": {0.8, 0.7}, "👏": {0.7, 0.5}, "💪": {0.5, 0.6},

		// Negative / Sad
		"😢": {-0.7, 0.8}, "😭": {-0.8, 0.9}, "😔": {-0.6, 0.6}, "😞": {-0.6, 0.5},
		"😥": {-0.5, 0.5}, "😰": {-0.6, 0.7}, "😪": {-0.4, 0.5}, "🥺": {-0.4, 0.6},
		"😩": {-0.6, 0.7}, "😫": {-0.6, 0.7}, "😵": {-0.3, 0.6}, "🤧": {-0.3, 0.4},
		"💔": {-0.8, 0.7}, "💧": {-0.3, 0.3}, "😤": {-0.5, 0.7}, "😠": {-0.7, 0.8},
		"😡": {-0.8, 0.9}, "🤬": {-0.9, 0.9}, "👿": {-0.8, 0.8}, "💢": {-0.6, 0.7},

		// Anxious / Fearful
		"😨": {-0.6, 0.8}, "😱": {-0.7, 0.9}, "😖": {-0.5, 0.6}, "😣": {-0.4, 0.5},
		"😟": {-0.4, 0.5}, "😬": {-0.3, 0.5}, "🙀": {-0.5, 0.7},

		// Neutral / Mixed
		"😐": {0.0, 0.2}, "😶": {0.0, 0.2}, "🤔": {0.0, 0.3}, "😴": {0.0, 0.2},
		"😲": {0.1, 0.7}, "😳": {0.0, 0.5}, "🤭": {0.2, 0.4}, "😏": {0.1, 0.4},
		"😮": {0.1, 0.6}, "🤐": {0.0, 0.3}, "😶‍🌫️": {0.0, 0.2},
	}

	var totalScore float64
	var maxIntensity float64
	var count int

	for emoji, e := range mapping {
		if strings.Contains(text, emoji) {
			totalScore += e.score
			if e.intensity > maxIntensity {
				maxIntensity = e.intensity
			}
			count++
		}
	}

	if count == 0 {
		return types.SentimentResult{}, false
	}

	avgScore := totalScore / float64(count)

	polarity := "neutral"
	if avgScore > 0.1 {
		polarity = "positive"
	} else if avgScore < -0.1 {
		polarity = "negative"
	}

	return types.SentimentResult{
		Polarity:  polarity,
		Score:     avgScore,
		Intensity: maxIntensity,
	}, true
}

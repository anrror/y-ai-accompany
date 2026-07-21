// Package emotion — Emoji 情绪映射。将 50+ 个 Emoji 字符映射到情绪类别和 VAD 值。
// 作为三级流水线的第一级（最高置信度 0.9），毫秒级响应。
package emotion

import (
	"strings"

	"github.com/y-ai-accompany/server/pkg/types"
)

// emojiEmotion maps emoji characters to emotion categories with VAD override.
// emojiEmotion 将 Emoji 字符映射到情绪类别及 VAD 覆盖值。
// 基于激活度（Arousal）评分选择最匹配的情绪，激活度是情绪区分度最高的维度。
func emojiEmotion(text string) (types.EmotionResultV2, bool) {
	type entry struct {
		emotion string
		vad     types.VAD
	}
	mapping := map[string]entry{
		// Joy
		"😊": {"joy", types.VAD{Valence: 0.75, Arousal: 0.60, Dominance: 0.65}}, "😄": {"joy", types.VAD{Valence: 0.85, Arousal: 0.80, Dominance: 0.70}},
		"😁": {"joy", types.VAD{Valence: 0.85, Arousal: 0.75, Dominance: 0.70}}, "😂": {"joy", types.VAD{Valence: 0.70, Arousal: 0.85, Dominance: 0.50}},
		"🤣": {"joy", types.VAD{Valence: 0.75, Arousal: 0.90, Dominance: 0.60}}, "😍": {"love", types.VAD{Valence: 0.90, Arousal: 0.80, Dominance: 0.55}},
		"🥰": {"love", types.VAD{Valence: 0.90, Arousal: 0.70, Dominance: 0.60}}, "😘": {"love", types.VAD{Valence: 0.85, Arousal: 0.65, Dominance: 0.60}},
		"😌": {"joy", types.VAD{Valence: 0.65, Arousal: 0.30, Dominance: 0.60}}, "🙂": {"joy", types.VAD{Valence: 0.55, Arousal: 0.30, Dominance: 0.55}},
		"😋": {"joy", types.VAD{Valence: 0.70, Arousal: 0.55, Dominance: 0.60}}, "😎": {"joy", types.VAD{Valence: 0.65, Arousal: 0.50, Dominance: 0.75}},
		"🤗": {"joy", types.VAD{Valence: 0.75, Arousal: 0.55, Dominance: 0.55}}, "🥳": {"joy", types.VAD{Valence: 0.90, Arousal: 0.85, Dominance: 0.70}},
		"😆": {"joy", types.VAD{Valence: 0.85, Arousal: 0.80, Dominance: 0.65}}, "😅": {"joy", types.VAD{Valence: 0.50, Arousal: 0.55, Dominance: 0.50}},
		"😉": {"joy", types.VAD{Valence: 0.60, Arousal: 0.45, Dominance: 0.60}}, "😃": {"joy", types.VAD{Valence: 0.85, Arousal: 0.80, Dominance: 0.65}},
		"😀": {"joy", types.VAD{Valence: 0.75, Arousal: 0.65, Dominance: 0.60}}, "☺️": {"joy", types.VAD{Valence: 0.65, Arousal: 0.30, Dominance: 0.55}},

		// Love
		"❤️": {"love", types.VAD{Valence: 0.90, Arousal: 0.70, Dominance: 0.55}}, "🧡": {"love", types.VAD{Valence: 0.85, Arousal: 0.65, Dominance: 0.55}},
		"💛": {"love", types.VAD{Valence: 0.80, Arousal: 0.60, Dominance: 0.55}}, "💚": {"love", types.VAD{Valence: 0.80, Arousal: 0.60, Dominance: 0.55}},
		"💙": {"love", types.VAD{Valence: 0.80, Arousal: 0.60, Dominance: 0.55}}, "💜": {"love", types.VAD{Valence: 0.80, Arousal: 0.60, Dominance: 0.55}},
		"💖": {"love", types.VAD{Valence: 0.90, Arousal: 0.75, Dominance: 0.55}}, "💕": {"love", types.VAD{Valence: 0.85, Arousal: 0.60, Dominance: 0.50}},

		// Sadness
		"😢": {"sadness", types.VAD{Valence: 0.20, Arousal: 0.45, Dominance: 0.30}}, "😭": {"sadness", types.VAD{Valence: 0.15, Arousal: 0.80, Dominance: 0.20}},
		"😔": {"sadness", types.VAD{Valence: 0.25, Arousal: 0.35, Dominance: 0.35}}, "😞": {"sadness", types.VAD{Valence: 0.25, Arousal: 0.35, Dominance: 0.30}},
		"😥": {"sadness", types.VAD{Valence: 0.30, Arousal: 0.45, Dominance: 0.30}}, "😪": {"sadness", types.VAD{Valence: 0.35, Arousal: 0.35, Dominance: 0.35}},
		"🥺": {"sadness", types.VAD{Valence: 0.30, Arousal: 0.55, Dominance: 0.25}}, "💔": {"sadness", types.VAD{Valence: 0.15, Arousal: 0.40, Dominance: 0.25}},
		"💧": {"sadness", types.VAD{Valence: 0.35, Arousal: 0.30, Dominance: 0.35}},

		// Anger
		"😠": {"anger", types.VAD{Valence: 0.20, Arousal: 0.80, Dominance: 0.70}}, "😡": {"anger", types.VAD{Valence: 0.10, Arousal: 0.90, Dominance: 0.80}},
		"🤬": {"anger", types.VAD{Valence: 0.10, Arousal: 0.90, Dominance: 0.80}}, "👿": {"anger", types.VAD{Valence: 0.15, Arousal: 0.85, Dominance: 0.80}},
		"💢": {"anger", types.VAD{Valence: 0.20, Arousal: 0.80, Dominance: 0.70}}, "😤": {"anger", types.VAD{Valence: 0.20, Arousal: 0.75, Dominance: 0.70}},

		// Fear / Anxiety
		"😨": {"fear", types.VAD{Valence: 0.20, Arousal: 0.80, Dominance: 0.25}}, "😱": {"fear", types.VAD{Valence: 0.15, Arousal: 0.90, Dominance: 0.20}},
		"😖": {"anxiety", types.VAD{Valence: 0.20, Arousal: 0.65, Dominance: 0.30}}, "😣": {"anxiety", types.VAD{Valence: 0.25, Arousal: 0.55, Dominance: 0.30}},
		"😟": {"anxiety", types.VAD{Valence: 0.25, Arousal: 0.55, Dominance: 0.30}}, "😰": {"anxiety", types.VAD{Valence: 0.25, Arousal: 0.70, Dominance: 0.25}},
		"😬": {"anxiety", types.VAD{Valence: 0.30, Arousal: 0.55, Dominance: 0.30}}, "🙀": {"fear", types.VAD{Valence: 0.20, Arousal: 0.80, Dominance: 0.20}},

		// Surprise
		"😮": {"surprise", types.VAD{Valence: 0.45, Arousal: 0.70, Dominance: 0.45}}, "😲": {"surprise", types.VAD{Valence: 0.40, Arousal: 0.80, Dominance: 0.40}},
		"😳": {"surprise", types.VAD{Valence: 0.35, Arousal: 0.65, Dominance: 0.40}}, "🤭": {"surprise", types.VAD{Valence: 0.50, Arousal: 0.55, Dominance: 0.45}},

		// Neutral
		"😐": {"neutral", types.VAD{Valence: 0.50, Arousal: 0.20, Dominance: 0.50}}, "😶": {"neutral", types.VAD{Valence: 0.50, Arousal: 0.15, Dominance: 0.50}},
		"🤔": {"neutral", types.VAD{Valence: 0.45, Arousal: 0.40, Dominance: 0.45}}, "😴": {"neutral", types.VAD{Valence: 0.45, Arousal: 0.10, Dominance: 0.45}},
		"🤐": {"neutral", types.VAD{Valence: 0.50, Arousal: 0.25, Dominance: 0.50}},
	}

	var best entry
	var maxScore float64
	found := false

	// Score based on VAD arousal (most distinctive dimension)
	for emoji, e := range mapping {
		if strings.Contains(text, emoji) {
			score := e.vad.Arousal
			if score > maxScore {
				maxScore = score
				best = e
				found = true
			}
		}
	}

	if !found {
		return types.EmotionResultV2{}, false
	}

	return types.EmotionResultV2{
		Primary: best.emotion,
		VAD:     best.vad,
	}, true
}

// Package emotion provides the Emotion Center (情绪中心) — business algorithm center
// for discrete emotion category detection and dimensional VAD (Valence-Arousal-Dominance)
// modeling. It answers "what specific emotion is the user experiencing?"
//
// 情绪中心——离散情绪类别检测和 VAD 三维建模。回答"用户具体是什么情绪？"
//
// Emotion taxonomy / 情绪分类:
//   - Primary / 主要情绪: joy, sadness, anger, fear, surprise, love, anxiety, gratitude, neutral
//   - VAD: Valence (0~1), Arousal (0~1), Dominance (0~1) per Russell's circumplex model
//
// Detection pipeline (progressive confidence) / 检测流水线（渐进式置信度）:
//   1. Emoji → fast emotion mapping / Emoji 快速情绪映射
//   2. Keyword → 60+ Chinese keywords mapped to emotion categories / 60+ 中文情绪关键词
//   3. LLM → full VAD extraction for complex expressions / LLM 完整 VAD 提取
package emotion

import (
	"github.com/y-ai-accompany/server/pkg/types"
)

// VADDefaults maps each primary emotion category to its canonical VAD position
// (Valence 0~1, Arousal 0~1, Dominance 0~1).
// Based on Russell's circumplex model and ANEW norms adapted for Chinese.
// VADDefaults 将每个主要情绪类别映射到其标准 VAD 位置（愉悦度 0~1，激活度 0~1，控制感 0~1）。
// 基于 Russell 环状模型和适用于中文的 ANEW 常模。
var VADDefaults = map[string]types.VAD{
	"joy":      {Valence: 0.85, Arousal: 0.75, Dominance: 0.70},
	"sadness":  {Valence: 0.20, Arousal: 0.35, Dominance: 0.30},
	"anger":    {Valence: 0.15, Arousal: 0.85, Dominance: 0.75},
	"fear":     {Valence: 0.20, Arousal: 0.80, Dominance: 0.25},
	"surprise": {Valence: 0.50, Arousal: 0.80, Dominance: 0.45},
	"love":     {Valence: 0.90, Arousal: 0.65, Dominance: 0.55},
	"anxiety":  {Valence: 0.25, Arousal: 0.75, Dominance: 0.30},
	"gratitude": {Valence: 0.85, Arousal: 0.50, Dominance: 0.55},
	"neutral":  {Valence: 0.50, Arousal: 0.30, Dominance: 0.50},
}

// VADMapChinese provides Chinese aliases for emotion categories.
// EmotionChinese 提供情绪类别的中文别名映射。
var EmotionChinese = map[string]string{
	"joy":      "喜悦",
	"sadness":  "悲伤",
	"anger":    "愤怒",
	"fear":     "恐惧",
	"surprise": "惊讶",
	"love":     "爱意",
	"anxiety":  "焦虑",
	"gratitude": "感激",
	"neutral":  "中性",
}

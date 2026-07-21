// Package emotion provides the Affective Center — the unified orchestrator for
// all affective computing (情感计算) in the platform.
//
// 情感计算引擎 — 双中心并行架构，将"情感"(Sentiment)和"情绪"(Emotion)作为两个独立的
// 业务算法中心解耦设计，每个中心拥有独立的检测流水线和扩展策略。
//
// Architecture / 架构:
//
//	AffectiveCenter
//	  ├── SentimentCenter (情感中心) — polarity: positive/negative/neutral (情感极性)
//	  └── EmotionCenter   (情绪中心) — discrete category + VAD (离散情绪类别 + 三维VAD)
//
// Detection pipeline (per center) / 检测流水线（每个中心独立运行）:
//	Emoji → Keyword+Modifier → LLM fallback
//
// The centers are independent business algorithm modules, each with its own
// detection strategies, making it easy to add new strategies (e.g. ML model,
// facial expression API) without affecting the other center.
// 两个中心是独立的业务算法模块，各自拥有独立的检测策略，新增策略（如ML模型、表情API）
// 不会影响另一个中心。
package emotion

import (
	"context"
	"sync"

	"github.com/y-ai-accompany/server/pkg/emotion/emotion"
	"github.com/y-ai-accompany/server/pkg/emotion/sentiment"
	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// AffectiveCenter is the unified orchestrator that runs both sentiment and emotion
// analysis in parallel and produces a combined AffectiveResult.
// AffectiveCenter 是统一编排器，并行运行情感分析和情绪分析，生成合并的 AffectiveResult。
type AffectiveCenter struct {
	sentiment *sentiment.Center
	emotion   *emotion.Center
}

// NewAffectiveCenter creates the full affective computing stack.
// llm may be nil (keyword+emoji-only mode).
// NewAffectiveCenter 创建完整的情感计算栈。llm 可为 nil（仅关键词+Emoji 模式）。
func NewAffectiveCenter(llm provider.LLMProvider) *AffectiveCenter {
	return &AffectiveCenter{
		sentiment: sentiment.New(llm),
		emotion:   emotion.New(llm),
	}
}

// Analyze performs both sentiment and emotion analysis on the given text.
// Both centers run their own detection pipelines from emoji→keyword→LLM fallback.
// Returns a unified AffectiveResult. Never returns nil.
// Analyze 对给定文本同时执行情感分析和情绪分析。两个中心各自运行 emoji→keyword→LLM 兜底
// 检测流水线。返回统一的 AffectiveResult，永不返回 nil。
func (ac *AffectiveCenter) Analyze(ctx context.Context, text string) *types.AffectiveResult {
	var (
		sentimentResult types.SentimentResult
		sentimentSrc    string
		sentimentConf   float64
		emotionResult   types.EmotionResultV2
		emotionSrc      string
		emotionConf     float64
	)

	// Run both centers concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		sentimentResult, sentimentSrc, sentimentConf = ac.sentiment.Detect(ctx, text)
	}()

	go func() {
		defer wg.Done()
		emotionResult, emotionSrc, emotionConf = ac.emotion.Detect(ctx, text)
	}()

	wg.Wait()

	// Pick the source with higher confidence as the overall source
	source := sentimentSrc
	confidence := sentimentConf
	if emotionConf > sentimentConf {
		source = emotionSrc
		confidence = emotionConf
	}

	return &types.AffectiveResult{
		Sentiment:  sentimentResult,
		Emotion:    emotionResult,
		Source:     source,
		Confidence: confidence,
	}
}

// SentimentCenter returns the underlying sentiment center for direct access.
// SentimentCenter 返回底层的情感中心，用于直接访问。
func (ac *AffectiveCenter) SentimentCenter() *sentiment.Center {
	return ac.sentiment
}

// EmotionCenter returns the underlying emotion center for direct access.
// EmotionCenter 返回底层的情绪中心，用于直接访问。
func (ac *AffectiveCenter) EmotionCenter() *emotion.Center {
	return ac.emotion
}

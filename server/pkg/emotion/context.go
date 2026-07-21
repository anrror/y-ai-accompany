// Package emotion — 对话情绪上下文追踪。
// ConversationContext 在对话轮次的滑动窗口中追踪情绪状态（P2），
// 维护每轮的情感快照并计算趋势。
package emotion

import (
	"math"
	"sync"

	"github.com/y-ai-accompany/server/pkg/types"
)

// ConversationContext tracks the emotional state across a sliding window of conversation turns (P2).
// It maintains per-turn affective snapshots and computes trends.
// ConversationContext 在对话轮次的滑动窗口中追踪情绪状态（P2）。
// 它维护每轮的情感快照并计算趋势（上升/下降/稳定）。
type ConversationContext struct {
	mu         sync.RWMutex
	turns      []types.AffectiveTurn
	windowSize int
}

// NewConversationContext creates a sliding window emotion tracker.
// windowSize sets the number of recent turns to keep (default 3 if 0).
// NewConversationContext 创建滑动窗口情绪追踪器。windowSize 设置保留的最近轮次数（0 时默认为 3）。
func NewConversationContext(windowSize int) *ConversationContext {
	if windowSize <= 0 {
		windowSize = 3
	}
	return &ConversationContext{
		turns:      make([]types.AffectiveTurn, 0, windowSize+1),
		windowSize: windowSize,
	}
}

// Push adds a new turn to the sliding window.
// Push 向滑动窗口添加新的对话轮次。
func (cc *ConversationContext) Push(role, emotion string, vad types.VAD) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.turns = append(cc.turns, types.AffectiveTurn{
		Role:      role,
		Emotion:   emotion,
		Valence:   vad.Valence,
		Arousal:   vad.Arousal,
		Dominance: vad.Dominance,
	})

	// Trim to window size
	if len(cc.turns) > cc.windowSize {
		cc.turns = cc.turns[len(cc.turns)-cc.windowSize:]
	}
}

// Snapshot returns the current conversation-level affect context.
// Snapshot 返回当前对话级别的情感上下文（主导情绪、趋势、窗口快照）。
func (cc *ConversationContext) Snapshot() types.AffectContext {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if len(cc.turns) == 0 {
		return types.AffectContext{
			Turns:    []types.AffectiveTurn{},
			Window:   cc.windowSize,
			Trend:    "stable",
			Dominant: "neutral",
		}
	}

	// Compute dominant emotion
	emotionCount := make(map[string]int)
	for _, t := range cc.turns {
		emotionCount[t.Emotion]++
	}
	dominant := "neutral"
	maxCount := 0
	for em, count := range emotionCount {
		if count > maxCount {
			maxCount = count
			dominant = em
		}
	}

	// Compute trend by comparing recent valence trajectory
	trend := cc.computeTrend()

	// Return a copy of the turns slice
	turnsCopy := make([]types.AffectiveTurn, len(cc.turns))
	copy(turnsCopy, cc.turns)

	return types.AffectContext{
		Turns:    turnsCopy,
		Window:   cc.windowSize,
		Trend:    trend,
		Dominant: dominant,
	}
}

// computeTrend determines the emotional trajectory across the window.
// "escalating" = valence increasing (getting more positive)
// "de-escalating" = valence decreasing (getting more negative)
// "stable" = no significant change
// computeTrend 确定窗口内的情绪轨迹趋势。
// "escalating" = 愉悦度上升（变得更积极）
// "de-escalating" = 愉悦度下降（变得更消极）
// "stable" = 无明显变化
func (cc *ConversationContext) computeTrend() string {
	if len(cc.turns) < 2 {
		return "stable"
	}

	// Use linear regression slope of valence values
	n := float64(len(cc.turns))
	var sumX, sumY, sumXY, sumX2 float64
	for i, t := range cc.turns {
		x := float64(i)
		y := t.Valence
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return "stable"
	}

	slope := (n*sumXY - sumX*sumY) / denom

	if slope > 0.15 {
		return "escalating"
	} else if slope < -0.15 {
		return "de-escalating"
	}
	return "stable"
}

// Clear resets the conversation context.
// Clear 重置对话上下文，清空所有追踪的轮次。
func (cc *ConversationContext) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.turns = make([]types.AffectiveTurn, 0, cc.windowSize+1)
}

// Len returns the number of turns tracked.
// Len 返回当前追踪的对话轮次数。
func (cc *ConversationContext) Len() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return len(cc.turns)
}

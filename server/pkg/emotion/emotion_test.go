package emotion

import (
	"testing"

	"github.com/y-ai-accompany/server/pkg/types"
)

// TestConversationContext tests the sliding window emotion tracking (P2).
func TestConversationContext(t *testing.T) {
	cc := NewConversationContext(3)

	// Empty state
	snap := cc.Snapshot()
	if snap.Trend != "stable" {
		t.Errorf("initial trend = %s, want stable", snap.Trend)
	}

	// Add turns with escalating valence
	cc.Push("user", "sadness", types.VAD{Valence: 0.2, Arousal: 0.5, Dominance: 0.3})
	cc.Push("user", "neutral", types.VAD{Valence: 0.5, Arousal: 0.3, Dominance: 0.5})
	cc.Push("user", "joy", types.VAD{Valence: 0.8, Arousal: 0.7, Dominance: 0.6})

	snap = cc.Snapshot()
	if snap.Trend != "escalating" {
		t.Errorf("trend = %s, want escalating", snap.Trend)
	}
	if snap.Dominant != "neutral" && snap.Dominant != "sadness" && snap.Dominant != "joy" {
		t.Errorf("unexpected dominant = %s", snap.Dominant)
	}

	// Test de-escalating
	cc.Clear()
	cc.Push("user", "joy", types.VAD{Valence: 0.9, Arousal: 0.7, Dominance: 0.6})
	cc.Push("user", "neutral", types.VAD{Valence: 0.5, Arousal: 0.3, Dominance: 0.5})
	cc.Push("user", "sadness", types.VAD{Valence: 0.2, Arousal: 0.5, Dominance: 0.3})

	snap = cc.Snapshot()
	if snap.Trend != "de-escalating" {
		t.Errorf("trend = %s, want de-escalating", snap.Trend)
	}

	// Test stable
	cc.Clear()
	cc.Push("user", "neutral", types.VAD{Valence: 0.5, Arousal: 0.3, Dominance: 0.5})
	cc.Push("user", "neutral", types.VAD{Valence: 0.5, Arousal: 0.3, Dominance: 0.5})

	snap = cc.Snapshot()
	if snap.Trend != "stable" {
		t.Errorf("trend = %s, want stable", snap.Trend)
	}
}

// TestAffectiveCenter tests the unified orchestrator with keyword+emoji mode (no LLM).
// This tests end-to-end: AffectiveCenter → SentimentCenter + EmotionCenter.
func TestAffectiveCenter(t *testing.T) {
	ac := NewAffectiveCenter(nil) // no LLM, keyword+emoji only

	tests := []struct {
		name        string
		text        string
		wantEmotion string
		wantSource  string
	}{
		{name: "happy keyword", text: "今天很开心", wantEmotion: "joy", wantSource: "keyword"},
		{name: "sad keyword", text: "我好难过", wantEmotion: "sadness", wantSource: "keyword"},
		{name: "angry keyword", text: "真的很生气", wantEmotion: "anger", wantSource: "keyword"},
		{name: "emoji smile", text: "你好😊", wantEmotion: "joy", wantSource: "emoji"},
		{name: "emoji sad", text: "😢", wantEmotion: "sadness", wantSource: "emoji"},
		{name: "love emoji", text: "❤️", wantEmotion: "love", wantSource: "emoji"},
		{name: "neutral fallback", text: "今天星期三", wantEmotion: "neutral", wantSource: "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ac.Analyze(nil, tt.text)
			if result == nil {
				t.Fatal("got nil result")
			}
			if result.Emotion.Primary != tt.wantEmotion {
				t.Errorf("emotion = %s, want %s", result.Emotion.Primary, tt.wantEmotion)
			}
			if result.Source != tt.wantSource && tt.wantSource != "" {
				t.Errorf("source = %s, want %s", result.Source, tt.wantSource)
			}
			if result.Confidence <= 0 {
				t.Error("confidence should be > 0")
			}
		})
	}
}

// TestAffectiveSentiment tests the sentiment side of the AffectiveCenter.
func TestAffectiveSentiment(t *testing.T) {
	ac := NewAffectiveCenter(nil)

	tests := []struct {
		name         string
		text         string
		wantPolarity string
	}{
		{name: "positive joy", text: "我今天很开心", wantPolarity: "positive"},
		{name: "positive happiness", text: "好幸福啊", wantPolarity: "positive"},
		{name: "positive thank", text: "非常感谢你", wantPolarity: "positive"},
		{name: "negative sad", text: "我好难过", wantPolarity: "negative"},
		{name: "negative cry", text: "忍不住哭了", wantPolarity: "negative"},
		{name: "negative anger", text: "很生气", wantPolarity: "negative"},
		{name: "negative fear", text: "有点害怕", wantPolarity: "negative"},
		{name: "negative anxiety", text: "我好焦虑", wantPolarity: "negative"},
		{name: "neutral", text: "今天星期三", wantPolarity: "neutral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ac.Analyze(nil, tt.text)
			if result == nil {
				t.Fatal("got nil result")
			}
			if result.Sentiment.Polarity != tt.wantPolarity {
				t.Errorf("polarity = %s, want %s (score=%.2f)", result.Sentiment.Polarity, tt.wantPolarity, result.Sentiment.Score)
			}
		})
	}
}

// TestLegacyDetector tests backward compatibility with the Detector interface.
func TestLegacyDetector(t *testing.T) {
	d := New(nil) // keyword+emoji only, no LLM

	tests := []struct {
		name        string
		text        string
		wantEmotion string
	}{
		{name: "joy", text: "开心", wantEmotion: "joy"},
		{name: "sadness", text: "难过", wantEmotion: "sadness"},
		{name: "anger", text: "生气", wantEmotion: "anger"},
		{name: "fear", text: "害怕", wantEmotion: "fear"},
		{name: "anxiety", text: "焦虑", wantEmotion: "anxiety"},
		{name: "cry emoji", text: "😭", wantEmotion: "sadness"},
		{name: "smile emoji", text: "😊", wantEmotion: "joy"},
		{name: "heart emoji", text: "❤️", wantEmotion: "love"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.Detect(nil, tt.text)
			if result == nil {
				t.Fatal("got nil result")
			}
			if result.Emotion != tt.wantEmotion {
				t.Errorf("emotion = %s, want %s", result.Emotion, tt.wantEmotion)
			}
		})
	}
}

// TestDetectFn verifies the standalone detect function works with the legacy path.
func TestDetectFn(t *testing.T) {
	result := DetectFn(nil, "开心", nil)
	if result == nil {
		t.Fatal("DetectFn returned nil")
	}
	if result.Emotion != "joy" {
		t.Errorf("emotion = %s, want joy", result.Emotion)
	}
}

// TestAffectiveSubCenters tests direct access to sub-centers.
func TestAffectiveSubCenters(t *testing.T) {
	ac := NewAffectiveCenter(nil)

	sc := ac.SentimentCenter()
	if sc == nil {
		t.Error("SentimentCenter() returned nil")
	}

	ec := ac.EmotionCenter()
	if ec == nil {
		t.Error("EmotionCenter() returned nil")
	}
}

// TestToEmotionResult validates the conversion from AffectiveResult to legacy EmotionResult.
func TestToEmotionResult(t *testing.T) {
	ar := &types.AffectiveResult{
		Sentiment: types.SentimentResult{Polarity: "positive", Score: 0.7, Intensity: 0.6},
		Emotion: types.EmotionResultV2{
			Primary: "joy",
			VAD:     types.VAD{Valence: 0.85, Arousal: 0.75, Dominance: 0.7},
		},
		Source:     "keyword",
		Confidence: 0.85,
	}

	legacy := ar.ToEmotionResult()
	if legacy.Emotion != "joy" {
		t.Errorf("legacy Emotion = %s, want joy", legacy.Emotion)
	}
	if legacy.Intensity != 0.75 {
		t.Errorf("legacy Intensity = %.2f, want 0.75", legacy.Intensity)
	}
	// VAD Valence 0.85 → old Valence = 0.85*2-1 = 0.7
	if legacy.Valence < 0.69 || legacy.Valence > 0.71 {
		t.Errorf("legacy Valence = %.2f, want ~0.70", legacy.Valence)
	}
}

// TestDetectAffective tests the new method on the backward-compat Detector.
func TestDetectAffective(t *testing.T) {
	d := New(nil)
	result := d.DetectAffective(nil, "开心")
	if result == nil {
		t.Fatal("DetectAffective returned nil")
	}
	if result.Emotion.Primary != "joy" {
		t.Errorf("emotion = %s, want joy", result.Emotion.Primary)
	}
	if result.Sentiment.Polarity != "positive" {
		t.Errorf("polarity = %s, want positive", result.Sentiment.Polarity)
	}
}

// TestContextClear tests that clearing the context works (P2).
func TestContextClear(t *testing.T) {
	cc := NewConversationContext(3)
	cc.Push("user", "joy", types.VAD{Valence: 0.8, Arousal: 0.7, Dominance: 0.6})

	if cc.Len() != 1 {
		t.Errorf("Len() = %d, want 1", cc.Len())
	}

	cc.Clear()
	if cc.Len() != 0 {
		t.Errorf("after Clear, Len() = %d, want 0", cc.Len())
	}
}

// TestContextWindowSize tests that the context respects window size (P2).
func TestContextWindowSize(t *testing.T) {
	cc := NewConversationContext(3)

	// Push 5 turns — only last 3 should remain
	cc.Push("user", "joy", types.VAD{Valence: 0.9, Arousal: 0.7, Dominance: 0.6})
	cc.Push("user", "joy", types.VAD{Valence: 0.8, Arousal: 0.6, Dominance: 0.6})
	cc.Push("user", "neutral", types.VAD{Valence: 0.5, Arousal: 0.3, Dominance: 0.5})
	cc.Push("user", "sadness", types.VAD{Valence: 0.2, Arousal: 0.5, Dominance: 0.3})
	cc.Push("user", "sadness", types.VAD{Valence: 0.1, Arousal: 0.6, Dominance: 0.2})

	snap := cc.Snapshot()
	if len(snap.Turns) > 3 {
		t.Errorf("turns = %d, want ≤ 3", len(snap.Turns))
	}
}

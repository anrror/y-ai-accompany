// Package types defines the core data types shared across the framework.
// All types are pure data carriers — no behavior, no dependencies beyond encoding/json.
package types

// Message represents a single chat message in OpenAI-compatible format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OCEAN represents the Big-5 personality model.
// All values range [0.0, 1.0].
type OCEAN struct {
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
}

// ToMap converts OCEAN to a string-keyed map for generic personality operations.
func (p OCEAN) ToMap() map[string]float64 {
	return map[string]float64{
		"openness":          p.Openness,
		"conscientiousness": p.Conscientiousness,
		"extraversion":      p.Extraversion,
		"agreeableness":     p.Agreeableness,
		"neuroticism":       p.Neuroticism,
	}
}

// FromMap fills an OCEAN from a string-keyed map. Missing keys default to 0.5.
func (p *OCEAN) FromMap(m map[string]float64) {
	get := func(k string) float64 {
		if v, ok := m[k]; ok {
			return v
		}
		return 0.5
	}
	p.Openness = get("openness")
	p.Conscientiousness = get("conscientiousness")
	p.Extraversion = get("extraversion")
	p.Agreeableness = get("agreeableness")
	p.Neuroticism = get("neuroticism")
}

// Identity defines the persona and presentation of an agent.
type Identity struct {
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	Persona      string `json:"persona"`
	SpeakingStyle string `json:"speaking_style"`
	Greeting     string `json:"greeting"`
}

// LLMConfig controls LLM inference parameters.
type LLMConfig struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// MemoryConfig controls memory subsystem behavior per agent.
type MemoryConfig struct {
	Enabled            bool `json:"enabled"`
	RetrievalTopK      int  `json:"retrieval_top_k"`
	WorkingMemoryTurns int  `json:"working_memory_turns"`
}

// MemoryEntry is a single unit of stored memory (short-term or long-term).
type MemoryEntry struct {
	ID           string   `json:"id,omitempty"`
	UserID       string   `json:"user_id"`
	AgentID      string   `json:"agent_id"`
	Content      string   `json:"content"`
	MemoryType   string   `json:"memory_type"`   // "short_term" or "long_term"
	EmotionLabel string   `json:"emotion_label,omitempty"`
	Topics       []string `json:"topics,omitempty"`
	Importance   float64  `json:"importance,omitempty"`
	IsPermanent  bool     `json:"is_permanent"`
	Timestamp    int64    `json:"timestamp"`
}

// SessionState holds transient session data for a user-agent pair.
type SessionState struct {
	UserID      string `json:"user_id"`
	AgentID     string `json:"agent_id"`
	State       string `json:"state"`
	LastMessage string `json:"last_message,omitempty"`
	UpdatedAt   int64  `json:"updated_at"`
}

// PersonalityFeatures are the signals that drive personality evolution.
type PersonalityFeatures struct {
	SentimentValence float64 `json:"sentiment_valence"` // -1.0 to 1.0
	TopicNovelty     float64 `json:"topic_novelty"`     // 0.0 to 1.0
	UserEngagement   float64 `json:"user_engagement"`   // 0.0 to 1.0
}

// EmotionResult is the output of emotion detection (legacy format for backward compat).
// New code should prefer the richer AffectiveResult types from the emotion package.
type EmotionResult struct {
	Emotion   string  `json:"emotion"`
	Intensity float64 `json:"intensity"`
	Valence   float64 `json:"valence"`
}

// VAD represents the three-dimensional emotional state model:
// Valence (pleasure/displeasure), Arousal (activation/deactivation), Dominance (control/submission).
// All dimensions range [0.0, 1.0].
type VAD struct {
	Valence   float64 `json:"valence"`   // pleasure: 0.0 (unpleasant) ~ 1.0 (pleasant)
	Arousal   float64 `json:"arousal"`   // activation: 0.0 (calm) ~ 1.0 (excited)
	Dominance float64 `json:"dominance"` // control: 0.0 (submissive) ~ 1.0 (in control)
}

// SentimentResult represents the sentiment polarity analysis (情感 - evaluative dimension).
type SentimentResult struct {
	Polarity  string  `json:"polarity"`  // "positive" / "negative" / "neutral"
	Score     float64 `json:"score"`     // -1.0 (very negative) ~ 1.0 (very positive)
	Intensity float64 `json:"intensity"` // 0.0 ~ 1.0 (emotional intensity regardless of polarity)
}

// EmotionResultV2 represents the full emotion detection result (情绪 - categorical + dimensional).
type EmotionResultV2 struct {
	Primary   string `json:"primary"`             // primary emotion category: "joy"/"sadness"/"anger"/"fear"/"surprise"/"love"/"anxiety"/"gratitude"/"neutral"
	Secondary string `json:"secondary,omitempty"` // secondary emotion if mixed
	VAD       VAD    `json:"vad"`                 // Valence-Arousal-Dominance dimensions
}

// AffectiveResult is the unified output combining both sentiment and emotion analysis.
type AffectiveResult struct {
	Sentiment  SentimentResult   `json:"sentiment"`
	Emotion    EmotionResultV2   `json:"emotion"`
	Source     string            `json:"source"`     // detection source: "emoji" / "keyword" / "llm"
	Confidence float64           `json:"confidence"` // 0.0 ~ 1.0
}

// ToEmotionResult converts AffectiveResult to the legacy EmotionResult format.
func (r *AffectiveResult) ToEmotionResult() EmotionResult {
	// Old Valence (-1~1) maps from VAD Valence (0~1): 0→-1, 0.5→0, 1→1
	oldValence := r.Emotion.VAD.Valence*2 - 1
	return EmotionResult{
		Emotion:   r.Emotion.Primary,
		Intensity: r.Emotion.VAD.Arousal,
		Valence:   oldValence,
	}
}

// AffectContext maintains the conversation-level emotional context over a sliding window.
type AffectContext struct {
	Turns    []AffectiveTurn `json:"turns"`
	Window   int             `json:"window"`
	Trend    string          `json:"trend"`     // "escalating" / "de-escalating" / "stable"
	Dominant string          `json:"dominant"`  // dominant emotion across the window
}

// AffectiveTurn is a single turn's affective state in the conversation history.
type AffectiveTurn struct {
	Role      string  `json:"role"`       // "user" or "assistant"
	Emotion   string  `json:"emotion"`    // primary emotion category
	Valence   float64 `json:"valence"`    // VAD valence (0~1)
	Arousal   float64 `json:"arousal"`    // VAD arousal (0~1)
	Dominance float64 `json:"dominance"`  // VAD dominance (0~1)
}

// MemoryExtract is the LLM-structured output of memory distillation.
type MemoryExtract struct {
	Summary     string   `json:"summary"`
	Topics      []string `json:"topics"`
	Emotion     string   `json:"emotion"`
	Facts       []string `json:"facts"`
	Preferences []string `json:"preferences"`
}

// ChatInput is the unified input structure for the conversation pipeline.
type ChatInput struct {
	UserID  string `json:"user_id"`
	AgentID string `json:"agent_id"`
	Message string `json:"message"`
}

// ChatOutput is the unified output structure from the conversation pipeline.
type ChatOutput struct {
	Reply      string         `json:"reply"`
	// Stream is a channel for real-time token delivery to SSE consumers.
	// The pipeline writes individual tokens as they arrive from the LLM;
	// the HTTP handler in internal/ reads them to produce SSE data frames.
	// Must be created with make(chan string, N) before pipeline execution.
	// Stream 是流式 SSE 消费者的实时 token 传送通道。
	// 管线在 LLM token 到达时逐个写入；internal/ 中的 HTTP handler 读取并产生 SSE data 帧。
	// 必须在管线执行前通过 make(chan string, N) 创建。
	Stream     chan string  `json:"-"`
	Emotion    *EmotionResult `json:"emotion,omitempty"`
	IsStream   bool           `json:"-"`
}

// ChatCompletionRequest is the OpenAI-compatible chat completion request body.
type ChatCompletionRequest struct {
	UserID   string    `json:"user_id"`
	AgentID  string    `json:"agent_id"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// SafetyConfig controls per-agent safety layer enable/disable.
// Each layer can be independently toggled per agent YAML configuration.
type SafetyConfig struct {
	// InputGuardEnabled 控制 Layer 2 输入审核开关（默认开启）。
	// 开启后，用户消息在进入管线前会经过 Guard 模型审核。
	InputGuardEnabled bool `json:"input_guard_enabled" yaml:"input_guard_enabled"`
	// OutputGuardEnabled 控制 Layer 4 输出审核开关（默认开启）。
	// 开启后，Agent 回复在返回前会经过 Guard 模型审核。
	OutputGuardEnabled bool `json:"output_guard_enabled" yaml:"output_guard_enabled"`
	// SafetyNoticeEnabled 控制 Layer 3 安全提示注入开关（默认开启）。
	// 开启后，System Prompt 中会注入 AI 安全声明（不自称为人、不鼓励自伤等）。
	SafetyNoticeEnabled bool `json:"safety_notice_enabled" yaml:"safety_notice_enabled"`
}

// DefaultSafetyConfig 返回所有安全层全开的默认配置。
func DefaultSafetyConfig() SafetyConfig {
	return SafetyConfig{
		InputGuardEnabled:    true,
		OutputGuardEnabled:   true,
		SafetyNoticeEnabled:  true,
	}
}

// GuardResult represents the outcome of a safety check.
type GuardResult struct {
	Safe   bool   `json:"safe"`
	Reason string `json:"reason,omitempty"`
}

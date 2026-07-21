// Package memory provides the Memory as a Service (MaaS) abstraction.
// It defines the store interface and implements the three-tier memory pipeline:
// working memory (short sliding window) → short-term (LLM summaries) → long-term (solidified facts).
package memory

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// Store is the persistence interface for memory entries.
// Implementations include pgvector, in-memory, and other vector stores.
type Store interface {
	// Save persists a memory entry. The caller must set UserID, AgentID, and Timestamp.
	Save(ctx context.Context, entry *types.MemoryEntry) error

	// Search retrieves memories relevant to the query, ordered by relevance.
	// topK limits the result count.
	Search(ctx context.Context, userID, agentID, query string, topK int) ([]types.MemoryEntry, error)

	// DeleteAll removes all memories for a user-agent pair.
	DeleteAll(ctx context.Context, userID, agentID string) error

	// Stats returns counts of short-term and long-term memories.
	Stats(ctx context.Context, userID, agentID string) (shortTerm, longTerm int, err error)
}

// Distiller extracts structured memory from raw conversation turns.
// It uses an LLM to produce summaries, facts, and preferences.
type Distiller struct {
	llm provider.LLMProvider
}

// NewDistiller creates a memory distiller backed by an LLM provider.
func NewDistiller(llm provider.LLMProvider) *Distiller {
	return &Distiller{llm: llm}
}

// Distill extracts memories from a user message and agent reply pair.
func (d *Distiller) Distill(ctx context.Context, userMsg, agentReply string) *types.MemoryExtract {
	if d.llm == nil {
		return &types.MemoryExtract{}
	}

	prompt := `Extract worth-remembering info. Return JSON only:
{"summary":"...","topics":[],"emotion":"...","facts":[],"preferences":[]}
User: ` + userMsg + `
Agent: ` + agentReply

	reply, err := d.llm.Chat(ctx, []types.Message{{Role: "user", Content: prompt}}, provider.ModelConfig{
		Temperature: 0.1,
		MaxTokens:   200,
	})
	if err != nil {
		return &types.MemoryExtract{}
	}

	return parseMemoryExtract(reply)
}

// parseMemoryExtract decodes a JSON response into a MemoryExtract.
func parseMemoryExtract(raw string) *types.MemoryExtract {
	cleaned := trimJSONBlock(raw)
	var result types.MemoryExtract
	if json.Unmarshal([]byte(cleaned), &result) == nil {
		return &result
	}
	return &types.MemoryExtract{}
}

// trimJSONBlock removes markdown code fences around JSON.
func trimJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = s[7:]
	} else if strings.HasPrefix(s, "```") {
		s = s[3:]
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}

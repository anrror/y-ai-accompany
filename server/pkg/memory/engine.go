// Package memory 提供记忆即服务（MaaS）抽象。
// 此文件实现了三级记忆管线的核心引擎（Engine），包括工作记忆、短期记忆和长期记忆的管理。
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/y-ai-accompany/server/pkg/types"
)

// Engine 是核心记忆系统，实现三级记忆管线：
//
//	工作记忆（内存中，N 轮滑动窗口）
//	  → 短期记忆（LLM 蒸馏摘要，向量存储）
//	  → 长期记忆（固化事实，永久存储）
//
// 组合了 MemoryStore（持久化）和 Embedding 服务（向量检索）。
// 并发安全（通过 sync.RWMutex 保护）。
type Engine struct {
	mu        sync.RWMutex
	working   map[string][]types.Message   // key: "userID:agentID"
	shortTerm map[string][]types.MemoryEntry
	longTerm  map[string][]types.MemoryEntry

	store       Store
	embedder    Embedder
	distiller   *Distiller

	emu       sync.RWMutex      // 单独的锁保护 embedCache，避免与 mu 混合使用时的死锁/竞态
	embedCache map[string][]float32
}

// Embedder 是 Engine 所需的 Embedding 接口（比 provider 包中的定义更精简）。
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// EngineConfig 包含 Engine 构造时的可选依赖项。
// 所有字段均可为 nil，此时对应功能使用内存回退（in-memory fallback）。
type EngineConfig struct {
	Store     Store
	Embedder  Embedder
	Distiller *Distiller
}

// NewEngine 创建一个记忆引擎。config 中的 nil 字段会导致对应组件使用内存回退行为。
func NewEngine(cfg EngineConfig) *Engine {
	return &Engine{
		working:    make(map[string][]types.Message),
		shortTerm:  make(map[string][]types.MemoryEntry),
		longTerm:   make(map[string][]types.MemoryEntry),
		store:      cfg.Store,
		embedder:   cfg.Embedder,
		distiller:  cfg.Distiller,
		embedCache: make(map[string][]float32),
	}
}

// memKey 生成记忆存储的内部键，格式为 "userID:agentID"。
func memKey(userID, agentID string) string {
	return fmt.Sprintf("%s:%s", userID, agentID)
}

// AddWorking 将一条消息追加到工作记忆缓冲区。
// role 参数为 "user" 或 "assistant"，content 为消息内容。
func (e *Engine) AddWorking(userID, agentID, role, content string) {
	msg := types.Message{Role: role, Content: content}
	key := memKey(userID, agentID)

	e.mu.Lock()
	e.working[key] = append(e.working[key], msg)
	e.mu.Unlock()
}

// GetWorking 返回工作记忆的副本（避免外部修改内部状态）。
func (e *Engine) GetWorking(userID, agentID string) []types.Message {
	e.mu.RLock()
	defer e.mu.RUnlock()

	key := memKey(userID, agentID)
	msgs := e.working[key]
	result := make([]types.Message, len(msgs))
	copy(result, msgs)
	return result
}

// TrimWorking 将工作记忆截断为最近的 maxTurns*2 条消息（用户+助手各一条为一轮）。
func (e *Engine) TrimWorking(userID, agentID string, maxTurns int) {
	maxLen := maxTurns * 2
	e.mu.Lock()
	defer e.mu.Unlock()

	key := memKey(userID, agentID)
	if msgs := e.working[key]; len(msgs) > maxLen {
		e.working[key] = msgs[len(msgs)-maxLen:]
	}
}

// Search 检索与查询相关的记忆条目。
// 如果配置了 Store，委托给 Store.Search（如 pgvector 向量检索）；
// 否则使用内存搜索（searchInMemory），通过余弦相似度排序。
func (e *Engine) Search(ctx context.Context, userID, agentID, query string, topK int) ([]types.MemoryEntry, error) {
	if e.store != nil {
		return e.store.Search(ctx, userID, agentID, query, topK)
	}
	return e.searchInMemory(ctx, userID, agentID, query, topK)
}

// searchInMemory 在内存中执行记忆检索。
// 如果有 Embedder，使用余弦相似度对短期+长期记忆排序；
// 否则直接返回最近的 topK 条记忆。
func (e *Engine) searchInMemory(ctx context.Context, userID, agentID, query string, topK int) ([]types.MemoryEntry, error) {
	e.mu.RLock()
	key := memKey(userID, agentID)
	short := e.shortTerm[key]
	long := e.longTerm[key]
	e.mu.RUnlock()

	all := append(short, long...)
	if len(all) == 0 {
		return nil, nil
	}

	// If no embedder, return most recent
	if e.embedder == nil {
		if len(all) > topK {
			all = all[len(all)-topK:]
		}
		return all, nil
	}

	queryVec, err := e.getEmbedding(ctx, query)
	if err != nil {
		if len(all) > topK {
			all = all[len(all)-topK:]
		}
		return all, nil
	}

	type scored struct {
		entry types.MemoryEntry
		score float64
	}

	var scoredEntries []scored
	for _, entry := range all {
		entryVec, err := e.getEmbedding(ctx, entry.Content)
		if err != nil {
			continue
		}
		score := cosineSimilarity(queryVec, entryVec)
		scoredEntries = append(scoredEntries, scored{entry: entry, score: score})
	}

	// 按分数降序排列（替换原来的冒泡排序 O(n²)）
	sort.Slice(scoredEntries, func(i, j int) bool {
		return scoredEntries[i].score > scoredEntries[j].score
	})

	if topK > len(scoredEntries) {
		topK = len(scoredEntries)
	}
	result := make([]types.MemoryEntry, topK)
	for i := 0; i < topK; i++ {
		result[i] = scoredEntries[i].entry
	}
	return result, nil
}

// Save 持久化一条记忆条目到存储和内存缓存中。
// 如果配置了 Store，优先写入 Store；否则写入内存 map。
// 根据 MemoryType 字段（"short_term" 或 "long_term"）决定存入短期或长期记忆。
func (e *Engine) Save(userID, agentID string, entry types.MemoryEntry) {
	entry.UserID = userID
	entry.AgentID = agentID
	if entry.Timestamp == 0 {
		entry.Timestamp = time.Now().Unix()
	}

	if e.store != nil {
		if err := e.store.Save(context.Background(), &entry); err != nil {
			// Logging is left to the caller/store layer
		}
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	key := memKey(userID, agentID)
	if entry.MemoryType == "long_term" {
		e.longTerm[key] = append(e.longTerm[key], entry)
	} else {
		e.shortTerm[key] = append(e.shortTerm[key], entry)
	}
}

// DeleteAll 删除指定用户-Agent 对的所有记忆（工作记忆、短期记忆、长期记忆）。
func (e *Engine) DeleteAll(userID, agentID string) {
	key := memKey(userID, agentID)

	e.mu.Lock()
	delete(e.working, key)
	delete(e.shortTerm, key)
	delete(e.longTerm, key)
	e.mu.Unlock()

	if e.store != nil {
		e.store.DeleteAll(context.Background(), userID, agentID)
	}
}

// BuildContext 检索相关记忆并格式化为 System Prompt 可注入的文本。
// 返回格式如 "[2024-01-01 短期记忆] 用户喜欢..."，每条记忆一行。
// 如果未找到相关记忆，返回 "无相关记忆"。
func (e *Engine) BuildContext(ctx context.Context, userID, agentID, query string) string {
	memories, err := e.Search(ctx, userID, agentID, query, 3)
	if err != nil || len(memories) == 0 {
		return "无相关记忆"
	}

	var lines []string
	for _, m := range memories {
		label := "短期记忆"
		if m.MemoryType == "long_term" {
			label = "长期记忆"
		}
		ts := time.Unix(m.Timestamp, 0).Format("2006-01-02")
		lines = append(lines, fmt.Sprintf("[%s %s] %s", ts, label, m.Content))
	}
	return strings.Join(lines, "\n")
}

// Stats 返回记忆统计信息：工作记忆轮次、短期记忆条数、长期记忆条数。
// 如果配置了 Store，会合并 Store 中的统计数据。
func (e *Engine) Stats(userID, agentID string) map[string]int {
	e.mu.RLock()
	key := memKey(userID, agentID)
	working := len(e.working[key]) / 2
	short := len(e.shortTerm[key])
	long := len(e.longTerm[key])
	e.mu.RUnlock()

	if e.store != nil {
		s, l, err := e.store.Stats(context.Background(), userID, agentID)
		if err == nil {
			short += s
			long += l
		}
	}

	return map[string]int{
		"working_turns": working,
		"short_term":    short,
		"long_term":     long,
	}
}

// DistillAndSave 对一轮对话进行记忆蒸馏，并将提取的记忆保存。
// 蒸馏结果包括：摘要（存入短期记忆）和固化事实（存入长期记忆）。
// 如果未配置 Distiller，直接返回不执行任何操作。
func (e *Engine) DistillAndSave(ctx context.Context, userID, agentID, userMsg, agentReply string) {
	if e.distiller == nil {
		return
	}
	mem := e.distiller.Distill(ctx, userMsg, agentReply)
	if mem.Summary != "" {
		e.Save(userID, agentID, types.MemoryEntry{
			Content:      mem.Summary,
			Topics:       mem.Topics,
			EmotionLabel: mem.Emotion,
			MemoryType:   "short_term",
			Importance:   0.6,
		})
	}
	for _, fact := range mem.Facts {
		e.Save(userID, agentID, types.MemoryEntry{
			Content:    fact,
			MemoryType: "long_term",
			Importance: 0.8,
		})
	}
}

// getEmbedding 返回缓存的或新计算的文本嵌入向量。
// 使用独立的 emu 锁，避免与 Engine.mu 混合导致的竞态条件。
// 缓存超过 10000 条时自动淘汰一半旧条目，防止内存泄漏。
func (e *Engine) getEmbedding(ctx context.Context, text string) ([]float32, error) {
	// 先读缓存（只加读锁）
	e.emu.RLock()
	if cached, ok := e.embedCache[text]; ok {
		e.emu.RUnlock()
		return cached, nil
	}
	e.emu.RUnlock()

	// 缓存未命中，调用外部 embedding 服务
	embedding, err := e.embedder.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	// 写入缓存（加写锁）
	e.emu.Lock()
	e.embedCache[text] = embedding
	// 缓存超过 10000 条时，删除一半旧条目
	// 使用先入先出（FIFO）策略: map 迭代顺序不可预期，
	// 简单的计数淘汰即可满足需求
	if len(e.embedCache) > 10000 {
		evictCount := len(e.embedCache) / 2
		for k := range e.embedCache {
			if evictCount <= 0 {
				break
			}
			delete(e.embedCache, k)
			evictCount--
		}
	}
	e.emu.Unlock()
	return embedding, nil
}

// cosineSimilarity 计算两个 float32 向量的余弦相似度。
// 如果向量长度不同或为空，返回 0。结果范围 [-1, 1]。
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// extractJSON is a minimal JSON unmarshal used by distiller.
func extractJSON(dst interface{}, raw string) bool {
	return json.Unmarshal([]byte(raw), dst) == nil
}

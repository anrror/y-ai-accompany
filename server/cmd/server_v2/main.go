// Package main is the framework-based server entrypoint.
// Uses y-ai-agent-base public packages for agent/pipeline/provider management
// with a custom Gin HTTP layer.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/anrror/y-ai-agent-base/pkg/agent"
	appcfg "github.com/anrror/y-ai-agent-base/pkg/config"
	"github.com/anrror/y-ai-agent-base/pkg/pipeline"
	"github.com/anrror/y-ai-agent-base/pkg/provider"
	"github.com/anrror/y-ai-agent-base/pkg/provider/openai"
	"github.com/anrror/y-ai-agent-base/pkg/types"

	"github.com/y-ai-accompany/server/internal/accompany"
)

func main() {
	cfg, err := appcfg.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// --- Core components ---
	prov := openai.NewOpenAIProvider(&provider.ProviderConfig{
		Type:    cfg.Providers.Chat.Type,
		APIKey:  cfg.Providers.Chat.APIKey,
		BaseURL: cfg.Providers.Chat.BaseURL,
		Model:   cfg.Providers.Chat.Model,
	})

	metrics := pipeline.NewMetrics()
	pipe := pipeline.New(prov,
		pipeline.MetricsMiddleware(metrics),
		// 注意：pipeline.Timeout 会导致 streaming context 被过早取消，
		// streaming 由 Gin handler 层面 的 ReadTimeout/WriteTimeout 兜底。
	)

	reg := agent.NewRegistry()

	// --- Accompany extensions (inject middleware ONCE into shared pipeline) ---
	acc := accompany.New(accompany.DefaultConfig())
	if mw := acc.Safety(); mw != nil {
		pipe.Use(mw.Middleware())
	}
	if mw := acc.Emotion(); mw != nil {
		pipe.Use(mw.Middleware())
	}
	if mw := acc.Memory(); mw != nil {
		pipe.Use(mw.Middleware())
	}
	if mw := acc.Personality(); mw != nil {
		pipe.Use(mw.Middleware())
	}
	slog.Info("accompany module initialized",
		"safety", acc.Safety() != nil,
		"emotion", acc.Emotion() != nil,
		"memory", acc.Memory() != nil,
		"personality", acc.Personality() != nil,
	)

	// --- Seed agents ---
	if err := seedDefaultAgents(reg, prov, pipe); err != nil {
		log.Fatalf("seed agents: %v", err)
	}

	// --- HTTP server (Gin) ---
	gin.SetMode(ginMode(cfg.Server.Mode))
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", healthHandler(prov))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/chat/completions", chatHandler(reg, prov, pipe))
		v1.GET("/agents", listAgentsHandler(reg))
		v1.GET("/agents/:id", getAgentHandler(reg))
		v1.GET("/debug/memory", debugMemoryHandler(acc.Memory()))
	}

	port := cfg.Server.Port
	if port == 0 {
		port = 8080
	}
	addr := fmt.Sprintf(":%d", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server_v2 listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "error", err)
		}
	}()

	<-quit
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// ─── Handlers ────────────────────────────────────────────────────────────────

func healthHandler(prov provider.LLMProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "ok"
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if err := prov.Ping(ctx); err != nil {
			status = "degraded"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":    status,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// chatRequest embeds the framework's ChatCompletionRequest with an extra UserID field.
type chatRequest struct {
	UserID string `json:"user_id"`
	types.ChatCompletionRequest
}

func chatHandler(reg *agent.Registry, prov provider.LLMProvider, pipe pipeline.Pipeline) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req chatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
			return
		}
		if req.Model == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
			return
		}
		if len(req.Messages) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "messages array must not be empty"})
			return
		}

		ag, ok := reg.Get(req.Model)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("agent %q not found", req.Model)})
			return
		}

		agCfg := ag.GetConfig()
		modelCfg := types.ModelConfig{
			Model:       agCfg.LLMConfig.Model,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		}

		input := types.ChatInput{
			Messages:    req.Messages,
			ModelConfig: &modelCfg,
			UserID:      req.UserID,
		}

		if req.Stream {
			handleStream(c, ag, input)
			return
		}
		handleJSON(c, ag, input)
	}
}

// debugMemoryHandler 返回指定 user_id + agent_id 的记忆统计和内容（仅调试用）。
func debugMemoryHandler(mem *accompany.MemoryExtension) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mem == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "memory extension not available"})
			return
		}
		eng := mem.Engine()
		userID := c.Query("user_id")
		agentID := c.Query("agent_id")
		if userID == "" || agentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and agent_id query params required"})
			return
		}
		stats := eng.Stats(userID, agentID)
		c.JSON(http.StatusOK, gin.H{
			"stats":   stats,
			"working": eng.GetWorking(userID, agentID),
		})
	}
}

func handleJSON(c *gin.Context, ag *agent.Agent, input types.ChatInput) {
	output, err := ag.Run(c.Request.Context(), input)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "agent run failed", "agent_id", ag.ID(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      genChatID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   ag.ID(),
		"choices": []gin.H{
			{
				"index":         0,
				"message":       gin.H{"role": output.Role, "content": output.Content},
				"finish_reason": "stop",
			},
		},
		"usage": gin.H{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
	})
}

func handleStream(c *gin.Context, ag *agent.Agent, input types.ChatInput) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	events := make(chan types.StreamEvent, 32)
	id := genChatID()
	model := ag.ID()
	created := time.Now().Unix()

	flusher, canFlush := c.Writer.(http.Flusher)
	send := func(data gin.H) {
		b, _ := json.Marshal(data)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		if canFlush {
			flusher.Flush()
		}
	}

	send(gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []gin.H{{"index": 0, "delta": gin.H{"role": "assistant"}}},
	})

	go func() {
		defer close(events)
		if err := ag.RunStream(c.Request.Context(), input, events); err != nil {
			select {
			case events <- types.StreamEvent{Done: true, Error: err}:
			default:
			}
		}
	}()

	finishReason := "stop"
	for evt := range events {
		if evt.Error != nil {
			slog.ErrorContext(c.Request.Context(), "stream error", "agent_id", model, "error", evt.Error)
			break
		}
		if evt.Done {
			break
		}
		if evt.Content != "" {
			send(gin.H{
				"id":      id,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   model,
				"choices": []gin.H{{"index": 0, "delta": gin.H{"content": evt.Content}}},
			})
		}
	}

	send(gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []gin.H{{"index": 0, "finish_reason": &finishReason}},
	})

	_, _ = fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	if canFlush {
		flusher.Flush()
	}
}

func listAgentsHandler(reg *agent.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		agents := reg.List()
		result := make([]gin.H, 0, len(agents))
		for _, ag := range agents {
			cfg := ag.GetConfig()
			result = append(result, gin.H{
				"agent_id": cfg.AgentID,
				"name":     agentName(cfg.Identity),
				"status":   cfg.Status,
				"model":    cfg.LLMConfig.Model,
			})
		}
		c.JSON(http.StatusOK, gin.H{"agents": result, "total": len(result)})
	}
}

func getAgentHandler(reg *agent.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		ag, ok := reg.Get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("agent %q not found", id)})
			return
		}
		cfg := ag.GetConfig()
		c.JSON(http.StatusOK, gin.H{
			"agent_id":    cfg.AgentID,
			"identity":    cfg.Identity,
			"personality": cfg.Personality,
			"model":       cfg.LLMConfig.Model,
			"status":      cfg.Status,
		})
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func genChatID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	}
	return "chatcmpl-" + hex.EncodeToString(b)
}

func agentName(id *agent.Identity) string {
	if id != nil && id.Name != "" {
		return id.Name
	}
	return ""
}

func ginMode(mode string) string {
	switch mode {
	case "production":
		return gin.ReleaseMode
	case "test":
		return gin.TestMode
	default:
		return gin.DebugMode
	}
}

// ─── Agent YAML Config ──────────────────────────────────────────────────────

// agentYAMLConfig matches the schema of agents/*.yaml files.
type agentYAMLConfig struct {
	AgentID  string `yaml:"agent_id"`
	Identity struct {
		Name          string `yaml:"name"`
		Avatar        string `yaml:"avatar"`
		Persona       string `yaml:"persona"`
		SpeakingStyle string `yaml:"speaking_style"`
		Greeting      string `yaml:"greeting"`
	} `yaml:"identity"`
	Personality struct {
		Openness          float64 `yaml:"openness"`
		Conscientiousness float64 `yaml:"conscientiousness"`
		Extraversion      float64 `yaml:"extraversion"`
		Agreeableness     float64 `yaml:"agreeableness"`
		Neuroticism       float64 `yaml:"neuroticism"`
	} `yaml:"personality"`
	Capabilities struct {
		LLMConfig struct {
			Model       string  `yaml:"model"`
			Temperature float64 `yaml:"temperature"`
			MaxTokens   int     `yaml:"max_tokens"`
		} `yaml:"llm_config"`
		PromptTemplate string `yaml:"prompt_template"`
	} `yaml:"capabilities"`
	MemoryConfig struct {
		Enabled            bool `yaml:"enabled"`
		RetrievalTopK      int  `yaml:"retrieval_top_k"`
		WorkingMemoryTurns int  `yaml:"working_memory_turns"`
	} `yaml:"memory_config"`
	SafetyConfig struct {
		InputGuardEnabled  bool `yaml:"input_guard_enabled"`
		OutputGuardEnabled bool `yaml:"output_guard_enabled"`
		SafetyNoticeEnabled bool `yaml:"safety_notice_enabled"`
	} `yaml:"safety_config"`
}

// ─── Agent Seeding ───────────────────────────────────────────────────────────

func seedDefaultAgents(reg *agent.Registry, prov provider.LLMProvider, pipe pipeline.Pipeline) error {
	entries, err := os.ReadDir("agents")
	if err != nil {
		return fmt.Errorf("read agents directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := "agents/" + entry.Name()
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("skip agent file, read error", "path", path, "error", err)
			continue
		}

		var yc agentYAMLConfig
		if err := yaml.Unmarshal(data, &yc); err != nil {
			slog.Warn("skip agent file, yaml parse error", "path", path, "error", err)
			continue
		}
		if yc.AgentID == "" {
			slog.Warn("skip agent file, missing agent_id", "path", path)
			continue
		}

		ocean := agent.OCEAN{
			Openness:          yc.Personality.Openness,
			Conscientiousness: yc.Personality.Conscientiousness,
			Extraversion:      yc.Personality.Extraversion,
			Agreeableness:     yc.Personality.Agreeableness,
			Neuroticism:       yc.Personality.Neuroticism,
		}
		promptTmpl := renderPrompt(yc.Capabilities.PromptTemplate, yc, ocean)

		cfg := agent.Config{
			AgentID: yc.AgentID,
			Identity: &agent.Identity{
				Name:        yc.Identity.Name,
				Role:        yc.Identity.Name + "的AI陪伴角色",
				Description: yc.Identity.Persona,
				Tone:        yc.Identity.SpeakingStyle,
				Verbosity:   "medium",
			},
			Personality: ocean,
			LLMConfig: types.ModelConfig{
				Model:       yc.Capabilities.LLMConfig.Model,
				Temperature: yc.Capabilities.LLMConfig.Temperature,
				MaxTokens:   yc.Capabilities.LLMConfig.MaxTokens,
			},
			PromptTmpl: promptTmpl,
			Status:     agent.StatusReady,
		}
		// Map YAML memory_config to framework types.MemoryConfig.
		cfg.MemoryConfig.MaxEntries = 100
		cfg.MemoryConfig.TTLMillis = 3_600_000
		cfg.MemoryConfig.Consolidation = true
		// Map YAML safety_config to framework types.SafetyConfig.
		cfg.SafetyConfig.Enabled = yc.SafetyConfig.InputGuardEnabled || yc.SafetyConfig.OutputGuardEnabled
		cfg.SafetyConfig.InputGuard = yc.SafetyConfig.InputGuardEnabled
		cfg.SafetyConfig.OutputGuard = yc.SafetyConfig.OutputGuardEnabled
		cfg.FillDefaults()

		ag, err := cfg.ToBuilder().
			WithProvider(prov).
			WithPipeline(pipe).
			Build()
		if err != nil {
			slog.Warn("skip agent, build error", "id", yc.AgentID, "error", err)
			continue
		}

		if err := reg.Register(ag); err != nil {
			slog.Warn("skip agent, register error", "id", yc.AgentID, "error", err)
			continue
		}
		slog.Info("agent registered", "id", yc.AgentID, "name", yc.Identity.Name)
	}

	return nil
}

// renderPrompt replaces template placeholders with actual agent data.
func renderPrompt(tmpl string, yc agentYAMLConfig, o agent.OCEAN) string {
	r := strings.NewReplacer(
		"{agent_name}", yc.Identity.Name,
		"{persona}", yc.Identity.Persona,
		"{speaking_style}", yc.Identity.SpeakingStyle,
		"{greeting}", yc.Identity.Greeting,
		"{intimacy}", "0",
		"{memories}", "",
		"{personality_rules}", fmt.Sprintf(
			"开放性(创造力/好奇心): %.1f\n尽责性(条理性/可靠性): %.1f\n外向性(社交性/活力): %.1f\n宜人性(同理心/合作): %.1f\n神经质(敏感度/情绪波动): %.1f",
			o.Openness, o.Conscientiousness, o.Extraversion, o.Agreeableness, o.Neuroticism,
		),
	)
	return r.Replace(tmpl)
}

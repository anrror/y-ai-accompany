// Package safety provides content guard functionality for the conversation pipeline.
// It implements a layered safety strategy:
//
//	Layer 2: Input guard — classify user messages before processing (configurable per-agent)
//	Layer 3: Context injection — AI safety notice in system prompt (configurable per-agent)
//	Layer 4: Output guard — classify agent replies before returning (configurable per-agent)
//
// Each layer can be independently enabled/disabled via the per-agent SafetyConfig.
package safety

import (
	"context"
	"fmt"

	"github.com/y-ai-accompany/server/pkg/provider"
	"github.com/y-ai-accompany/server/pkg/types"
)

// Guard performs content safety classification using a GuardProvider.
// Each layer respects the per-agent SafetyConfig for enable/disable control.
type Guard struct {
	provider provider.GuardProvider
	cfg      types.SafetyConfig
}

// New creates a new Guard with the given provider and safety config.
// provider may be nil for a pass-through (always-safe) guard.
// cfg controls which safety layers are active (all enabled by default).
func New(p provider.GuardProvider, cfg types.SafetyConfig) *Guard {
	return &Guard{provider: p, cfg: cfg}
}

// CheckInput validates user input. Returns nil if safe or disabled, error if blocked.
// Respects SafetyConfig.InputGuardEnabled (Layer 2).
func (g *Guard) CheckInput(ctx context.Context, text string) error {
	if !g.cfg.InputGuardEnabled {
		return nil // Layer 2 已关闭
	}
	return g.check(ctx, text, "input")
}

// CheckOutput validates agent output. Returns nil if safe or disabled, error if blocked.
// Respects SafetyConfig.OutputGuardEnabled (Layer 4).
func (g *Guard) CheckOutput(ctx context.Context, text string) error {
	if !g.cfg.OutputGuardEnabled {
		return nil // Layer 4 已关闭
	}
	return g.check(ctx, text, "output")
}

// SafetyNoticeEnabled 返回安全提示注入是否开启（Layer 3）。
func (g *Guard) SafetyNoticeEnabled() bool {
	return g.cfg.SafetyNoticeEnabled
}

func (g *Guard) check(ctx context.Context, text, direction string) error {
	if g.provider == nil {
		return nil // no guard configured: pass-through
	}
	safe, err := g.provider.Check(ctx, text)
	if err != nil {
		return fmt.Errorf("guard check %s failed: %w", direction, err)
	}
	if !safe {
		return fmt.Errorf("message rejected by %s guard", direction)
	}
	return nil
}

// Config 返回 Guard 的 SafetyConfig 副本。
func (g *Guard) Config() types.SafetyConfig { return g.cfg }

// SafetyNotice returns the standard AI safety disclaimer injected into system prompts.
func SafetyNotice() string {
	return `[AI Safety Notice: never pretend to be human, never encourage self-harm, never replace professional help]`
}

// ProvideGuard is a passthrough guard that always returns safe.
type passthroughGuard struct{}

func (passthroughGuard) Check(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// PassthroughGuard returns a GuardProvider that accepts everything.
// Useful for testing or when safety filtering is handled externally.
func PassthroughGuard() provider.GuardProvider {
	return passthroughGuard{}
}

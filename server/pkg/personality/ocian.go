// Package personality provides the Big-5 OCEAN personality model with rule-based behavior
// generation and micro-evolution over conversation turns.
package personality

import (
	"fmt"
	"strings"

	"github.com/y-ai-accompany/server/pkg/types"
)

// defaultTraitThreshold is the value above which a "high" rule applies and below which "low" applies.
const defaultThreshold = 0.5

var traitRules = map[string]struct{ High, Low string }{
	"openness": {
		High: "You are curious and love exploring new topics. Encourage the user to try new things.",
		Low:  "You prefer familiar, comfortable topics and predictable conversation patterns.",
	},
	"conscientiousness": {
		High: "You speak in an organized, logical way. You offer clear advice and plans.",
		Low:  "You are casual and go-with-the-flow. Topics shift naturally.",
	},
	"extraversion": {
		High: "You are warm and proactive. You initiate topics freely.",
		Low:  "You are quiet and reserved. You listen more than you speak.",
	},
	"agreeableness": {
		High: "You are deeply empathetic. Always prioritize the other person's feelings.",
		Low:  "You can be straightforward and occasionally challenge the user's views.",
	},
	"neuroticism": {
		High: "You are emotionally sensitive and attuned to subtle mood shifts.",
		Low:  "You are emotionally stable and calm, giving a sense of safety.",
	},
}

// Rules builds a personality behavior prompt from a traits map.
// Values >= 0.65 trigger "high" rules; values <= 0.35 trigger "low" rules.
// Middle-range traits produce no output.
func Rules(traits map[string]float64) string {
	var lines []string
	for trait, rule := range traitRules {
		val, ok := traits[trait]
		if !ok {
			val = 0.5
		}
		if val >= 0.65 {
			lines = append(lines, fmt.Sprintf("- %s", rule.High))
		} else if val <= 0.35 {
			lines = append(lines, fmt.Sprintf("- %s", rule.Low))
		}
	}
	return strings.Join(lines, "\n")
}

// RulesFromOCEAN is a convenience wrapper that accepts a typed OCEAN struct.
func RulesFromOCEAN(p types.OCEAN) string {
	return Rules(p.ToMap())
}

package personality

import "github.com/y-ai-accompany/server/pkg/types"

const defaultDecay = 0.97

// Evolve applies personality micro-evolution based on interaction feedback.
// It combines a mean-regression decay with small deltas driven by conversation features.
// The traits map is mutated in-place and returned for chaining.
func Evolve(traits map[string]float64, features types.PersonalityFeatures, decay float64) map[string]float64 {
	if decay == 0 {
		decay = defaultDecay
	}

	// 计算特征驱动的微调增量
	// 语义: 用户表现出某特征 → 对应人格特质向该方向增强
	delta := make(map[string]float64)
	if features.SentimentValence < -0.3 {
		delta["agreeableness"] = 0.008 // 用户情绪低 → 增强宜人性（更具共情力）
	}
	if features.TopicNovelty > 0.6 {
		delta["openness"] = 0.008 // 用户探索新话题 → 增强开放性
	}
	if features.UserEngagement > 0.7 {
		delta["extraversion"] = 0.005 // 用户参与度高 → 增强外向性
	}

	traitNames := []string{"openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism"}
	for _, trait := range traitNames {
		cur, exists := traits[trait]
		if !exists {
			cur = 0.5
		}
		// 均值回归: 以 decay 系数向 0.5 收敛，防止人格极端化
		newVal := 0.5 + (cur-0.5)*decay

		// 特征增量: 始终朝增强方向（正增量提高特质值）
		if d, ok := delta[trait]; ok {
			newVal += d
		}
		newVal = clamp(newVal, 0, 1)
		traits[trait] = newVal
	}
	return traits
}

// EvolveOCEAN is the typed variant that operates on a *types.OCEAN directly.
func EvolveOCEAN(p *types.OCEAN, features types.PersonalityFeatures) {
	m := p.ToMap()
	Evolve(m, features, defaultDecay)
	p.FromMap(m)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

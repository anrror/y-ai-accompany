// Package emotion — 情绪关键词词典。包含 60+ 中文情绪关键词到情绪类别的映射。
package emotion

// keyword maps Chinese keywords to emotion categories with intensity.
// kwEntry 将中文关键词映射到情绪类别及强度。
type kwEntry struct {
	emotion   string
	intensity float64
}

// keywords is the emotion lexicon — 60+ Chinese emotion keywords.
// keywords 是情绪词典——60+ 中文情绪关键词，按情绪类别分组。
var keywords = map[string]kwEntry{
	// === Joy ===
	"开心": {"joy", 0.7}, "快乐": {"joy", 0.7}, "高兴": {"joy", 0.6},
	"幸福": {"joy", 0.7}, "喜悦": {"joy", 0.7}, "愉快": {"joy", 0.6},
	"欢乐": {"joy", 0.7}, "哈哈": {"joy", 0.7}, "嘻嘻": {"joy", 0.6},
	"嘿嘿": {"joy", 0.5}, "美好": {"joy", 0.5}, "甜蜜": {"joy", 0.6},

	// === Sadness ===
	"难过": {"sadness", 0.7}, "伤心": {"sadness", 0.8}, "悲伤": {"sadness", 0.8},
	"悲痛": {"sadness", 0.9}, "心碎": {"sadness", 0.9}, "哭": {"sadness", 0.7},
	"哭泣": {"sadness", 0.8}, "流泪": {"sadness", 0.7},
	"失落": {"sadness", 0.6}, "沮丧": {"sadness", 0.7},
	"绝望": {"sadness", 0.9}, "孤独": {"sadness", 0.7},
	"寂寞": {"sadness", 0.6}, "空虚": {"sadness", 0.5},

	// === Anger ===
	"生气": {"anger", 0.7}, "愤怒": {"anger", 0.8}, "恼火": {"anger", 0.7},
	"烦": {"anger", 0.5}, "烦躁": {"anger", 0.6}, "不耐烦": {"anger", 0.5},
	"讨厌": {"anger", 0.6}, "恨": {"anger", 0.8}, "恶心": {"anger", 0.6},

	// === Fear ===
	"害怕": {"fear", 0.8}, "恐惧": {"fear", 0.9}, "惊慌": {"fear", 0.7},
	"恐慌": {"fear", 0.9},

	// === Anxiety ===
	"担心": {"anxiety", 0.6}, "焦虑": {"anxiety", 0.8},
	"紧张": {"anxiety", 0.6}, "不安": {"anxiety", 0.6},
	"压力": {"anxiety", 0.6}, "崩溃": {"anxiety", 0.8},
	"疲惫": {"anxiety", 0.5}, "累": {"anxiety", 0.4},

	// === Love ===
	"爱": {"love", 0.8}, "喜欢": {"love", 0.7}, "欣赏": {"love", 0.5},
	"想念": {"love", 0.6}, "思念": {"love", 0.6}, "牵挂": {"love", 0.5},
	"温柔": {"love", 0.5}, "温暖": {"love", 0.5},

	// === Gratitude ===
	"谢谢": {"gratitude", 0.5}, "感谢": {"gratitude", 0.6},
	"感恩": {"gratitude", 0.7}, "感激": {"gratitude", 0.7},

	// === Surprise ===
	"惊讶": {"surprise", 0.7}, "吃惊": {"surprise", 0.7},
	"想不到": {"surprise", 0.5}, "没想到": {"surprise", 0.5},

	// === Neutral / Mixed ===
	"嗯": {"neutral", 0.1}, "哦": {"neutral", 0.1}, "好吧": {"neutral", 0.2},
	"也许": {"neutral", 0.1}, "可能": {"neutral", 0.1},
	"平静": {"neutral", 0.2}, "放松": {"joy", 0.5},
	"释然": {"joy", 0.4}, "安心": {"joy", 0.4},
}

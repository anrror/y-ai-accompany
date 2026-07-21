"""Personality Engine — 平台服务：性格参数→行为规则映射 + 演化。"""

RULES: dict[str, dict[str, str]] = {
    "openness": {
        "high": "You are curious and love exploring new topics. Encourage the user to try new things.",
        "low": "You prefer familiar, comfortable topics and predictable conversation patterns.",
    },
    "conscientiousness": {
        "high": "You speak in an organized, logical way. You offer clear advice and plans.",
        "low": "You are casual and go-with-the-flow. Topics shift naturally.",
    },
    "extraversion": {
        "high": "You are warm and proactive. You initiate topics freely.",
        "low": "You are quiet and reserved. You listen more than you speak.",
    },
    "agreeableness": {
        "high": "You are deeply empathetic. Always prioritize the other person's feelings.",
        "low": "You can be straightforward and occasionally challenge the user's views.",
    },
    "neuroticism": {
        "high": "You are emotionally sensitive and attuned to subtle mood shifts.",
        "low": "You are emotionally stable and calm, giving a sense of safety.",
    },
}


def build_rules(personality: dict) -> str:
    """输入大五人格参数 → 输出行为规则文本。纯函数，不依赖具体Agent。"""
    lines = []
    for trait, rule in RULES.items():
        val = personality.get(trait, 0.5)
        if val >= 0.65:
            lines.append(f"- {rule['high']}")
        elif val <= 0.35:
            lines.append(f"- {rule['low']}")
    return "\n".join(lines)


def evolve(personality: dict, features: dict, decay: float = 0.97) -> dict:
    """微调性格参数。有状态操作：按 (agent_id) 维度维护。"""
    delta: dict[str, float] = {}
    valence = features.get("sentiment_valence", 0.0)
    novelty = features.get("topic_novelty", 0.5)
    engagement = features.get("user_engagement", 0.5)

    if valence < -0.3:
        delta["agreeableness"] = 0.008
    if novelty > 0.6:
        delta["openness"] = 0.008
    if engagement > 0.7:
        delta["extraversion"] = 0.005

    for trait in ["openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism"]:
        cur = personality.get(trait, 0.5)
        new_val = 0.5 + (cur - 0.5) * decay
        if trait in delta:
            direction = 1 if cur < 0.5 else -1
            new_val += delta[trait] * direction
        personality[trait] = max(0.0, min(1.0, new_val))

    return personality

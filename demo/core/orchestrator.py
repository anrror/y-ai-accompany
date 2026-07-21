"""Conversation Orchestrator — 平台核心管线，服务所有已注册Agent。"""
import json
import time
from typing import Generator

from core.registry import AgentRegistry
from core.memory import MemoryEngine
from core.personality import build_rules, evolve
from core.llm import chat_stream, chat, guard_check


class Orchestrator:
    """对话编排器。不绑定任何Agent，运行时从Registry加载Agent配置。"""

    def __init__(self, registry: AgentRegistry, memory: MemoryEngine):
        self.registry = registry
        self.memory = memory

    # ── 公用子服务 ──

    def _detect_emotion(self, text: str) -> dict:
        prompt = (
            'Analyze emotion in this Chinese text. Return JSON only:\n'
            '{"emotion":"joy|sadness|anger|fear|surprise|neutral","intensity":0.0-1.0,"valence":-1.0-1.0}\n'
            f'Text: {text}'
        )
        try:
            raw = chat(
                messages=[{"role": "user", "content": prompt}],
                max_tokens=100, temperature=0.1,
            )
            cleaned = raw.strip().removeprefix("```json").removesuffix("```").strip()
            return json.loads(cleaned)
        except Exception:
            return {"emotion": "neutral", "intensity": 0.0, "valence": 0.0}

    def _extract_memories(self, user_msg: str, agent_reply: str) -> dict:
        prompt = (
            'Extract worth-remembering info. Return JSON only:\n'
            '{"summary":"...","topics":[],"emotion":"...","facts":[],"preferences":[]}\n'
            f'User: {user_msg}\nAgent: {agent_reply}'
        )
        try:
            raw = chat(
                messages=[{"role": "user", "content": prompt}],
                max_tokens=200, temperature=0.1,
            )
            cleaned = raw.strip().removeprefix("```json").removesuffix("```").strip()
            return json.loads(cleaned)
        except Exception:
            return {"summary": "", "topics": [], "emotion": "neutral", "facts": [], "preferences": []}

    # ── 对话管线 (同一套代码服务所有Agent) ──

    def chat(
        self,
        user_id: str,
        agent_id: str,
        message: str,
        session_id: str = "default",
    ) -> Generator[str, None, dict]:
        # 1. 从Registry加载Agent配置
        agent = self.registry.get(agent_id)
        if not agent:
            yield f"[平台] Agent '{agent_id}' 未注册"
            return {"error": "agent_not_found"}

        # 2. 安全审核 (平台服务)
        guard = guard_check(message)
        if not guard.get("safe", True):
            yield "[平台] 抱歉，这个话题我们换个时间再聊。"
            return {"error": "guard_rejected"}

        # 3. 情绪感知 (平台服务)
        emotion = self._detect_emotion(message)

        # 4. 检索记忆 (Memory as a Service)
        memories_text = self.memory.build_context(user_id, agent_id, message)

        # 5. 构建System Prompt
        personality_rules = build_rules(agent.personality)
        template = agent.capabilities.get("prompt_template", "")
        intimacy = self._calc_intimacy(user_id, agent_id)

        system_prompt = template.format(
            agent_name=agent.identity.get("name", agent_id),
            persona=agent.identity.get("persona", ""),
            personality_rules=personality_rules,
            speaking_style=agent.identity.get("speaking_style", ""),
            memories=memories_text,
            intimacy=intimacy,
        )

        # 6. 组装上下文
        llm_messages = [{"role": "system", "content": system_prompt}]
        working = self.memory.get_working(user_id, agent_id)
        for msg in working[-10:]:
            llm_messages.append({"role": msg["role"], "content": msg["content"]})
        llm_messages.append({"role": "user", "content": message})

        # 7. 用户消息入工作记忆
        self.memory.add_working(user_id, agent_id, "user", message)

        # 8. LLM推理 (使用Agent配置的模型参数)
        full_reply = ""
        llm_cfg = agent.capabilities.get("llm_config", {})
        try:
            stream = chat_stream(
                llm_messages,
                model=llm_cfg.get("model"),
                temperature=llm_cfg.get("temperature", 0.7),
                max_tokens=llm_cfg.get("max_tokens", 1024),
            )
            for chunk in stream:
                if not chunk.choices:
                    continue
                delta = chunk.choices[0].delta
                if delta and delta.content:
                    full_reply += delta.content
                    yield delta.content
        except Exception as e:
            yield f"\n[平台LLM错误: {e}]"
            self.memory.add_working(user_id, agent_id, "assistant", f"[Error: {e}]")
            return {"error": str(e)}

        # 9. 回复入工作记忆
        self.memory.add_working(user_id, agent_id, "assistant", full_reply)
        self.memory.trim_working(user_id, agent_id, agent.memory_config.get("working_memory_turns", 15))

        # 10. 异步后处理：记忆蒸馏 + 性格演化
        mem = self._extract_memories(message, full_reply)
        if mem.get("summary"):
            self.memory.save_memory(user_id, agent_id, {
                "content": mem["summary"],
                "topics": mem.get("topics", []),
                "emotion": mem.get("emotion", "neutral"),
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
                "type": "short_term",
            })
        for fact in mem.get("facts", []):
            self.memory.save_memory(user_id, agent_id, {
                "content": fact,
                "importance": 0.8,
                "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
                "type": "long_term",
            })

        evolve(
            agent.personality,
            {
                "sentiment_valence": emotion.get("valence", 0),
                "topic_novelty": 0.5,
                "user_engagement": 0.6 if len(message) > 10 else 0.3,
            },
        )

        return {"emotion": emotion, "memories_extracted": bool(mem.get("facts"))}

    def _calc_intimacy(self, user_id: str, agent_id: str) -> int:
        short = len(self.memory.load_memories(user_id, agent_id, "short_term"))
        long = len(self.memory.load_memories(user_id, agent_id, "long_term"))
        return min(100, short * 5 + long * 10)

    def get_summary(self, user_id: str, agent_id: str) -> dict:
        working = self.memory.get_working(user_id, agent_id)
        short = self.memory.load_memories(user_id, agent_id, "short_term")
        long = self.memory.load_memories(user_id, agent_id, "long_term")
        return {
            "working_turns": len(working) // 2,
            "short_term": len(short),
            "long_term": len(long),
        }

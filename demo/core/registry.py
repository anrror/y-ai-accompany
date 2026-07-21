"""Agent Registry — 管理所有已注册Agent的生命周期。"""
import copy
from typing import Optional


class AgentConfig:
    def __init__(self, data: dict):
        self.agent_id: str = data["agent_id"]
        self.identity: dict = data.get("identity", {})
        self.personality: dict = data.get("personality", {})
        self.capabilities: dict = data.get("capabilities", {})
        self.memory_config: dict = data.get("memory_config", {})
        self.safety_config: dict = data.get("safety_config", {})
        self.status: str = "active"

    def to_dict(self) -> dict:
        return {
            "agent_id": self.agent_id,
            "identity": self.identity,
            "personality": self.personality,
            "capabilities": self.capabilities,
            "memory_config": self.memory_config,
            "safety_config": self.safety_config,
            "status": self.status,
        }

    def clone(self):
        return AgentConfig(copy.deepcopy(self.to_dict()))


class AgentRegistry:
    """平台核心：Agent注册中心。管理所有接入的Agent。"""

    def __init__(self):
        self._agents: dict[str, AgentConfig] = {}

    def register(self, config: dict) -> AgentConfig:
        agent_id = config.get("agent_id", "")
        if not agent_id:
            raise ValueError("agent_id is required")
        if agent_id in self._agents:
            raise ValueError(f"Agent '{agent_id}' already registered")

        filled = self._fill_defaults(config)
        agent = AgentConfig(filled)
        self._agents[agent_id] = agent
        return agent

    def unregister(self, agent_id: str) -> bool:
        if agent_id not in self._agents:
            return False
        self._agents[agent_id].status = "inactive"
        del self._agents[agent_id]
        return True

    def get(self, agent_id: str) -> Optional[AgentConfig]:
        return self._agents.get(agent_id)

    def list(self) -> list[AgentConfig]:
        return list(self._agents.values())

    def _fill_defaults(self, config: dict) -> dict:
        cfg = copy.deepcopy(config)
        ident = cfg.setdefault("identity", {})
        ident.setdefault("avatar", "🤖")
        ident.setdefault("greeting", "")
        pers = cfg.setdefault("personality", {})
        for trait in ["openness", "conscientiousness", "extraversion", "agreeableness", "neuroticism"]:
            pers.setdefault(trait, 0.5)
        caps = cfg.setdefault("capabilities", {})
        llm = caps.setdefault("llm_config", {})
        llm.setdefault("model", "Qwen3.6-27B")
        llm.setdefault("temperature", 0.7)
        llm.setdefault("max_tokens", 1024)
        caps.setdefault("prompt_template", "")
        mem = cfg.setdefault("memory_config", {})
        mem.setdefault("enabled", True)
        mem.setdefault("retrieval_top_k", 5)
        mem.setdefault("working_memory_turns", 15)
        safe = cfg.setdefault("safety_config", {})
        safe.setdefault("input_guard_enabled", True)
        safe.setdefault("output_guard_enabled", True)
        safe.setdefault("safety_notice_enabled", True)
        return cfg

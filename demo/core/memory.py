"""Memory Engine — 平台服务(MaaS): 所有Agent共享的三级记忆系统。"""
import json
import os
import time
from collections import defaultdict

from core.llm import embed as _embed


class MemoryEngine:
    """平台记忆服务。按 (user_id, agent_id) 隔离。"""

    def __init__(self, data_dir: str = "memory_data"):
        self.data_dir = data_dir
        os.makedirs(data_dir, exist_ok=True)
        self._working: dict[str, list[dict]] = defaultdict(list)
        self._embed_cache: dict[str, list[float]] = {}

    # ── 路径隔离：每个 (user, agent) 独立文件 ──

    def _path(self, user_id: str, agent_id: str, kind: str) -> str:
        return os.path.join(self.data_dir, f"{user_id}__{agent_id}__{kind}.jsonl")

    # ── 工作记忆 ──

    def add_working(self, user_id: str, agent_id: str, role: str, content: str):
        key = f"{user_id}:{agent_id}"
        self._working[key].append({"role": role, "content": content, "ts": time.time()})

    def get_working(self, user_id: str, agent_id: str) -> list[dict]:
        return self._working.get(f"{user_id}:{agent_id}", [])

    def trim_working(self, user_id: str, agent_id: str, max_turns: int = 15):
        key = f"{user_id}:{agent_id}"
        total = len(self._working[key])
        if total > max_turns * 2:
            self._working[key] = self._working[key][-max_turns * 2:]

    # ── 短期/长期记忆 (持久化) ──

    def save_memory(self, user_id: str, agent_id: str, entry: dict):
        kind = entry.get("type", "short_term")
        with open(self._path(user_id, agent_id, kind), "a", encoding="utf-8") as f:
            f.write(json.dumps(entry, ensure_ascii=False) + "\n")

    def load_memories(self, user_id: str, agent_id: str, kind: str) -> list[dict]:
        path = self._path(user_id, agent_id, kind)
        if not os.path.exists(path):
            return []
        with open(path, encoding="utf-8") as f:
            return [json.loads(line) for line in f if line.strip()]

    # ── 检索 ──

    def retrieve(self, user_id: str, agent_id: str, query: str, top_k: int = 5) -> list[dict]:
        q_vec = self._get_embedding(query)
        candidates = []

        for kind in ("short_term", "long_term"):
            for entry in self.load_memories(user_id, agent_id, kind):
                text = entry.get("content", "")
                if not text:
                    continue
                e_vec = self._get_embedding(text)
                score = self._cosine_sim(q_vec, e_vec)
                candidates.append({**entry, "score": round(score, 4), "kind": kind})

        candidates.sort(key=lambda x: x["score"], reverse=True)
        return candidates[:top_k]

    def build_context(self, user_id: str, agent_id: str, query: str) -> str:
        memories = self.retrieve(user_id, agent_id, query, top_k=3)
        if not memories:
            return "无相关记忆"
        lines = []
        for m in memories:
            label = "短期记忆" if m["kind"] == "short_term" else "长期记忆"
            lines.append(f"[{m.get('timestamp', '')[:10]} {label}] {m['content']}")
        return "\n".join(lines)

    def clear_all(self, user_id: str, agent_id: str):
        key = f"{user_id}:{agent_id}"
        self._working.pop(key, None)
        for kind in ("short_term", "long_term"):
            path = self._path(user_id, agent_id, kind)
            if os.path.exists(path):
                os.remove(path)

    # ── 辅助 ──

    def _get_embedding(self, text: str) -> list[float]:
        if text not in self._embed_cache:
            self._embed_cache[text] = _embed(text)
        return self._embed_cache[text]

    @staticmethod
    def _cosine_sim(a: list[float], b: list[float]) -> float:
        dot = sum(x * y for x, y in zip(a, b))
        na = sum(x * x for x in a) ** 0.5
        nb = sum(x * x for x in b) ** 0.5
        return dot / (na * nb) if na * nb > 0 else 0.0

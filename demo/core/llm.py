"""LLM/AI 模型客户端 — 平台基础设施，所有Agent共享。"""
import os
from openai import OpenAI

_client: OpenAI | None = None


def _get_client() -> OpenAI:
    global _client
    if _client is None:
        _client = OpenAI(
            base_url=os.getenv("API_BASE_URL"),
            api_key=os.getenv("API_KEY"),
        )
    return _client


def chat_stream(messages: list[dict], **kwargs):
    client = _get_client()
    return client.chat.completions.create(
        model=kwargs.get("model", os.getenv("LLM_MODEL", "Qwen3.6-27B")),
        messages=messages,
        stream=True,
        temperature=kwargs.get("temperature", 0.7),
        max_tokens=kwargs.get("max_tokens", 1024),
    )


def chat(messages: list[dict], **kwargs) -> str:
    client = _get_client()
    resp = client.chat.completions.create(
        model=kwargs.get("model", os.getenv("LLM_MODEL", "Qwen3.6-27B")),
        messages=messages,
        stream=False,
        temperature=kwargs.get("temperature", 0.7),
        max_tokens=kwargs.get("max_tokens", 1024),
    )
    return resp.choices[0].message.content or ""


def embed(text: str) -> list[float]:
    client = _get_client()
    model = os.getenv("EMBEDDING_MODEL", "Qwen3-Embedding-8B")
    return client.embeddings.create(model=model, input=text).data[0].embedding


def guard_check(text: str) -> dict:
    client = _get_client()
    model = os.getenv("GUARD_MODEL", "Qwen3Guard-Gen-8B")
    resp = client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": text}],
        max_tokens=10,
    )
    raw = resp.choices[0].message.content or ""
    return {"safe": "unsafe" not in raw.lower() and "不安全" not in raw, "raw": raw}

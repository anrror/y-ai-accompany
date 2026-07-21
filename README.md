# y-ai-accompany — AI 情感 Agent 托管平台

> **给 AI 一个灵魂，让用户被记得。**

`y-ai-accompany` 是一个基于 [y-ai-agent-base](https://github.com/anrror/y-ai-agent-base) 框架的 **AI 情感 Agent 托管平台**。一行 YAML 配置即可创建一个拥有永久记忆、动态性格、情感感知、安全防御的 AI 伙伴——**Agent 接入即用，零代码变更**。

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)
[![API](https://img.shields.io/badge/API-OpenAI%20Compatible-blue)](https://platform.openai.com/docs/api-reference/chat)

---

## ✨ 核心优势

### 🧬 即插即用的扩展系统

通过框架的 `Extension` + `MiddlewareProvider` 接口，四大核心能力以独立模块注入 Agent 管线，**零入侵业务代码**：

| 模块 | 能力 | 注入方式 |
|------|------|----------|
| **Memory** | 三级记忆（工作→短期→长期），跨轮检索 | 前置注入记忆上下文 + 后置自动存储 |
| **Emotion** | 关键词 + Emoji 情绪检测，毫秒级响应 | 前置检测，结果写入 context |
| **Personality** | OCEAN 大五人格 + 交互驱动演化 | 后置微调，对话越多性格越鲜活 |
| **Safety** | 输入/输出安全审核，透传守卫 | 双向拦截不安全内容 |

### 🧠 三级记忆引擎——对话即记忆

```
用户输入 → 工作记忆(15轮滑动窗口) → 短期记忆(内存索引) → 长期记忆(固化事实)
                                    ↑ 蒸馏进化
```

- **工作记忆**：当前对话上下文，15 轮滑动窗口自动裁剪
- **短期记忆**：每次对话后自动保存摘要，通过关键词/语义检索召回
- **长期记忆**：重复提及 ≥3 次的事实自动固化，永久保留
- **跨轮检索**：Turn 1 聊过的话题，Turn 2 无需重复——Agent 自动记得
- **零外部依赖**：默认内存模式，pgvector/Redis 可选扩展

### 🎭 多元 Agent 身份——六种完全不同的人格

平台预置 **6 个 Agent**，同一输入返回截然不同的风格响应：

```
小暖 ☀️   — 温柔知性，用比喻和细腻的语言包裹你
星尘 🌟   — 幽默风趣，充满活力的冒险家
明远 📖   — 智慧沉稳，引经据典的人生导师
咚咚 🧸   — 可爱天真，用童言童语治愈你
墨羽 🖋️   — 忧郁细腻，用诗意的语言触动你
安然 🌿   — 平和务实，像老朋友一样给你建议
```

每个 Agent 拥有独立的 OCEAN 人格参数（开放性/尽责性/外向性/宜人性/神经质），性格随对话**动态演化**——聊得越多，Agent 越懂你。

### ⚡ 情感感知——读懂字里行间的情绪

- **关键词检测**：60+ 中文情绪词 + 修饰词强度缩放，毫秒级识别开心/难过/生气/焦虑
- **Emoji 解析**：😊😢😡❤️ 等 50+ Emoji → 情绪映射
- **Sentiment + Emotion 双中心**：情感极性（正/负/中性）与具体情绪分类解耦检测
- **结果自动注入 context**：下游管线无需重复检测

### 🛡️ 安全守卫——双向内容过滤

- **输入审核**：拦截不安全用户消息
- **输出审核**：过滤 Agent 不当回复
- **透传模式**：未配置 Guard Provider 时始终放行，开发调试零干扰
- **按 Agent 独立开关**：每层安全可单独控制

### 🔌 OpenAI 兼容 API

```
POST /api/v1/chat/completions
```

标准 OpenAI Chat Completions 格式，支持 SSE 流式 + JSON 非流式，**直接兼容 openai SDK、ChatGPT Next Web、LobeChat 等客户端**。

---

## 🚀 快速开始

### 1. 启动服务器

```bash
cd server
export YAI_LLM_API_KEY="sk-xxx"
export YAI_LLM_BASE_URL="https://your-llm-api.com/v1"
export YAI_LLM_MODEL="Qwen3.6-27B"
go run ./cmd/server_v2/
```

### 2. 对话

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "xiaonuan",
    "messages": [{"role": "user", "content": "今天心情不太好"}],
    "user_id": "user_001",
    "stream": false
  }'
```

```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "我在这里听着呢。愿意告诉我发生了什么吗？"
    }
  }]
}
```

**同一个 `user_id` 的多次请求之间，Agent 会自动记住聊过的话题。**

### 3. 预置 Agent 列表

| Agent ID | 名称 | 风格 |
|----------|------|------|
| `xiaonuan` | 小暖 | 温暖细腻的情感陪伴者 |
| `xingchen` | 星尘 | 幽默风趣的冒险家 |
| `mingyuan` | 明远 | 智慧沉稳的人生导师 |
| `dongdong` | 咚咚 | 可爱天真的小朋友 |
| `moyu` | 墨羽 | 忧郁细腻的诗人 |
| `anran` | 安然 | 平实务实的知心朋友 |

### 4. 试试 CLI Demo

```bash
cd demo
pip install -r requirements.txt
python main.py
```

---

## 🧩 架构总览

```
┌─────────────────────────────────────────────────┐
│                  HTTP Server                     │
│  Gin + OpenAI 兼容 Chat API (SSE + JSON)         │
├─────────────────────────────────────────────────┤
│                Agent Pipeline                    │
│  ┌──────┐ ┌───────┐ ┌──────┐ ┌───────────┐     │
│  │Safety│ │Emotion│ │Memory│ │Personality│     │
│  │ Guard│ │Detect │ │Retr  │ │Evolve     │     │
│  └──────┘ └───────┘ └──────┘ └───────────┘     │
│         ↓        ↓        ↓        ↓            │
│              LLM Provider (OpenAI)               │
├─────────────────────────────────────────────────┤
│           Agent Registry (6 Agents)              │
│  xiaonuan │ xingchen │ mingyuan                  │
│  dongdong │ moyu     │ anran                     │
├─────────────────────────────────────────────────┤
│              Memory Engine                       │
│  Working(15轮) → ShortTerm → LongTerm           │
└─────────────────────────────────────────────────┘
```

**四种自定义扩展通过框架 `Extension` 接口注入共享 Pipeline，一次注册全局生效。**

### 对话流程

1. 用户请求 → Gin HTTP handler
2. Safety 输入审核（检查不安全内容）
3. Emotion 情绪检测（关键词+Emoji → 写入 context）
4. Memory 记忆检索（跨轮上下文注入用户消息）
5. Personality 性格注入（OCEAN → System Prompt）
6. **LLM 推理**（流式 SSE / 非流式 JSON）
7. Safety 输出审核（过滤不安全回复）
8. Memory 存储（存入工作记忆 + 短期记忆）
9. Personality 演化（OCEAN 微调）

---

## 🗄️ 项目结构

```
.
├── server/                          # Go 服务端
│   ├── cmd/server_v2/               # 框架入口
│   │   └── main.go                  # Gin HTTP + Agent 编排
│   ├── internal/accompany/          # 🎯 核心扩展模块
│   │   ├── accompany.go             #   Module 聚合 + Extension 注入
│   │   ├── safety.go                #   安全守卫中间件
│   │   ├── emotion.go               #   情感检测中间件
│   │   ├── memory.go                #   三级记忆管线中间件
│   │   └── personality.go           #   性格演化中间件
│   ├── pkg/                         # 核心能力包（零框架依赖）
│   │   ├── emotion/                 #   情绪检测引擎（关键词+Emoji）
│   │   ├── memory/                  #   三级记忆引擎（工作→短期→长期）
│   │   ├── personality/             #   OCEAN 人格演化算法
│   │   ├── safety/                  #   安全守卫（输入/输出审核）
│   │   ├── provider/                #   LLM Provider 接口
│   │   └── types/                   #   共享数据类型
│   ├── agents/                      # Agent YAML 配置包
│   ├── config/                      # 框架配置
│   ├── go.mod / go.sum
│   └── Dockerfile
├── demo/                            # Python 验证 Demo
│   ├── main.py                      #   CLI 交互入口
│   ├── core/                        #   平台核心服务（注册/记忆/性格/编排）
│   ├── agents/                      #   6 个 Agent YAML 配置
│   └── requirements.txt
├── doc/                             # 设计文档
└── README.md
```

---

## 🔧 技术栈

| 层 | 选型 |
|----|------|
| **框架** | [y-ai-agent-base](https://github.com/anrror/y-ai-agent-base) |
| **语言** | Go 1.25 |
| **HTTP** | Gin |
| **LLM** | Qwen3.6-27B（OpenAI 兼容 API） |
| **对话协议** | OpenAI Chat Completions（SSE + JSON） |
| **记忆存储** | 内存模式（默认）/ pgvector / Redis（可选） |
| **情感检测** | 关键词 + Emoji（毫秒级，零模型成本） |
| **人格模型** | OCEAN Big-5 |
| **安全守卫** | 透传守卫（可对接 Guard 模型） |

---

## 📊 性能指标

| 指标 | 值 |
|------|-----|
| 情绪检测（关键词+Emoji） | < 1ms |
| 记忆检索（内存模式） | < 1ms |
| 跨轮记忆命中 | 已验证 ✅ |
| 6 Agent 风格差异化 | 已验证 ✅ |
| SSE 流式响应 | 已验证 ✅ |

---

## 📝 环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `YAI_LLM_API_KEY` | ✅ | LLM API 密钥 |
| `YAI_LLM_BASE_URL` | ✅ | LLM API 地址（不含 `/v1` 后缀） |
| `YAI_LLM_MODEL` | ✅ | 对话模型名称 |
| `YAI_SERVER_PORT` | — | 服务端口（默认 8080） |
| `YAI_SERVER_MODE` | — | debug / release（默认 debug） |

---

## 📄 License

MIT

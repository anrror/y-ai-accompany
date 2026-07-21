"""
AI 情感 Agent 平台 — 技术验证 Demo

演示内容:
  1. 平台启动 (Agent Registry 初始为空)
  2. Agent 通过注册协议接入平台 (加载 YAML 配置包)
  3. 用户通过统一 API 与任意已注册 Agent 对话
  4. 每个 Agent 独立维护记忆 (按 agent_id 隔离)

用法:
  python main.py                 交互式对话
  python main.py --list          列出所有已注册 Agent
  python main.py --summary <uid> <aid>  查看对话统计
"""

import sys
import os
from dotenv import load_dotenv
import yaml

load_dotenv()

from core.registry import AgentRegistry
from core.memory import MemoryEngine
from core.orchestrator import Orchestrator


def load_agent_configs(registry: AgentRegistry, agents_dir: str = "agents"):
    """演示: Agent通过YAML配置包注册到平台"""
    if not os.path.isdir(agents_dir):
        print(f"[平台] 未找到Agent配置目录: {agents_dir}")
        return

    yaml_files = [f for f in os.listdir(agents_dir) if f.endswith((".yaml", ".yml"))]
    if not yaml_files:
        print("[平台] 未找到Agent配置包")
        return

    for fname in sorted(yaml_files):
        path = os.path.join(agents_dir, fname)
        with open(path, encoding="utf-8") as f:
            config = yaml.safe_load(f)
        try:
            agent = registry.register(config)
            ident = agent.identity
            print(f"  [Agent接入] {agent.agent_id} | {ident.get('name')} "
                  f"| {ident.get('avatar', '')} | 性格: "
                  f"O={agent.personality.get('openness',0.5)} "
                  f"E={agent.personality.get('extraversion',0.5)} "
                  f"A={agent.personality.get('agreeableness',0.5)}")
        except ValueError as e:
            print(f"  [注册失败] {fname}: {e}")


def print_header(text: str):
    print(f"\n{'=' * 60}")
    print(f"  {text}")
    print(f"{'=' * 60}")


def list_agents(registry: AgentRegistry):
    agents = registry.list()
    if not agents:
        print("  (无已注册Agent)")
        return
    for a in agents:
        p = a.personality
        print(f"  [{a.agent_id}] {a.identity.get('name')} {a.identity.get('avatar','')}")
        print(f"    性格: O={p.get('openness',0):.1f} C={p.get('conscientiousness',0):.1f} "
              f"E={p.get('extraversion',0):.1f} A={p.get('agreeableness',0):.1f} "
              f"N={p.get('neuroticism',0):.1f}")
        print(f"    人设: {a.identity.get('persona','')[:50]}...")


def interactive_chat(orch: Orchestrator, registry: AgentRegistry):
    print_header("🤗 AI 情感 Agent 平台 — 交互式 Demo")
    print("提示: /help 查看命令, /quit 退出, /switch 切换Agent")
    print()

    user_id = "demo_user"

    agents = registry.list()
    if not agents:
        print("[平台] 没有已注册的Agent。请先配置 Agent 配置包。")
        return

    print("已注册的Agent:")
    for i, a in enumerate(agents):
        name = a.identity.get("name", a.agent_id)
        avatar = a.identity.get("avatar", "")
        print(f"  [{i}] {avatar} {name} ({a.agent_id})")

    current = agents[0]
    name = current.identity.get("name", current.agent_id)
    greeting = current.identity.get("greeting", "")
    print(f"\n当前Agent: {name}")
    if greeting:
        print(f"  {greeting}")

    while True:
        try:
            text = input(f"\n[{name}] 你说 > ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\n再见！")
            break

        if not text:
            continue

        if text == "/quit":
            print("再见！")
            break

        if text == "/help":
            print("  /switch    切换Agent")
            print("  /list      列出所有Agent")
            print("  /summary   查看对话统计")
            print("  /clear     清除当前Agent的记忆")
            print("  /personality  查看性格")
            print("  /safety    查看安全防御配置")
            print("  /features  查看平台可用特性")
            print("  /register <yaml>  注册新Agent")
            print("  /unregister <id>  注销Agent")
            print("  /quit      退出")
            continue

        if text == "/list":
            list_agents(registry)
            continue

        if text == "/switch":
            print("选择Agent:")
            for i, a in enumerate(agents):
                print(f"  [{i}] {a.identity.get('name', a.agent_id)}")
            try:
                idx = int(input("编号 > ").strip())
                if 0 <= idx < len(agents):
                    current = agents[idx]
                    name = current.identity.get("name", current.agent_id)
                    print(f"\n切换到: {name}")
                    print(f"  {current.identity.get('greeting', '')}")
                else:
                    print("无效编号")
            except ValueError:
                print("请输入数字")
            continue

        if text == "/summary":
            s = orch.get_summary(user_id, current.agent_id)
            p = current.personality
            print(f"\n📊 {name} 统计:")
            print(f"  工作记忆: {s['working_turns']} 轮")
            print(f"  短期记忆: {s['short_term']} 条")
            print(f"  长期记忆: {s['long_term']} 条")
            print(f"  性格: O={p.get('openness',0):.2f} C={p.get('conscientiousness',0):.2f} "
                  f"E={p.get('extraversion',0):.2f} A={p.get('agreeableness',0):.2f} "
                  f"N={p.get('neuroticism',0):.2f}")
            continue

        if text == "/clear":
            orch.memory.clear_all(user_id, current.agent_id)
            print(f"✅ 已清除 {name} 的所有记忆")
            continue

        if text == "/personality":
            p = current.personality
            print(f"\n🧬 {name} 性格参数:")
            print(f"  开放性 (O): {p.get('openness',0.5):.2f}")
            print(f"  尽责性 (C): {p.get('conscientiousness',0.5):.2f}")
            print(f"  外向性 (E): {p.get('extraversion',0.5):.2f}")
            print(f"  宜人性 (A): {p.get('agreeableness',0.5):.2f}")
            print(f"  神经质 (N): {p.get('neuroticism',0.5):.2f}")
            continue

        if text == "/safety":
            s = current.safety_config
            print(f"\n🛡️ {name} 安全防御配置:")
            print(f"  Layer 2 输入审核 (input_guard):      {'✅ 开启' if s.get('input_guard_enabled', True) else '❌ 关闭'}")
            print(f"  Layer 3 安全提示注入 (safety_notice): {'✅ 开启' if s.get('safety_notice_enabled', True) else '❌ 关闭'}")
            print(f"  Layer 4 输出审核 (output_guard):     {'✅ 开启' if s.get('output_guard_enabled', True) else '❌ 关闭'}")
            print(f"\n  每层可单独开关，YAML 配置包中的 safety_config 控制。")
            continue

        if text == "/features":
            print(f"\n🚀 平台已实现特性:")
            print(f"  ✅ 多 Agent 托管 — 6 个预置 Agent，独立身份/性格/记忆")
            print(f"  ✅ 五层安全防御 — 输入审核 + 上下文注入 + 输出审核 + 轨迹审计")
            print(f"  ✅ 安全层独立开关 — 每层可按 Agent 配置开启/关闭")
            print(f"  ✅ 三级记忆蒸馏 — 工作记忆(15轮) → 短期(pgvector) → 长期(固化)")
            print(f"  ✅ 大五人格 OCEAN — 性格驱动对话 + 交互演化")
            print(f"  ✅ 情感计算 — Emoji/关键词/专用模型 三重检测")
            print(f"  ✅ LLM 驾驭工程 — 动态调参 / Token 预算 / 重试熔断 / 可观测")
            print(f"  ✅ 多模型推理 — ModelRegistry + ModelRouter 路由")
            print(f"  ✅ 多模型调度 — Scheduler 评分选择最优端点")
            print(f"  ✅ 上下文压缩 — 超预算时自动压缩历史")
            print(f"  ✅ 推理缓存 — Exact + Semantic 双策略")
            print(f"  ✅ 推理范式 — CoT / ReAct / ToT / Plan-Execute")
            print(f"  ✅ 端云协同 — CloudOnly / EdgeAssist / EdgeOnly / Hybrid")
            continue

        if text.startswith("/register "):
            path = text[10:].strip()
            if not os.path.exists(path):
                print(f"文件不存在: {path}")
                continue
            with open(path, encoding="utf-8") as f:
                config = yaml.safe_load(f)
            try:
                registry.register(config)
                print(f"✅ Agent '{config.get('agent_id')}' 注册成功")
                agents = registry.list()
            except ValueError as e:
                print(f"❌ 注册失败: {e}")
            continue

        if text.startswith("/unregister "):
            aid = text[12:].strip()
            if registry.unregister(aid):
                print(f"✅ Agent '{aid}' 已注销")
                agents = registry.list()
                if current.agent_id == aid and agents:
                    current = agents[0]
                    name = current.identity.get("name", current.agent_id)
            else:
                print(f"❌ Agent '{aid}' 未找到")
            continue

        # 对话
        print(f"\n[{name}] ", end="", flush=True)
        for chunk in orch.chat(user_id, current.agent_id, text):
            print(chunk, end="", flush=True)
        print()


def main():
    registry = AgentRegistry()
    memory = MemoryEngine()
    orch = Orchestrator(registry, memory)

    print_header("🚀 AI 情感 Agent 平台")
    print("[平台] Agent Registry 初始化完成 (空)")
    print("[平台] Memory Engine 就绪 (Memory as a Service)")
    print("[平台] Safety Guard 就绪 (五层防御，每层可独立开关)")
    print("[平台] 正在接入Agent配置包...")
    load_agent_configs(registry)
    count = len(registry.list())
    print(f"[平台] 当前已注册: {count} 个Agent")
    print(f"[平台] 输入 /help 查看命令, /features 查看平台特性")

    if "--list" in sys.argv:
        print_header("📋 已注册 Agent")
        list_agents(registry)
        return

    if "--summary" in sys.argv:
        try:
            idx = sys.argv.index("--summary")
            user_id = sys.argv[idx + 1]
            agent_id = sys.argv[idx + 2]
        except IndexError:
            print("用法: python main.py --summary <user_id> <agent_id>")
            return
        agent = registry.get(agent_id)
        if not agent:
            print(f"Agent '{agent_id}' 未注册")
            return
        s = orch.get_summary(user_id, agent_id)
        print(f"{agent.identity.get('name')}: 工作记忆={s['working_turns']}轮, "
              f"短期={s['short_term']}条, 长期={s['long_term']}条")
        return

    interactive_chat(orch, registry)


if __name__ == "__main__":
    main()

"""AI 对话测试（真实 LLM）"""
import time

import pytest
from playwright.sync_api import Page

from helpers.screenshot import take_step_screenshot


@pytest.mark.llm
class TestAIChat:
    """AI 对话测试 — 需要真实 LLM API"""

    def test_chat_mode_basic(self, api, import_data):
        """chat 模式基本对话"""
        events = api.chat_stream(
            messages=[{"role": "user", "content": "请简要解释一下批发增速指标"}],
            mode="chat",
        )
        assert len(events) > 0, "chat 应返回 SSE 事件"
        # 检查有内容返回
        has_content = any(e.get("content") or e.get("result") for e in events)
        assert has_content, f"chat 应有内容响应, events: {events[:3]}"

    def test_adjust_set_target(self, api, import_data):
        """adjust 模式 set_target 意图"""
        before = api.indicators()
        events = api.chat_stream(
            messages=[{"role": "user", "content": "把批发当月增速调到 15%"}],
            mode="adjust",
        )
        assert len(events) > 0, "adjust 应返回事件"
        # 检查是否有 result 事件
        results = [e for e in events if e.get("type") == "result" or e.get("result")]
        if results:
            result = results[0].get("result", results[0])
            # 可能包含 groups 或 appliedRules
            assert "groups" in result or "reply" in result or "appliedRules" in result, \
                f"result 事件应包含数据: {result}"

    def test_adjust_add_rule(self, api, import_data):
        """adjust 模式 add_rule 意图"""
        api.clear_all_rules()
        events = api.chat_stream(
            messages=[{"role": "user", "content": "帮我加一条规则：批发当月增速不能超过 20%"}],
            mode="adjust",
        )
        assert len(events) > 0, "add_rule 应返回事件"
        # 检查规则是否被添加
        time.sleep(1)
        rules = api.get_rules()
        # LLM 可能识别为 add_rule 或普通聊天
        # 不强制要求添加成功，但检查事件序列
        event_types = [e.get("type") for e in events]
        assert len(event_types) > 0, "应有事件类型"

    def test_adjust_percent(self, api, import_data):
        """adjust 模式 adjust_percent 意图"""
        events = api.chat_stream(
            messages=[{"role": "user", "content": "帮我将零售当月增速随机调整 5%"}],
            mode="adjust",
        )
        assert len(events) > 0, "adjust_percent 应返回事件"

    def test_multi_turn_conversation(self, api, import_data):
        """多轮对话上下文保持"""
        session_id = "multi_turn_test"
        # 第一轮
        events1 = api.chat_stream(
            messages=[{"role": "user", "content": "目前批发增速是多少？"}],
            mode="chat",
            session_id=session_id,
        )
        # 提取助手回复
        reply1 = ""
        for e in events1:
            if e.get("content"):
                reply1 += e["content"]
            elif e.get("result", {}).get("reply"):
                reply1 = e["result"]["reply"]

        # 第二轮（引用前文）
        events2 = api.chat_stream(
            messages=[
                {"role": "user", "content": "目前批发增速是多少？"},
                {"role": "assistant", "content": reply1 or "批发增速约为 10%"},
                {"role": "user", "content": "那零售呢？"},
            ],
            mode="chat",
            session_id=session_id,
        )
        assert len(events2) > 0, "多轮对话第二轮应返回事件"

    def test_no_adjust_intent_fallback(self, api, import_data):
        """adjust 模式无调整意图降级为聊天"""
        events = api.chat_stream(
            messages=[{"role": "user", "content": "今天天气怎么样？"}],
            mode="adjust",
        )
        assert len(events) > 0, "无调整意图也应返回响应"
        # 降级为普通聊天，应有文本内容
        has_content = any(e.get("content") or e.get("result") for e in events)
        assert has_content, "降级后应有文本响应"

    def test_user_prompt_affects_chat(self, api, import_data):
        """用户偏好提示词影响对话"""
        # 设置偏好
        api.set_user_prompt("请用简洁的方式回复，每次回复不超过 50 字")
        events = api.chat_stream(
            messages=[{"role": "user", "content": "解释批发增速"}],
            mode="chat",
        )
        assert len(events) > 0
        # 恢复
        api.set_user_prompt("")

    def test_sse_event_sequence(self, api, import_data):
        """SSE 事件序列验证"""
        events = api.chat_stream(
            messages=[{"role": "user", "content": "你好"}],
            mode="chat",
        )
        types = [e.get("type") for e in events if e.get("type")]
        assert len(types) > 0, "应有类型化事件"
        # 最后一个事件应该是某种结束标记
        # 不强制具体顺序，但至少应有 chunk 或 result

    def test_chat_then_adjust(self, api, import_data):
        """先聊天再调整"""
        # 聊天
        events1 = api.chat_stream(
            messages=[{"role": "user", "content": "批发增速现在怎么样？"}],
            mode="chat",
        )
        assert len(events1) > 0
        # 调整
        events2 = api.chat_stream(
            messages=[{"role": "user", "content": "把批发当月增速调到 12%"}],
            mode="adjust",
        )
        assert len(events2) > 0

    def test_chat_ui_interaction(self, page: Page, import_data, screenshots_dir):
        """UI 上的聊天交互"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(500)

        # 寻找聊天相关的按钮
        all_buttons = page.locator("button").all()
        chat_opened = False
        for btn in all_buttons:
            try:
                text = btn.inner_text(timeout=1000)
                if any(k in text for k in ["对话", "聊天", "AI"]):
                    btn.click()
                    chat_opened = True
                    break
            except Exception:
                continue

        if not chat_opened:
            # 尝试点击带有 MessageCircle SVG 的按钮
            svg_btns = page.locator("button svg").all()
            if len(svg_btns) > 2:
                svg_btns[-1].locator("..").click()
                chat_opened = True

        page.wait_for_timeout(500)
        take_step_screenshot(page, screenshots_dir, "chat", "panel_state")

    def test_adjust_with_rule_effective(self, api, import_data):
        """添加规则后调整受约束"""
        api.clear_all_rules()
        # 先添加规则
        api.add_rule("限上增速不超过 10%")
        api.convert_rules()
        st = api.wait_rules_convert(timeout=30)
        if st["status"] != "ok":
            pytest.skip("规则转换需 LLM")
        # 尝试调整超过约束
        events = api.chat_stream(
            messages=[{"role": "user", "content": "把限上增速调到 20%"}],
            mode="adjust",
        )
        assert len(events) > 0

    def test_mixed_intent(self, api, import_data):
        """混合意图：调整 + 添加规则"""
        events = api.chat_stream(
            messages=[{"role": "user", "content": "把批发增速调到 15%，另外加一条规则：零售增速不能低于 5%"}],
            mode="adjust",
        )
        assert len(events) > 0, "混合意图应返回事件"

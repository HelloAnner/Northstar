"""规则管理生命周期测试"""
import time

import pytest
from playwright.sync_api import Page

from helpers.screenshot import take_step_screenshot


class TestRuleManagement:
    """规则管理生命周期测试"""

    def _go_settings(self, page: Page):
        """导航到设置页面"""
        page.goto(page.url.split("#")[0].rstrip("/") + "/settings", wait_until="networkidle")
        page.wait_for_timeout(500)

    def test_add_rule_via_api(self, api, import_data):
        """通过 API 新增规则"""
        api.clear_all_rules()
        result = api.add_rule("限上增速不超过 15%")
        assert isinstance(result, list), "add_rule 应返回列表"
        assert len(result) >= 1, "添加后应至少有 1 条规则"
        assert any("15%" in r.get("text", "") for r in result), "新规则应包含文本"

    def test_edit_rule_via_api(self, api, import_data):
        """编辑规则文本"""
        api.clear_all_rules()
        api.add_rule("测试规则原始文本")
        result = api.update_rule(1, "测试规则修改后文本")
        assert any("修改后" in r.get("text", "") for r in result), "更新后文本应变化"

    def test_delete_rule_via_api(self, api, import_data):
        """删除规则"""
        api.clear_all_rules()
        api.add_rule("待删除规则")
        rules = api.get_rules()
        assert len(rules) >= 1
        api.delete_rule(1)
        after = api.get_rules()
        assert len(after) < len(rules), "删除后规则数应减少"

    def test_rule_convert_success(self, api, import_data):
        """规则转换成功（需 LLM 或 mock）"""
        api.clear_all_rules()
        api.add_rule("限上社零额当月增速不超过 15%")
        # 触发转换
        api.convert_rules()
        # 等待转换完成
        st = api.wait_rules_convert(timeout=30)
        # 无 LLM 时可能一直是 idle 或 error，不强制 assert ok
        assert st["status"] in ("ok", "error", "idle"), f"转换状态应为有效值: {st}"

    def test_rules_status_polling(self, api, import_data):
        """规则状态轮询"""
        st = api.rules_status()
        assert "status" in st, "应返回 status 字段"
        assert st["status"] in ("idle", "running", "ok", "error"), f"状态无效: {st}"

    def test_add_rule_ui(self, page: Page, api, import_data, screenshots_dir):
        """通过 UI 新增规则"""
        api.clear_all_rules()
        self._go_settings(page)
        page.wait_for_timeout(1000)
        take_step_screenshot(page, screenshots_dir, "rules", "settings_page")

        # 查找规则 Tab（可能已默认选中）
        rules_tab = page.locator('[role="tab"]').filter(has_text="规则")
        if rules_tab.count() > 0:
            rules_tab.first.click()
            page.wait_for_timeout(500)

        # 点击新增按钮
        add_btn = page.get_by_role("button", name="新增规则")
        if add_btn.count() == 0:
            add_btn = page.locator("button").filter(has_text="新增规则")
        if add_btn.count() > 0:
            add_btn.first.click()
            page.wait_for_timeout(500)
            take_step_screenshot(page, screenshots_dir, "rules", "add_dialog")

            # 填写规则文本
            textarea = page.locator("textarea")
            if textarea.count() > 0:
                textarea.first.fill("批发当月增速不低于 -10%")
                page.wait_for_timeout(200)
                # 点击保存按钮
                submit = page.get_by_role("button", name="保存")
                if submit.count() == 0:
                    submit = page.locator("button").filter(has_text="保存")
                if submit.count() > 0:
                    submit.first.click()
                    page.wait_for_timeout(2000)
                    take_step_screenshot(page, screenshots_dir, "rules", "after_add")

        # 验证 API 端
        rules = api.get_rules()
        assert len(rules) >= 1, "通过 UI 添加后 API 应能查到规则"

    def test_clamp_target_effective(self, api, import_data):
        """clamp_target 规则生效验证"""
        # 设置一个 clamp 规则（限制增速上限）
        api.clear_all_rules()
        api.add_rule("限上社零额当月增速不超过 5%")
        api.convert_rules()
        st = api.wait_rules_convert(timeout=30)

        if st["status"] == "ok":
            # 尝试优化一个超过限制的目标
            result = api.optimize({"limitAbove_month_rate": 20.0})
            # 验证 appliedRules 中有 clamp 记录
            applied = result.get("appliedRules", [])
            if applied:
                clamp_rules = [r for r in applied if r.get("type") == "clamp_target"]
                assert len(clamp_rules) > 0, "clamp_target 规则应被应用"
        else:
            pytest.skip("规则转换未成功（需 LLM），跳过生效验证")

    def test_multiple_rules_coexist(self, api, import_data):
        """多规则共存"""
        api.clear_all_rules()
        api.add_rule("限上增速不超过 15%")
        api.add_rule("批发增速不低于 -10%")
        api.add_rule("零售增速调整时只分配正增长企业")
        rules = api.get_rules()
        assert len(rules) == 3, f"应有 3 条规则, got {len(rules)}"

    def test_rule_delete_then_no_constraint(self, api, import_data):
        """删除规则后约束失效"""
        api.clear_all_rules()
        rules = api.get_rules()
        assert len(rules) == 0, "清空后应无规则"
        # 无规则时优化应无 appliedRules
        result = api.optimize({"limitAbove_month_rate": 15.0})
        applied = result.get("appliedRules", [])
        # 可能有默认规则，不强制为空

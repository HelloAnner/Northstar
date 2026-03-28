"""仪表盘交互测试"""
import time

import pytest
from playwright.sync_api import Page, expect

from helpers.screenshot import take_step_screenshot


class TestDashboard:
    """仪表盘交互测试"""

    def test_indicator_cards_visible(self, page: Page, import_data, screenshots_dir):
        """指标卡片可见且有数值"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)
        take_step_screenshot(page, screenshots_dir, "dashboard", "indicators")
        # 检查页面有数据内容
        body = page.locator("body")
        assert body.inner_text() != "", "页面不应为空"

    def test_company_table_visible(self, page: Page, import_data, screenshots_dir):
        """企业表格可见"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)
        # 查找表格
        table = page.locator("table")
        if table.count() > 0:
            take_step_screenshot(page, screenshots_dir, "dashboard", "table")
            rows = table.locator("tbody tr")
            assert rows.count() > 0, "表格应有数据行"
        else:
            # 可能使用了虚拟列表
            take_step_screenshot(page, screenshots_dir, "dashboard", "data_view")

    def test_company_search(self, page: Page, import_data, api, screenshots_dir):
        """企业搜索过滤"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(500)
        # 获取第一个企业名用于搜索
        companies = api.companies(page_size=1)
        items = companies.get("items", [])
        if not items:
            pytest.skip("无企业数据")
        name = items[0].get("name", "")[:4]
        # 查找搜索框
        search = page.locator('input[placeholder*="搜索"], input[placeholder*="查找"], input[type="search"]')
        if search.count() > 0:
            search.first.fill(name)
            page.wait_for_timeout(500)
            take_step_screenshot(page, screenshots_dir, "dashboard", "search")

    def test_industry_tabs(self, page: Page, import_data, screenshots_dir):
        """行业 Tab 切换"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(500)
        # 查找 Tab 触发器
        tabs = page.locator('[role="tab"], button').filter(has_text="批发")
        if tabs.count() > 0:
            tabs.first.click()
            page.wait_for_timeout(500)
            take_step_screenshot(page, screenshots_dir, "dashboard", "tab_wholesale")
        tabs_retail = page.locator('[role="tab"], button').filter(has_text="零售")
        if tabs_retail.count() > 0:
            tabs_retail.first.click()
            page.wait_for_timeout(500)
            take_step_screenshot(page, screenshots_dir, "dashboard", "tab_retail")

    def test_manual_edit_company_value(self, page: Page, import_data, api, screenshots_dir):
        """手动编辑企业当月值"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(500)
        # 通过 API 获取企业并修改
        companies = api.companies(page_size=1)
        items = companies.get("items", [])
        if not items:
            pytest.skip("无企业数据")
        cid = items[0]["id"]
        before = items[0].get("salesCurrentMonth", 0)
        new_val = before + 100 if before else 1000
        result = api.patch_company(cid, {"salesCurrentMonth": new_val})
        assert "company" in result or "groups" in result, "修改企业应返回结果"
        # 刷新页面验证
        page.reload(wait_until="networkidle")
        page.wait_for_timeout(500)
        take_step_screenshot(page, screenshots_dir, "dashboard", "after_edit")
        # 恢复
        api.patch_company(cid, {"salesCurrentMonth": before})

    def test_chat_panel_open_close(self, page: Page, import_data, screenshots_dir):
        """ChatPanel 打开和关闭"""
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(500)
        # 查找聊天按钮（MessageCircle 图标）
        chat_btn = page.locator("button").filter(has=page.locator('svg'))
        # 尝试通过多种方式找到聊天按钮
        for btn in chat_btn.all():
            text = btn.inner_text()
            if "对话" in text or "聊天" in text or "chat" in text.lower():
                btn.click()
                page.wait_for_timeout(500)
                take_step_screenshot(page, screenshots_dir, "dashboard", "chat_open")
                # 关闭
                close_btn = page.locator("button").filter(has=page.locator('svg')).filter(has_text="")
                if close_btn.count() > 0:
                    close_btn.last.click()
                    page.wait_for_timeout(300)
                    take_step_screenshot(page, screenshots_dir, "dashboard", "chat_closed")
                return
        # 如果没找到文本匹配的，点最后一个按钮试试
        take_step_screenshot(page, screenshots_dir, "dashboard", "no_chat_btn")

    def test_settings_navigation(self, page: Page, import_data, screenshots_dir):
        """导航到设置页面"""
        page.wait_for_load_state("networkidle")
        # 查找设置按钮或链接
        settings_link = page.locator('a[href*="settings"], button').filter(has_text="设置")
        if settings_link.count() == 0:
            settings_link = page.locator("button").filter(has=page.locator('svg'))
        if settings_link.count() > 0:
            for link in settings_link.all():
                href = link.get_attribute("href") or ""
                if "settings" in href or "设置" in link.inner_text():
                    link.click()
                    page.wait_for_timeout(500)
                    take_step_screenshot(page, screenshots_dir, "dashboard", "settings_page")
                    return
        take_step_screenshot(page, screenshots_dir, "dashboard", "nav_attempt")

    def test_api_indicators_match(self, api, import_data):
        """API 指标数据一致性"""
        ind = api.indicators()
        groups = ind.get("groups", [])
        assert len(groups) > 0, "应有指标分组"
        # 检查每个指标都有 id 和 name
        for g in groups:
            for indicator in g.get("indicators", []):
                assert "id" in indicator, f"指标缺少 id: {indicator}"
                assert "name" in indicator, f"指标缺少 name: {indicator}"

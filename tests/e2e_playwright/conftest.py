"""pytest fixtures: 服务器、浏览器、页面、截图"""
import os
import time
from pathlib import Path

import pytest
from playwright.sync_api import sync_playwright

from helpers.api_client import ApiClient
from helpers.server import NorthstarServer

# 注册自定义报告插件
pytest_plugins = ["report_plugin"]

# ---- 常量 ----
E2E_PORT = int(os.environ.get("E2E_PORT", "18090"))
BASE_URL = f"http://localhost:{E2E_PORT}"
REPO_ROOT = Path(__file__).resolve().parents[2]
PRD_EXCEL = REPO_ROOT / "prd" / "12月月报（预估）_补全企业名称社会代码_20260129.xlsx"
TEMPLATE_EXCEL = REPO_ROOT / "prd" / "12月月报（定）.xlsx"
RESULTS_DIR = REPO_ROOT / "tests" / "e2e-result"


def has_llm_config() -> bool:
    key = os.environ.get("DEEPSEEK_API_KEY") or os.environ.get("LLM_API_KEY")
    base = os.environ.get("DEEPSEEK_BASE_URL") or os.environ.get("LLM_BASE_URL")
    return bool(key and base)


# ---- Session fixtures ----

@pytest.fixture(scope="session")
def results_dir() -> Path:
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    return RESULTS_DIR


@pytest.fixture(scope="session")
def screenshots_dir(results_dir) -> Path:
    d = results_dir / "screenshots"
    d.mkdir(parents=True, exist_ok=True)
    return d


@pytest.fixture(scope="session")
def northstar_server():
    """构建并启动 NorthStar 服务器（session 级别）"""
    server = NorthstarServer(port=E2E_PORT)
    server.build()
    server.start(timeout=30)
    llm_ok = server.configure_llm()
    yield server
    server.cleanup()


@pytest.fixture(scope="session")
def api(northstar_server) -> ApiClient:
    """API 客户端"""
    client = ApiClient(northstar_server.base_url)
    yield client
    client.close()


@pytest.fixture(scope="session")
def import_data(api) -> dict:
    """导入 PRD Excel 数据（session 级别，只导入一次）"""
    if not PRD_EXCEL.exists():
        pytest.skip(f"PRD Excel not found: {PRD_EXCEL}")
    events = api.import_excel(PRD_EXCEL)
    # 等待初始化完成
    for _ in range(10):
        st = api.status()
        if st.get("initialized"):
            break
        time.sleep(0.5)
    return {"events": events, "status": api.status()}


@pytest.fixture(scope="session")
def pw():
    """Playwright 实例（session 级别）"""
    with sync_playwright() as p:
        yield p


@pytest.fixture(scope="session")
def browser(pw):
    """浏览器实例（session 级别）"""
    b = pw.chromium.launch(headless=True)
    yield b
    b.close()


# ---- Function fixtures ----

@pytest.fixture
def context(browser):
    """每个测试独立的浏览器上下文"""
    ctx = browser.new_context(viewport={"width": 1440, "height": 900})
    yield ctx
    ctx.close()


@pytest.fixture
def page(request, context, northstar_server, screenshots_dir):
    """每个测试独立的页面，teardown 时自动截图"""
    p = context.new_page()
    p.goto(northstar_server.base_url, wait_until="networkidle")
    yield p
    # 关闭前自动截图
    try:
        if not p.is_closed():
            test_name = request.node.name.replace("[", "_").replace("]", "_").replace("/", "_")
            rep_call = getattr(request.node, "rep_call", None)
            status = "fail" if rep_call and rep_call.failed else "pass"
            path = screenshots_dir / ("%s_%s.png" % (test_name, status))
            p.screenshot(path=str(path), full_page=False)
            if not hasattr(request.node, "_screenshots"):
                request.node._screenshots = []
            request.node._screenshots.append(path)
    except Exception:
        pass
    p.close()


# ---- LLM skip ----

def pytest_collection_modifyitems(items):
    """无 LLM 配置时自动 skip 标记 llm 的测试"""
    if not has_llm_config():
        skip_marker = pytest.mark.skip(reason="LLM config not available (set DEEPSEEK_API_KEY)")
        for item in items:
            if "llm" in item.keywords:
                item.add_marker(skip_marker)


# ---- 辅助 fixtures ----

@pytest.fixture
def prd_excel() -> Path:
    """PRD Excel 文件路径"""
    if not PRD_EXCEL.exists():
        pytest.skip(f"PRD Excel not found: {PRD_EXCEL}")
    return PRD_EXCEL


@pytest.fixture
def template_excel() -> Path:
    """定稿模板 Excel 文件路径"""
    if not TEMPLATE_EXCEL.exists():
        pytest.skip(f"Template Excel not found: {TEMPLATE_EXCEL}")
    return TEMPLATE_EXCEL

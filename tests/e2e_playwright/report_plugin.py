"""自定义 pytest 插件：HTML 报告 + 截图收集"""
from __future__ import annotations

import base64
import time
from collections import defaultdict
from datetime import datetime
from pathlib import Path

import pytest

from report_template import GROUP_TEMPLATE, ITEM_TEMPLATE, REPORT_HTML


class TestResult:
    __slots__ = ("nodeid", "name", "doc", "status", "duration", "error", "screenshots", "group")

    def __init__(self, nodeid: str):
        self.nodeid = nodeid
        parts = nodeid.split("::")
        self.name = parts[-1] if parts else nodeid
        self.doc = ""
        self.status = "pass"
        self.duration = 0.0
        self.error = ""
        self.screenshots: list = []
        # 按文件分组
        self.group = parts[0].replace(".py", "").replace("test_", "") if parts else "unknown"


class ReportCollector:
    def __init__(self):
        self.results: dict = {}
        self.start_time = time.time()

    def get_or_create(self, nodeid: str) -> TestResult:
        if nodeid not in self.results:
            self.results[nodeid] = TestResult(nodeid)
        return self.results[nodeid]


_collector = ReportCollector()


@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item, call):
    """收集每个测试的结果 + 存储 rep_call 供 auto_screenshot 使用"""
    outcome = yield
    rep = outcome.get_result()
    # 存储 call 报告供 auto_screenshot fixture 使用
    if rep.when == "call":
        item.rep_call = rep

    result = _collector.get_or_create(item.nodeid)
    # 提取中文 docstring
    if call.when == "setup":
        fn = getattr(item, "function", None)
        if fn and fn.__doc__:
            result.doc = fn.__doc__.strip().split("\n")[0]
    if call.when == "call":
        result.duration = call.duration
        if call.excinfo:
            result.status = "fail"
            result.error = str(call.excinfo.getrepr(style="short"))
    elif call.when == "setup" and call.excinfo:
        result.status = "fail"
        result.error = str(call.excinfo.getrepr(style="short"))
    # 跳过
    if hasattr(call, "excinfo") and call.excinfo and call.excinfo.typename == "Skipped":
        result.status = "skip"
        result.error = str(call.excinfo.value)

    # 收集截图
    if call.when == "teardown":
        screenshots = getattr(item, "_screenshots", [])
        if screenshots:
            result.screenshots.extend(screenshots)


def pytest_sessionfinish(session, exitstatus):
    """渲染 HTML 报告"""
    report_path = session.config.getoption("--html-report", default=None)
    if not report_path:
        report_dir = Path(session.config.rootdir) / ".." / ".." / "tests" / "e2e-result"
        report_dir.mkdir(parents=True, exist_ok=True)
        report_path = str(report_dir / "report.html")

    total_duration = time.time() - _collector.start_time
    results = list(_collector.results.values())

    total = len(results)
    passed = sum(1 for r in results if r.status == "pass")
    failed = sum(1 for r in results if r.status == "fail")
    skipped = sum(1 for r in results if r.status == "skip")
    pass_rate = round(passed / total * 100, 1) if total > 0 else 0

    if pass_rate >= 90:
        rate_class = "good"
    elif pass_rate >= 70:
        rate_class = "warn"
    else:
        rate_class = "bad"

    # 按文件分组
    groups = defaultdict(list)
    for r in results:
        groups[r.group].append(r)

    groups_html = ""
    for group_name, items in sorted(groups.items()):
        g_passed = sum(1 for i in items if i.status == "pass")
        g_failed = sum(1 for i in items if i.status == "fail")
        g_skipped = sum(1 for i in items if i.status == "skip")

        badges = ""
        if g_passed:
            badges += '<span class="badge pass">%d passed</span>' % g_passed
        if g_failed:
            badges += '<span class="badge fail">%d failed</span>' % g_failed
        if g_skipped:
            badges += '<span class="badge skip">%d skipped</span>' % g_skipped

        items_html = ""
        for item in items:
            screenshots_html = ""
            if item.screenshots:
                thumbs = ""
                for sp in item.screenshots:
                    if Path(sp).exists():
                        b64 = base64.b64encode(Path(sp).read_bytes()).decode()
                        src = "data:image/png;base64," + b64
                        thumbs += '<img class="thumb" src="%s" onclick="showImg(this.src)" />' % src
                if thumbs:
                    screenshots_html = '<div class="screenshots">%s</div>' % thumbs

            error_html = ""
            if item.error and item.status == "fail":
                escaped = item.error.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
                error_html = '<div class="test-error">%s</div>' % escaped

            doc_html = '<div class="test-doc">%s</div>' % item.doc if item.doc else ""

            items_html += ITEM_TEMPLATE.substitute(
                status=item.status,
                name=item.name,
                doc_html=doc_html,
                duration="%.2fs" % item.duration,
                error_html=error_html,
                screenshots_html=screenshots_html,
            )

        groups_html += GROUP_TEMPLATE.substitute(
            group_name=group_name,
            badges=badges,
            items_html=items_html,
        )

    def fmt_duration(secs):
        m, s = divmod(int(secs), 60)
        return "%dm %ds" % (m, s) if m else "%ds" % s

    html = REPORT_HTML.substitute(
        timestamp=datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        duration=fmt_duration(total_duration),
        total=total,
        passed=passed,
        failed=failed,
        skipped=skipped,
        pass_rate=pass_rate,
        rate_class=rate_class,
        groups_html=groups_html,
    )

    Path(report_path).parent.mkdir(parents=True, exist_ok=True)
    Path(report_path).write_text(html, encoding="utf-8")
    print("\n" + "=" * 60)
    print("  HTML Report: %s" % report_path)
    print("  Total: %d | Passed: %d | Failed: %d | Skipped: %d" % (total, passed, failed, skipped))
    print("=" * 60)


def pytest_addoption(parser):
    parser.addoption("--html-report", default=None, help="HTML report output path")

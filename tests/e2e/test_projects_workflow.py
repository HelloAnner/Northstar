"""
Northstar E2E - 多项目管理工作流测试（specs/002）

目标：
- 新建/切换项目
- 在项目 A 中导入数据并持久化
- 切换到项目 B 验证数据隔离
- 切回项目 A 验证 state.json 可加载恢复
- 清理：删除测试项目
"""

import os
import time
import pytest
import requests


BASE_URL = "http://localhost:18080"
FIXTURES_DIR = os.path.join(os.path.dirname(__file__), "fixtures")


def wait_server_ready(session: requests.Session, timeout_sec: int = 30) -> None:
    for _ in range(timeout_sec):
        try:
            r = session.get(f"{BASE_URL}/api/v1/indicators")
            if r.status_code == 200:
                return
        except requests.ConnectionError:
            pass
        time.sleep(1)
    pytest.fail("服务器未能在规定时间内启动")


def extract_data(resp: requests.Response):
    payload = resp.json()
    return payload.get("data")


def ensure_ok(resp: requests.Response) -> None:
    assert resp.status_code == 200, resp.text
    payload = resp.json()
    assert payload.get("code") == 0, payload


@pytest.fixture(scope="module")
def session():
    s = requests.Session()
    wait_server_ready(s)
    yield s
    s.close()


class TestProjectsWorkflow:
    project_a = None
    project_b = None
    file_id = None

    def test_01_create_project_a(self, session):
        resp = session.post(
            f"{BASE_URL}/api/v1/projects",
            json={"name": "e2e_project_a"},
        )
        ensure_ok(resp)
        data = extract_data(resp)
        assert "projectId" in data
        assert data["hasData"] is False
        TestProjectsWorkflow.project_a = data["projectId"]

    def test_02_create_project_b(self, session):
        resp = session.post(
            f"{BASE_URL}/api/v1/projects",
            json={"name": "e2e_project_b"},
        )
        ensure_ok(resp)
        data = extract_data(resp)
        assert "projectId" in data
        TestProjectsWorkflow.project_b = data["projectId"]

    def test_03_select_project_a(self, session):
        assert TestProjectsWorkflow.project_a is not None
        resp = session.post(f"{BASE_URL}/api/v1/projects/{TestProjectsWorkflow.project_a}/select")
        ensure_ok(resp)

        cur = session.get(f"{BASE_URL}/api/v1/projects/current")
        ensure_ok(cur)
        cur_data = extract_data(cur)
        assert cur_data["project"]["projectId"] == TestProjectsWorkflow.project_a

    def test_04_import_into_project_a(self, session):
        test_file = os.path.join(FIXTURES_DIR, "test_companies_300.xlsx")
        assert os.path.exists(test_file), f"缺少测试文件: {test_file}"

        with open(test_file, "rb") as f:
            files = {
                "file": (
                    "test_companies_300.xlsx",
                    f,
                    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                )
            }
            resp = session.post(f"{BASE_URL}/api/v1/import/upload", files=files)
        ensure_ok(resp)
        TestProjectsWorkflow.file_id = extract_data(resp)["fileId"]

        mapping = {
            "companyName": "企业名称",
            "creditCode": "统一社会信用代码",
            "industryCode": "行业代码",
            "companyScale": "企业规模",
            "retailCurrentMonth": "本期零售额",
            "retailLastYearMonth": "上年同期零售额",
            "retailCurrentCumulative": "本期累计零售额",
            "retailLastYearCumulative": "上年累计零售额",
            "salesCurrentMonth": "本期销售额",
            "salesLastYearMonth": "上年同期销售额",
            "salesCurrentCumulative": "本期累计销售额",
            "salesLastYearCumulative": "上年累计销售额",
        }
        resp = session.post(
            f"{BASE_URL}/api/v1/import/{TestProjectsWorkflow.file_id}/mapping",
            json={"sheet": "企业数据", "mapping": mapping},
        )
        ensure_ok(resp)

        resp = session.post(
            f"{BASE_URL}/api/v1/import/{TestProjectsWorkflow.file_id}/execute",
            json={"sheet": "企业数据", "generateHistory": False, "currentMonth": 6},
        )
        ensure_ok(resp)

        # 验证项目 A 已就绪（hasData=true）且企业数量 > 0
        detail = session.get(f"{BASE_URL}/api/v1/projects/{TestProjectsWorkflow.project_a}")
        ensure_ok(detail)
        d = extract_data(detail)
        assert d["project"]["hasData"] is True
        assert d["project"].get("companyCount", 0) > 0

    def test_04b_adjust_indicator_in_project_a(self, session):
        """在项目 A 中调整指标，验证 /indicators/adjust 生效并联动刷新"""
        target_rate = 0.05  # 目标 5%

        resp = session.post(
            f"{BASE_URL}/api/v1/indicators/adjust",
            json={"key": "limitAboveMonthRate", "value": target_rate},
        )
        ensure_ok(resp)
        indicators = extract_data(resp)
        assert abs(indicators["limitAboveMonthRate"] - target_rate) < 0.002

        # 二次读取确认落地
        resp = session.get(f"{BASE_URL}/api/v1/indicators")
        ensure_ok(resp)
        indicators = extract_data(resp)
        assert abs(indicators["limitAboveMonthRate"] - target_rate) < 0.002

    def test_05_switch_to_project_b_isolated(self, session):
        assert TestProjectsWorkflow.project_b is not None
        resp = session.post(f"{BASE_URL}/api/v1/projects/{TestProjectsWorkflow.project_b}/select")
        ensure_ok(resp)

        # 项目 B 初始应为空
        companies = session.get(f"{BASE_URL}/api/v1/companies", params={"page": 1, "pageSize": 10})
        ensure_ok(companies)
        data = extract_data(companies)
        assert data["total"] == 0

    def test_06_switch_back_project_a_restored(self, session):
        resp = session.post(f"{BASE_URL}/api/v1/projects/{TestProjectsWorkflow.project_a}/select")
        ensure_ok(resp)

        companies = session.get(f"{BASE_URL}/api/v1/companies", params={"page": 1, "pageSize": 10})
        ensure_ok(companies)
        data = extract_data(companies)
        assert data["total"] > 0

    def test_07_delete_projects_cleanup(self, session):
        # 删除项目需要后端提供 DELETE /projects/:projectId
        for pid in [TestProjectsWorkflow.project_a, TestProjectsWorkflow.project_b]:
            if not pid:
                continue
            resp = session.delete(f"{BASE_URL}/api/v1/projects/{pid}")
            ensure_ok(resp)

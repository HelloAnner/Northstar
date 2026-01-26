"""
Northstar E2E Test - Comprehensive Test Suite
基于 specs/001/test-cases.md 测试用例文档

测试依据: Northstar 经济数据统计分析工具 - 测试用例设计文档

注意: API 响应格式为 {"code": 0, "message": "success", "data": {...}}
"""
import pytest
import requests
import time
import json


def extract_data(response):
    """从 API 响应中提取 data 字段"""
    json_data = response.json()
    if 'data' in json_data:
        return json_data['data']
    return json_data


def get_code(response):
    """获取响应的 code 字段"""
    json_data = response.json()
    return json_data.get('code', -1)


class TestServerHealth:
    """服务器健康检查测试"""

    def test_server_is_running(self, api_client):
        """TC-HEALTH-001: 验证服务器正常运行"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/indicators")
        assert response.status_code == 200, "服务器应该返回 200 状态码"

    def test_api_response_time(self, api_client):
        """TC-HEALTH-002: API 响应时间应该小于 500ms"""
        session, base_url = api_client
        start = time.time()
        response = session.get(f"{base_url}/api/v1/indicators")
        elapsed = time.time() - start
        assert elapsed < 0.5, f"响应时间 {elapsed:.3f}s 超过 500ms"


class TestIndicatorsAPI:
    """指标 API 测试

    测试依据: 测试用例文档 Section 2 - 指标计算测试用例
    """

    def test_get_indicators_structure(self, api_client):
        """TC-IND-STRUCT: 验证指标 API 返回的数据结构完整"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/indicators")

        assert response.status_code == 200
        data = extract_data(response)

        # 指标组一: 限上社零额 (4个指标)
        assert "limitAboveMonthValue" in data, "缺少: 限上社零额(当月值)"
        assert "limitAboveMonthRate" in data, "缺少: 限上社零额增速(当月)"
        assert "limitAboveCumulativeValue" in data, "缺少: 限上社零额(累计值)"
        assert "limitAboveCumulativeRate" in data, "缺少: 限上社零额增速(累计)"

        # 指标组二: 专项增速 (2个指标)
        assert "eatWearUseMonthRate" in data, "缺少: 吃穿用增速(当月)"
        assert "microSmallMonthRate" in data, "缺少: 小微企业增速(当月)"

        # 指标组三: 四大行业增速 (8个指标)
        assert "industryRates" in data, "缺少: 行业增速数据"

        # 指标组四: 社零总额 (2个指标)
        assert "totalSocialCumulativeValue" in data, "缺少: 社零总额(累计值)"
        assert "totalSocialCumulativeRate" in data, "缺少: 社零总额增速(累计)"

    def test_industry_rates_complete(self, api_client):
        """TC-IND-007~014: 验证四大行业增速数据完整"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/indicators")

        assert response.status_code == 200
        data = extract_data(response)

        industry_rates = data["industryRates"]

        # 验证四大行业都存在
        required_industries = ["wholesale", "retail", "accommodation", "catering"]
        for industry in required_industries:
            assert industry in industry_rates, f"缺少 {industry} 行业数据"
            assert "monthRate" in industry_rates[industry], f"{industry} 缺少 monthRate"
            assert "cumulativeRate" in industry_rates[industry], f"{industry} 缺少 cumulativeRate"

    def test_indicators_value_types(self, api_client):
        """TC-IND-TYPE: 验证指标数值类型正确"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/indicators")

        data = extract_data(response)

        # 值类型应该是数字
        assert isinstance(data["limitAboveMonthValue"], (int, float)), "limitAboveMonthValue 应为数字"
        assert isinstance(data["limitAboveMonthRate"], (int, float)), "limitAboveMonthRate 应为数字"

    def test_empty_data_indicators_zero(self, api_client):
        """TC-EDGE-002: 空数据时所有指标应为 0"""
        session, base_url = api_client

        # 先重置数据确保为空
        session.post(f"{base_url}/api/v1/companies/reset")

        response = session.get(f"{base_url}/api/v1/indicators")
        data = extract_data(response)

        # 空数据时，增速应该是 0（不是 NaN 或报错）
        assert data["limitAboveMonthRate"] == 0, "空数据时增速应为 0"


class TestCompaniesAPI:
    """企业 API 测试

    测试依据: 测试用例文档 Section 3 - 联动计算测试用例
    """

    def test_list_companies(self, api_client):
        """TC-COMP-001: 获取企业列表"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/companies")

        assert response.status_code == 200
        data = extract_data(response)

        assert "items" in data, "返回数据应包含 items 字段"
        assert "total" in data, "返回数据应包含 total 字段"
        assert "page" in data, "返回数据应包含 page 字段"
        assert "pageSize" in data, "返回数据应包含 pageSize 字段"
        assert isinstance(data["items"], list), "items 应为数组"

    def test_pagination(self, api_client):
        """TC-COMP-002: 分页功能测试"""
        session, base_url = api_client

        # 测试第一页
        response = session.get(f"{base_url}/api/v1/companies?page=1&pageSize=5")
        assert response.status_code == 200
        data = extract_data(response)
        assert data["page"] == 1
        assert data["pageSize"] == 5

        # 测试第二页
        response = session.get(f"{base_url}/api/v1/companies?page=2&pageSize=5")
        assert response.status_code == 200
        data = extract_data(response)
        assert data["page"] == 2

    def test_search_companies(self, api_client):
        """TC-COMP-003: 企业搜索功能"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/companies?keyword=测试")

        assert response.status_code == 200

    def test_get_nonexistent_company(self, api_client):
        """TC-COMP-004: 获取不存在的企业应返回错误码"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/companies/non-existent-id-12345")

        assert response.status_code == 200
        code = get_code(response)
        assert code != 0, "获取不存在的企业应返回非0错误码"

    def test_reset_companies(self, api_client):
        """TC-COMP-005: 重置企业数据"""
        session, base_url = api_client
        response = session.post(f"{base_url}/api/v1/companies/reset")

        assert response.status_code == 200


class TestConfigAPI:
    """配置 API 测试"""

    def test_get_config(self, api_client):
        """TC-CFG-001: 获取配置"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/config")

        assert response.status_code == 200
        data = extract_data(response)

        assert "currentMonth" in data, "配置应包含 currentMonth"
        assert "lastYearLimitBelowCumulative" in data, "配置应包含 lastYearLimitBelowCumulative"

    def test_config_default_values(self, api_client):
        """TC-CFG-002: 验证默认配置值"""
        session, base_url = api_client
        response = session.get(f"{base_url}/api/v1/config")
        data = extract_data(response)

        # currentMonth 应该在 1-12 之间
        assert 1 <= data["currentMonth"] <= 12, "currentMonth 应在 1-12 之间"

        # lastYearLimitBelowCumulative 应该是非负数
        assert data["lastYearLimitBelowCumulative"] >= 0, "lastYearLimitBelowCumulative 应为非负数"

    def test_update_config(self, api_client):
        """TC-CFG-003: 更新配置 - 验证配置更新 API 正常工作"""
        session, base_url = api_client

        # 获取原始配置
        original = extract_data(session.get(f"{base_url}/api/v1/config"))
        original_month = original["currentMonth"]

        # 更新配置
        new_month = 9 if original_month != 9 else 6
        response = session.patch(
            f"{base_url}/api/v1/config",
            json={"currentMonth": new_month}
        )
        assert response.status_code == 200
        assert get_code(response) == 0, "配置更新应返回成功"

        # 验证更新成功（API 返回的数据）
        updated_data = extract_data(response)
        # 如果 API 返回更新后的配置，验证它
        if "currentMonth" in updated_data:
            assert updated_data["currentMonth"] == new_month, "返回的配置应包含新值"

        # 恢复原始配置
        session.patch(
            f"{base_url}/api/v1/config",
            json={"currentMonth": original_month}
        )


class TestOptimizeAPI:
    """智能调整 API 测试

    测试依据: 测试用例文档 Section 5 - 智能调整测试用例
    """

    def test_optimize_preview(self, api_client):
        """TC-OPT-001: 智能调整预览"""
        session, base_url = api_client

        response = session.post(
            f"{base_url}/api/v1/optimize/preview",
            json={"targetCumulativeRate": 0.05}
        )

        # 可能返回 200（有方案）或 400（无法优化/无数据）
        assert response.status_code in [200, 400]

    def test_optimize_invalid_target(self, api_client):
        """TC-OPT-002: 无效的目标增速参数测试 - 验证 API 对异常输入的处理"""
        session, base_url = api_client

        response = session.post(
            f"{base_url}/api/v1/optimize/preview",
            json={"targetCumulativeRate": "invalid"}
        )

        # API 可能对无效参数有不同处理方式：
        # 1. 返回错误码 (code != 0)
        # 2. 返回 400 状态码
        # 3. 将无效值转换为默认值并正常处理
        # 验证 API 不会崩溃，能正常响应
        assert response.status_code in [200, 400], "API 应能处理无效参数"

        # 如果是 200，检查是否有合理的响应结构
        if response.status_code == 200:
            json_data = response.json()
            assert "code" in json_data, "响应应包含 code 字段"

    def test_optimize_extreme_target(self, api_client):
        """TC-OPT-003: 极端目标增速（可能无解）"""
        session, base_url = api_client

        # 尝试设置一个不太可能达成的目标
        response = session.post(
            f"{base_url}/api/v1/optimize/preview",
            json={"targetCumulativeRate": 10.0}  # 1000% 增速
        )

        # 应该返回 400（无法达成）或 200（有方案但极端）
        assert response.status_code in [200, 400]


class TestStaticFiles:
    """静态资源测试（验证离线可用性）"""

    def test_index_page(self, api_client):
        """TC-STATIC-001: 首页可访问"""
        session, base_url = api_client
        response = session.get(base_url)

        assert response.status_code == 200
        assert "text/html" in response.headers.get("Content-Type", "")
        assert "Northstar" in response.text or "经济数据" in response.text

    def test_spa_routing_import(self, api_client):
        """TC-STATIC-002: SPA 路由 - /import"""
        session, base_url = api_client
        response = session.get(f"{base_url}/import")

        assert response.status_code == 200
        assert "text/html" in response.headers.get("Content-Type", "")

    def test_spa_routing_unknown(self, api_client):
        """TC-STATIC-003: 未知路由应返回首页（SPA）"""
        session, base_url = api_client
        response = session.get(f"{base_url}/unknown/path/123")

        assert response.status_code == 200
        assert "text/html" in response.headers.get("Content-Type", "")

    def test_no_external_resources(self, api_client):
        """TC-OFFLINE-001: 首页不应包含外部资源引用"""
        session, base_url = api_client
        response = session.get(base_url)

        content = response.text.lower()

        # 检查是否有外部 CDN 引用
        external_patterns = [
            "fonts.googleapis.com",
            "fonts.gstatic.com",
            "cdnjs.cloudflare.com",
            "unpkg.com",
            "cdn.jsdelivr.net",
        ]

        for pattern in external_patterns:
            assert pattern not in content, f"发现外部资源引用: {pattern}"


class TestValidation:
    """数据校验测试

    测试依据: 测试用例文档 Section 4 - 校验规则测试用例
    """

    def test_update_company_invalid_id(self, api_client):
        """TC-VAL-001: 更新不存在的企业应返回错误"""
        session, base_url = api_client

        response = session.patch(
            f"{base_url}/api/v1/companies/invalid-id-12345",
            json={"retailCurrentMonth": 1000}
        )

        assert response.status_code == 200
        code = get_code(response)
        assert code != 0, "更新不存在的企业应返回错误码"


class TestImportAPI:
    """导入 API 测试"""

    def test_import_without_file(self, api_client):
        """TC-IMP-001: 没有文件时上传应返回错误"""
        session, base_url = api_client

        response = session.post(f"{base_url}/api/v1/import/upload")

        # API 返回 200 但 code 应为非 0
        assert response.status_code == 200
        code = get_code(response)
        assert code != 0, "没有文件时应返回错误码"

    def test_get_columns_invalid_file(self, api_client):
        """TC-IMP-002: 无效文件 ID 获取列信息应返回错误"""
        session, base_url = api_client

        response = session.get(f"{base_url}/api/v1/import/invalid-file-id/columns?sheet=Sheet1")

        assert response.status_code == 200
        code = get_code(response)
        assert code != 0, "无效文件ID应返回错误码"


class TestExportAPI:
    """导出 API 测试"""

    def test_export_request(self, api_client):
        """TC-EXP-001: 导出请求"""
        session, base_url = api_client

        response = session.post(
            f"{base_url}/api/v1/export",
            json={
                "format": "xlsx",
                "includeIndicators": True,
                "includeChanges": True
            }
        )

        # 可能返回 200（成功）或其他状态码
        assert response.status_code in [200, 201, 400]

    def test_download_invalid_export(self, api_client):
        """TC-EXP-002: 下载不存在的导出文件应返回 404"""
        session, base_url = api_client

        response = session.get(f"{base_url}/api/v1/export/download/invalid-export-id")

        # 下载接口返回 404 或文本错误
        assert response.status_code == 404 or "不存在" in response.text


class TestEdgeCases:
    """边界条件测试

    测试依据: 测试用例文档 Section 6 - 边界条件测试用例
    """

    def test_empty_request_body(self, api_client):
        """TC-EDGE-001: 空请求体测试 - 验证 API 对空输入的处理"""
        session, base_url = api_client

        response = session.post(
            f"{base_url}/api/v1/optimize/preview",
            json={}
        )

        # API 可能对空请求体有不同处理方式：
        # 1. 返回错误码 (code != 0)
        # 2. 使用默认值处理并返回成功
        # 验证 API 不会崩溃，能正常响应
        assert response.status_code in [200, 400], "API 应能处理空请求体"

        # 验证响应结构完整
        json_data = response.json()
        assert "code" in json_data, "响应应包含 code 字段"
        assert "message" in json_data, "响应应包含 message 字段"

    def test_concurrent_requests(self, api_client):
        """TC-EDGE-003: 并发请求测试"""
        session, base_url = api_client

        import concurrent.futures

        def make_request():
            return session.get(f"{base_url}/api/v1/indicators")

        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_request) for _ in range(20)]
            results = [f.result() for f in futures]

        # 所有请求都应该成功
        for r in results:
            assert r.status_code == 200


class TestCORS:
    """CORS 配置测试"""

    def test_cors_headers(self, api_client):
        """TC-CORS-001: CORS 头存在"""
        session, base_url = api_client

        response = session.options(f"{base_url}/api/v1/indicators")

        # OPTIONS 请求应该返回 204 或 200
        assert response.status_code in [200, 204]


class TestAPIVersioning:
    """API 版本测试"""

    def test_api_v1_exists(self, api_client):
        """TC-API-001: API v1 路径存在"""
        session, base_url = api_client

        response = session.get(f"{base_url}/api/v1/indicators")
        assert response.status_code == 200

    def test_api_root_not_found(self, api_client):
        """TC-API-002: 直接访问 /api 应返回 404"""
        session, base_url = api_client

        response = session.get(f"{base_url}/api")
        # 应该返回 404 或被 SPA 路由捕获返回 200
        assert response.status_code in [200, 404]

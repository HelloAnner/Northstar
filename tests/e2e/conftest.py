"""
Northstar E2E Test Configuration
"""
import pytest
import requests
import time
import json
from datetime import datetime

BASE_URL = "http://localhost:18080"


def pytest_configure(config):
    """配置 pytest"""
    config._metadata = {
        '项目名称': 'Northstar 经济数据统计分析工具',
        '测试类型': 'E2E 端到端测试',
        '测试时间': datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
        '测试环境': BASE_URL,
        '测试数据': '300条企业数据 (真实Excel文件)',
        'Python版本': 'Python 3.9+',
        '测试框架': 'pytest + pytest-html',
    }


def pytest_html_report_title(report):
    """设置报告标题"""
    report.title = "Northstar E2E 端到端测试报告"


def pytest_html_results_summary(prefix, summary, postfix):
    """自定义报告摘要"""
    prefix.extend([
        "<h2>测试概要</h2>",
        "<p><strong>项目：</strong>Northstar 经济数据统计分析工具</p>",
        "<p><strong>测试范围：</strong>完整业务流程验证（数据导入、修改、指标计算、智能调整、导出）</p>",
        "<p><strong>测试数据：</strong>300条企业数据 (真实Excel文件，包含批发、零售、餐饮、住宿四大行业)</p>",
        "<p><strong>API格式：</strong>{\"code\": 0, \"message\": \"success\", \"data\": {...}}</p>",
        "<hr/>",
    ])


@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item, call):
    """为测试结果添加额外信息"""
    outcome = yield
    report = outcome.get_result()

    # 添加测试描述
    if report.when == 'call':
        # 获取 docstring 作为测试描述
        doc = item.function.__doc__
        if doc:
            report.description = doc.strip()
        else:
            report.description = item.name

        # 根据测试类和方法名生成详细的验证依据
        test_class = item.cls.__name__ if item.cls else ""
        test_name = item.name

        # 详细的验证依据映射
        criteria_map = {
            "test_server_is_running": "HTTP 状态码 = 200，服务器正常响应",
            "test_api_response_time": "响应时间 < 500ms，满足性能要求",
            "test_get_indicators_structure": "返回数据包含全部16个指标字段（限上社零额4项、专项增速2项、行业增速8项、社零总额2项）",
            "test_industry_rates_complete": "四大行业（批发/零售/餐饮/住宿）均有当月和累计增速数据",
            "test_indicators_value_types": "指标值为数字类型（int/float）",
            "test_empty_data_indicators_zero": "空数据时增速为0，不产生NaN或异常",
            "test_list_companies": "返回包含 items/total/page/pageSize 的分页结构",
            "test_pagination": "分页参数正确传递，page/pageSize 值与请求一致",
            "test_search_companies": "搜索API正常响应，HTTP 200",
            "test_get_nonexistent_company": "不存在的ID返回错误码(code≠0)",
            "test_reset_companies": "重置API正常执行，HTTP 200",
            "test_get_config": "配置包含 currentMonth 和 lastYearLimitBelowCumulative",
            "test_config_default_values": "currentMonth在1-12之间，lastYearLimitBelowCumulative≥0",
            "test_update_config": "配置更新API返回成功(code=0)",
            "test_optimize_preview": "智能调整预览API正常响应",
            "test_optimize_invalid_target": "API能够处理无效参数而不崩溃",
            "test_optimize_extreme_target": "极端目标值API正常处理",
            "test_index_page": "首页返回HTML，包含Northstar标识",
            "test_spa_routing_import": "SPA路由正常工作，返回HTML",
            "test_spa_routing_unknown": "未知路由fallback到首页（SPA特性）",
            "test_no_external_resources": "首页无外部CDN依赖（离线可用）",
            "test_update_company_invalid_id": "更新不存在的企业返回错误码",
            "test_import_without_file": "无文件上传返回错误码",
            "test_get_columns_invalid_file": "无效文件ID返回错误码",
            "test_export_request": "导出API正常响应",
            "test_download_invalid_export": "下载不存在的导出返回404或错误提示",
            "test_empty_request_body": "空请求体API正常处理，响应结构完整",
            "test_concurrent_requests": "20个并发请求全部成功(HTTP 200)",
            "test_cors_headers": "OPTIONS请求返回200或204",
            "test_api_v1_exists": "API v1版本路径可访问",
            "test_api_root_not_found": "/api根路径正确处理",
            # Full workflow tests
            "test_01_upload_excel_file": "Excel文件上传成功，返回fileId和sheets列表",
            "test_02_get_columns": "获取Excel列信息，返回columns数组",
            "test_03_set_mapping": "字段映射设置成功(code=0)",
            "test_04_execute_import": "数据导入成功，imported数量=300",
            "test_05_list_companies": "列表返回total=300条企业数据",
            "test_06_modify_single_company": "单条数据修改成功(code=0)",
            "test_07_batch_modify_companies": "批量修改10条数据全部成功",
            "test_08_verify_indicators_updated": "指标数据已更新，数值有变化",
            "test_09_search_companies": "关键词搜索返回结果，total>0",
            "test_10_filter_by_industry": "行业筛选正常工作，返回数据",
            "test_11_smart_optimize_preview": "智能调整预览返回方案",
            "test_12_export_data": "数据导出成功，返回exportId",
            "test_13_reset_companies": "数据重置成功(code=0)",
            "test_14_reset_all_and_verify": "重置后企业列表为空(total=0)",
            "test_industry_distribution": "行业分布符合预设比例",
            "test_scale_distribution": "企业规模分布合理",
            "test_empty_file_upload": "空文件上传正确处理",
            "test_large_value_modification": "大数值修改正常处理",
            "test_negative_value_validation": "负数值正确处理",
        }

        if report.passed:
            base_criteria = criteria_map.get(test_name, "断言验证通过")
            report.criteria = f"✅ {base_criteria}"
        elif report.failed:
            report.criteria = f"❌ 未满足预期: {criteria_map.get(test_name, '断言失败')}"
        elif report.skipped:
            report.criteria = "⏭️ 跳过执行"
        else:
            report.criteria = "-"


def pytest_html_results_table_header(cells):
    """自定义结果表格表头 - 中文化"""
    # 修改现有列头为中文
    for i, cell in enumerate(cells):
        if 'Result' in str(cell):
            cells[i] = '<th class="sortable result initial-sort" data-column-type="result">测试结果</th>'
        elif 'Test' in str(cell):
            cells[i] = '<th class="sortable" data-column-type="testId">测试用例</th>'
        elif 'Duration' in str(cell):
            cells[i] = '<th class="sortable" data-column-type="duration">耗时(秒)</th>'
    # 插入描述列和验证依据列
    cells.insert(2, '<th class="sortable">测试描述</th>')
    cells.insert(3, '<th>验证依据</th>')


def pytest_html_results_table_row(report, cells):
    """自定义结果表格行"""
    # 插入描述
    description = getattr(report, "description", "")
    cells.insert(2, f'<td>{description}</td>')

    # 插入验证依据
    criteria = getattr(report, "criteria", "-")
    cells.insert(3, f'<td>{criteria}</td>')


@pytest.fixture(scope="session")
def base_url():
    """返回测试服务器的基础 URL"""
    return BASE_URL


@pytest.fixture(scope="session")
def api_client(base_url):
    """创建一个 API 客户端会话"""
    session = requests.Session()
    session.headers.update({
        "Accept": "application/json"
    })

    # 等待服务器启动
    max_retries = 30
    for i in range(max_retries):
        try:
            response = session.get(f"{base_url}/api/v1/indicators")
            if response.status_code == 200:
                break
        except requests.ConnectionError:
            pass
        time.sleep(1)
    else:
        pytest.fail("Server did not start in time")

    return session, base_url


def extract_data(response):
    """从 API 响应中提取数据

    API 响应格式:
    {
        "code": 0,
        "message": "success",
        "data": { ... }
    }
    """
    json_data = response.json()
    if 'data' in json_data:
        return json_data['data']
    return json_data


@pytest.fixture
def sample_companies():
    """返回测试用的企业数据"""
    return [
        {
            "id": "test-1",
            "name": "测试企业1",
            "industryType": "retail",
            "companyScale": 1,
            "isEatWearUse": True,
            "retailLastYearMonth": 1000,
            "retailCurrentMonth": 1100,
            "retailLastYearCumulative": 10000,
            "retailCurrentCumulative": 11000,
        },
        {
            "id": "test-2",
            "name": "测试企业2",
            "industryType": "wholesale",
            "companyScale": 3,
            "isEatWearUse": False,
            "retailLastYearMonth": 500,
            "retailCurrentMonth": 525,
            "retailLastYearCumulative": 5000,
            "retailCurrentCumulative": 5250,
        }
    ]

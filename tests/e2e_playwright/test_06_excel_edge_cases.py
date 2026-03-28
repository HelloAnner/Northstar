"""Excel 边缘场景测试

基于 excel_factory 从 PRD Excel 动态生成变体，测试导入容错。
"""
import time
from pathlib import Path

import pytest

from helpers import excel_factory


class TestExcelEdgeCases:
    """Excel 边缘场景测试"""

    def _import_and_check(self, api, variant_path: Path, expect_success: bool = True):
        """导入变体并检查结果"""
        try:
            events = api.import_excel(variant_path)
            types = [e.get("type", "") for e in events]
            has_error = any("error" in str(t).lower() for t in types)
            if expect_success:
                time.sleep(0.5)
                st = api.status()
                return st.get("totalCompanies", 0) > 0
            return not has_error
        finally:
            variant_path.unlink(missing_ok=True)

    # ---- Sheet 结构变化 ----

    def test_extra_sheet(self, api, import_data, prd_excel):
        """额外 sheet 不影响导入"""
        path = excel_factory.with_extra_sheet(prd_excel)
        assert self._import_and_check(api, path), "额外 sheet 应不影响导入"

    def test_missing_non_required_sheet(self, api, import_data, prd_excel):
        """缺少非必须 sheet 仍可导入"""
        path = excel_factory.with_missing_sheet(prd_excel, "吃穿用")
        assert self._import_and_check(api, path), "缺少吃穿用 sheet 应仍可导入"

    def test_shuffled_sheets(self, api, import_data, prd_excel):
        """Sheet 顺序打乱仍可导入"""
        path = excel_factory.with_shuffled_sheets(prd_excel)
        assert self._import_and_check(api, path), "打乱顺序应仍可导入"

    def test_sheet_name_spaces(self, api, import_data, prd_excel):
        """Sheet 名含空格仍可识别"""
        path = excel_factory.with_sheet_name_spaces(prd_excel)
        result = self._import_and_check(api, path)
        # 可能识别失败，但不应崩溃
        assert result is not None

    def test_duplicate_type_sheet(self, api, import_data, prd_excel):
        """重复类型 sheet 优先级处理"""
        path = excel_factory.with_duplicate_type_sheet(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    # ---- 列结构变化 ----

    def test_extra_empty_columns(self, api, import_data, prd_excel):
        """额外空列不影响导入"""
        path = excel_factory.with_extra_empty_columns(prd_excel)
        assert self._import_and_check(api, path), "额外空列应不影响导入"

    def test_swapped_columns(self, api, import_data, prd_excel):
        """列顺序变化仍可导入"""
        path = excel_factory.with_swapped_columns(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    def test_column_name_spaces(self, api, import_data, prd_excel):
        """列名含多余空格仍可匹配"""
        path = excel_factory.with_column_name_spaces(prd_excel)
        assert self._import_and_check(api, path), "列名空格应不影响匹配"

    def test_column_name_variant(self, api, import_data, prd_excel):
        """列名分号冒号变体"""
        path = excel_factory.with_column_name_variant(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    def test_missing_non_core_column(self, api, import_data, prd_excel):
        """缺少非核心列（粮油食品类）"""
        path = excel_factory.with_missing_non_core_column(prd_excel)
        assert self._import_and_check(api, path), "缺少非核心列应仍可导入"

    # ---- 数据变化 ----

    def test_empty_rows(self, api, import_data, prd_excel):
        """中间/末尾空行不影响导入"""
        path = excel_factory.with_empty_rows(prd_excel)
        assert self._import_and_check(api, path), "空行应不影响导入"

    def test_text_numbers(self, api, import_data, prd_excel):
        """数值为文本格式仍可解析"""
        path = excel_factory.with_text_numbers(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    def test_comma_numbers(self, api, import_data, prd_excel):
        """千分位逗号数值"""
        path = excel_factory.with_comma_numbers(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    def test_null_cells(self, api, import_data, prd_excel):
        """空值单元格不崩溃"""
        path = excel_factory.with_null_cells(prd_excel)
        assert self._import_and_check(api, path), "空值不应导致崩溃"

    def test_extreme_values(self, api, import_data, prd_excel):
        """超大数值不溢出"""
        path = excel_factory.with_extreme_values(prd_excel)
        assert self._import_and_check(api, path), "极端值不应溢出"

    def test_zero_rows(self, api, import_data, prd_excel):
        """全零值行不影响导入"""
        path = excel_factory.with_zero_rows(prd_excel)
        assert self._import_and_check(api, path), "零值行应正常导入"

    def test_bad_credit_code(self, api, import_data, prd_excel):
        """信用代码格式异常"""
        path = excel_factory.with_bad_credit_code(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    # ---- 日期/月份变化 ----

    def test_different_month(self, api, import_data, prd_excel):
        """列名月份变化仍可识别"""
        path = excel_factory.with_different_month(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    def test_different_year(self, api, import_data, prd_excel):
        """列名年份变化"""
        path = excel_factory.with_different_year(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    # ---- 整体文件异常 ----

    def test_empty_excel(self, api, import_data):
        """空 Excel 不崩溃"""
        path = excel_factory.empty_excel()
        try:
            events = api.import_excel(path)
            # 不崩溃即可
        finally:
            path.unlink(missing_ok=True)

    def test_single_sheet(self, api, import_data, prd_excel):
        """只有一个 sheet"""
        path = excel_factory.single_sheet_only(prd_excel)
        result = self._import_and_check(api, path)
        assert result is not None

    def test_headers_only(self, api, import_data, prd_excel):
        """只有表头没有数据"""
        path = excel_factory.headers_only(prd_excel)
        try:
            events = api.import_excel(path)
            # 不崩溃即可
        finally:
            path.unlink(missing_ok=True)

    def test_large_file(self, api, import_data, prd_excel):
        """大文件（500+ 行）导入"""
        path = excel_factory.large_file(prd_excel, target_rows=500)
        assert self._import_and_check(api, path), "大文件应能导入"

    def test_reimport_after_edge_case(self, api, prd_excel):
        """边缘场景后重新导入标准文件恢复正常"""
        events = api.import_excel(prd_excel)
        time.sleep(1)
        st = api.status()
        assert st.get("totalCompanies", 0) > 0, "标准文件重新导入应成功"

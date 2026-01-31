#!/usr/bin/env python3
import argparse
import json
import math
import os
import re
import sys
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Tuple

from openpyxl import load_workbook


def _read_text(path: Optional[str]) -> str:
    if not path:
        return ""
    p = Path(path)
    if not p.exists():
        return ""
    return p.read_text(encoding="utf-8", errors="replace")


def _load_json(path: str) -> Dict[str, Any]:
    return json.loads(Path(path).read_text(encoding="utf-8"))

def _unwrap_agent_browser_json(d: Dict[str, Any]) -> Dict[str, Any]:
    # agent-browser --json wraps as {success,data:{result:...}}; errors as {success:false,error:"..."}.
    if not isinstance(d, dict):
        return {"error": "invalid json", "rows": []}
    if d.get("success") is False:
        return {"error": d.get("error") or "agent-browser error", "rows": []}
    data = d.get("data")
    if isinstance(data, dict) and "result" in data and isinstance(data["result"], dict):
        return data["result"]
    return d


def _parse_number(v: Any) -> Optional[float]:
    if v is None:
        return None
    if isinstance(v, (int, float)) and not isinstance(v, bool):
        x = float(v)
        if math.isfinite(x):
            return x
        return None
    s = str(v).strip()
    if not s or s == "-":
        return None
    s = s.replace(",", "")
    if s.endswith("%"):
        s = s[:-1]
    try:
        x = float(s)
        if math.isfinite(x):
            return x
        return None
    except Exception:
        return None


def _close(a: Optional[float], b: Optional[float], eps: float = 1e-6) -> bool:
    if a is None and b is None:
        return True
    if a is None or b is None:
        return False
    return abs(a - b) <= eps


def _field_eps(field: str) -> float:
    # UI 通常展示 2 位小数；增速/比例类字段允许更大的容差
    if "增速" in field or "零销比" in field or "rate" in field.lower() or "%" in field:
        return 0.02
    return 0.005


def _normalize_rate_pair(field: str, a: Optional[float], b: Optional[float]) -> Tuple[Optional[float], Optional[float]]:
    # Some sheets store rate as decimal (0.54) while UI/export shows percent (54).
    # Normalize by scaling the smaller-magnitude side when it looks like a 100x difference.
    if a is None or b is None:
        return a, b
    is_rate = "增速" in field or "零销比" in field or "rate" in field.lower() or "%" in field
    if not is_rate:
        return a, b
    aa = abs(a)
    bb = abs(b)
    if aa <= 2 and bb > 2:
        return a * 100.0, b
    if bb <= 2 and aa > 2:
        return a, b * 100.0
    return a, b


def _safe(s: str) -> str:
    return (
        s.replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
        .replace("'", "&#39;")
    )


def _rel(p: str) -> str:
    try:
        return os.path.relpath(p, start=str(Path(p).parent.parent))
    except Exception:
        return p


def _find_header_row(ws) -> Optional[int]:
    # Heuristic: find row containing "统一社会信用代码"
    for r in range(1, 10):
        vals = [ws.cell(r, c).value for c in range(1, min(80, ws.max_column + 1))]
        joined = " ".join([str(v) for v in vals if v not in (None, "")])
        if "统一社会信用代码" in joined:
            return r
    return None


_CREDIT_CODE_RE = re.compile(r"^[0-9A-Z]{18}$")


def _is_credit_code(v: Any) -> bool:
    if v is None:
        return False
    s = str(v).strip().upper()
    return bool(_CREDIT_CODE_RE.match(s))

def _get_header_cols(headers: List[Any], name: str, nth: int = 1) -> Optional[int]:
    # Returns 1-based column index for nth occurrence of header name.
    seen = 0
    for i, h in enumerate(headers, start=1):
        if h is None:
            continue
        if str(h).strip() == name:
            seen += 1
            if seen == nth:
                return i
    return None


def _sheet_table(ws) -> Optional[Tuple[Dict[str, int], Dict[str, int]]]:
    header_row = _find_header_row(ws)
    if not header_row:
        return None
    headers: Dict[str, int] = {}
    for c in range(1, ws.max_column + 1):
        v = ws.cell(header_row, c).value
        if v is None or str(v).strip() == "":
            continue
        headers[str(v).strip()] = c
    if not headers:
        return None

    code_col = headers.get("统一社会信用代码")
    if not code_col:
        return None

    rows: Dict[str, int] = {}
    for r in range(header_row + 1, ws.max_row + 1):
        code = ws.cell(r, code_col).value
        if not _is_credit_code(code):
            continue
        rows[str(code).strip().upper()] = r
    return headers, rows


UI_TO_EXCEL_FIELD = {
    # 住餐在 UI 里使用更口语化的列名
    "本月客房收入": "客房收入;本年-本月",
    "本月餐费收入": "餐费收入;本年-本月",
    "本月商品销售额": "商品销售额;本年-本月",
    # UI 里的“增速”列名与导出表的衍生指标列名不同
    "商品销售额;增速(当月)": "(衍生指标)销售额当月增速",
    "商品销售额;累计增速": "(衍生指标)销售额累计增速",
    "零售额;增速(当月)": "(衍生指标)零售额当月增速",
    "零售额;累计增速": "(衍生指标)零售额累计增速",
    "营业额;增速(当月)": "(衍生指标)营业额当月增速",
    "营业额;累计增速": "(衍生指标)营业额累计增速",
}

DERIVED_SHEET_MAPPINGS: Dict[str, List[Tuple[str, str, int]]] = {
    # (ui_field, excel_header, nth_occurrence)
    "批发": [
        ("销售额;本年-上月", "2025年11月销售额", 1),
        ("销售额;本年-本月", "2025年12月销售额", 1),
        ("销售额;上年-本月", "2024年;12月;商品销售额;千元", 1),
        ("销售额;本年-1—上月", "2025年1-11月销售额", 1),
        ("销售额;上年-1—上月", "2024年1-11月销售额", 1),
        ("销售额;本年-1—本月", "2025年1-12月销售额", 1),
        ("销售额;上年-1—本月", "2024年;1-12月;商品销售额;千元", 1),
        ("销售额;增速(当月)", "12月销售额增速", 1),
        ("销售额;累计增速", "1-12月增速", 1),
        ("零售额;本年-上月", "2025年11月零售额", 1),
        ("零售额;本年-本月", "2025年12月零售额", 1),
        ("零售额;上年-本月", "2024年;12月;商品零售额;千元", 1),
        ("零售额;本年-1—上月", "2025年1-11月零售额", 1),
        ("零售额;上年-1—上月", "2024年1-11月零售额", 1),
        ("零售额;本年-1—本月", "2025年1-12月零售额", 1),
        ("零售额;上年-1—本月", "2024年;1-12月;商品零售额;千元", 1),
        ("零售额;增速(当月)", "12月零售额增速", 1),
        ("零售额;累计增速", "1-12月增速", 2),
        ("零销比(%)", "零售额占比", 1),
    ],
    "零售": [
        ("销售额;本年-上月", "2025年11月销售额", 1),
        ("销售额;本年-本月", "2025年12月销售额", 1),
        ("销售额;上年-本月", "2024年;12月;商品销售额;千元", 1),
        ("销售额;本年-1—上月", "2025年1-11月销售额", 1),
        ("销售额;上年-1—上月", "2024年1-11月销售额", 1),
        ("销售额;本年-1—本月", "2025年1-12月销售额", 1),
        ("销售额;上年-1—本月", "2024年;1-12月;商品销售额;千元", 1),
        ("销售额;增速(当月)", "12月销售额增速", 1),
        ("销售额;累计增速", "1-12月增速", 1),
        ("零售额;本年-上月", "2025年11月零售额", 1),
        ("零售额;本年-本月", "2025年12月零售额", 1),
        ("零售额;上年-本月", "2024年;12月;商品零售额;千元", 1),
        ("零售额;本年-1—上月", "2025年1-11月零售额", 1),
        ("零售额;上年-1—上月", "2024年1-11月零售额", 1),
        ("零售额;本年-1—本月", "2025年1-12月零售额", 1),
        ("零售额;上年-1—本月", "2024年;1-12月;商品零售额;千元", 1),
        ("零售额;增速(当月)", "12月零售额增速", 1),
        ("零售额;累计增速", "1-12月增速", 2),
        ("零销比(%)", "零售额占比", 1),
    ],
    "住宿": [
        # UI 住餐口径中，主指标列名仍沿用“销售额;*”
        ("销售额;本年-上月", "2025年11月营业额", 1),
        ("销售额;本年-本月", "2025年12月营业额", 1),
        ("销售额;上年-本月", "2024年12月;营业额总计;千元", 1),
        ("销售额;本年-1—上月", "2025年1-11月营业额", 1),
        ("销售额;本年-1—本月", "2025年1-12月营业额", 1),
        ("销售额;上年-1—本月", "2024年1-12月;营业额总计;千元", 1),
        ("销售额;增速(当月)", "12月增速", 1),
        ("销售额;累计增速", "1-12月增速", 1),
        ("客房收入;本年-上月", "11月客房收入", 1),
        ("客房收入;本年-本月", "2025年12月客房收入", 1),
        ("客房收入;上年-本月", "2024年12月;营业额总计;客房收入;千元", 1),
        ("客房收入;本年-1—上月", "2025年1-11月客房收入", 1),
        ("客房收入;本年-1—本月", "2025年1-12月客房收入", 1),
        ("客房收入;上年-1—本月", "2024年1-12月;营业额总计;客房收入;千元", 1),
        ("餐费收入;本年-上月", "11月餐费收入", 1),
        ("餐费收入;本年-本月", "2025年12月餐费收入", 1),
        ("餐费收入;上年-本月", "2024年12月;营业额总计;餐费收入;千元", 1),
        ("餐费收入;本年-1—上月", "2025年1-11月餐费收入", 1),
        ("餐费收入;本年-1—本月", "1-12月餐费收入", 1),
        ("餐费收入;上年-1—本月", "2024年1-12月;营业额总计;餐费收入;千元", 1),
        ("商品销售额;本年-上月", "11月销售额", 1),
        ("商品销售额;本年-本月", "2025年12月销售额", 1),
        ("商品销售额;上年-本月", "2024年12月;营业额总计;商品销售额;千元", 1),
        ("商品销售额;本年-1—上月", "2025年1-11月销售额", 1),
        ("商品销售额;本年-1—本月", "1-12月销售额", 1),
        ("商品销售额;上年-1—本月", "2024年1-12月;营业额总计;商品销售额;千元", 1),
        ("零售额;本年-本月", "2025年12月零售额", 1),
        ("零售额;上年-本月", "2024年12月零售额", 1),
    ],
    "餐饮": [
        ("销售额;本年-上月", "2025年11月营业额", 1),
        ("销售额;本年-本月", "2025年12月营业额", 1),
        ("销售额;上年-本月", "2024年12月;营业额总计;千元", 1),
        ("销售额;本年-1—上月", "2025年1-11月营业额", 1),
        ("销售额;本年-1—本月", "2025年1-12月营业额", 1),
        ("销售额;上年-1—本月", "2024年1-12月;营业额总计;千元", 1),
        ("销售额;增速(当月)", "12月增速", 1),
        ("销售额;累计增速", "1-12月增速", 1),
        ("客房收入;本年-上月", "11月客房收入", 1),
        ("客房收入;本年-本月", "2025年12月客房收入", 1),
        ("客房收入;上年-本月", "2024年12月;营业额总计;客房收入;千元", 1),
        ("客房收入;本年-1—上月", "2025年1-11月客房收入", 1),
        ("客房收入;本年-1—本月", "2025年1-12月客房收入", 1),
        ("客房收入;上年-1—本月", "2024年1-12月;营业额总计;客房收入;千元", 1),
        ("餐费收入;本年-上月", "11月餐费收入", 1),
        ("餐费收入;本年-本月", "2025年12月餐费收入", 1),
        ("餐费收入;上年-本月", "2024年12月;营业额总计;餐费收入;千元", 1),
        ("餐费收入;本年-1—上月", "2025年1-11月餐费收入", 1),
        ("餐费收入;本年-1—本月", "1-12月餐费收入", 1),
        ("餐费收入;上年-1—本月", "2024年1-12月;营业额总计;餐费收入;千元", 1),
        ("商品销售额;本年-上月", "11月销售额", 1),
        ("商品销售额;本年-本月", "2025年12月销售额", 1),
        ("商品销售额;上年-本月", "2024年12月;营业额总计;商品销售额;千元", 1),
        ("商品销售额;本年-1—上月", "2025年1-11月销售额", 1),
        ("商品销售额;本年-1—本月", "1-12月销售额", 1),
        ("商品销售额;上年-1—本月", "2024年1-12月;营业额总计;商品销售额;千元", 1),
        ("零售额;本年-本月", "2025年12月零售额", 1),
        ("零售额;上年-本月", "2024年12月零售额", 1),
    ],
}


@dataclass
class Mismatch:
    credit_code: str
    name: str
    sheet: str
    field: str
    expected: Any
    actual: Any


def _compare_ui_vs_excel(
    ui_rows: List[Dict[str, Any]],
    wb,
    ignore_fields: Iterable[str],
) -> Tuple[List[str], List[Mismatch]]:
    ignore = set(ignore_fields)
    missing_rows: List[str] = []
    mismatches: List[Mismatch] = []

    # Group UI rows by source sheet (UI extracted key: "来源表")
    by_sheet: Dict[str, List[Dict[str, Any]]] = {}
    for r in ui_rows:
        sheet = str(r.get("来源表") or "").strip()
        by_sheet.setdefault(sheet, []).append(r)

    for sheet_name, rows in by_sheet.items():
        if not sheet_name:
            for r in rows:
                missing_rows.append(f"{r.get('__creditCode')}: missing 来源表")
            continue
        if sheet_name not in wb.sheetnames:
            for r in rows:
                missing_rows.append(f"{r.get('__creditCode')}: 来源表 not found in excel: {sheet_name}")
            continue

        ws = wb[sheet_name]
        table = _sheet_table(ws)
        if not table:
            for r in rows:
                missing_rows.append(f"{r.get('__creditCode')}: sheet has no recognizable header table: {sheet_name}")
            continue
        headers, excel_rows = table

        # Ensure excel rows -> UI rows coverage (excel has row -> UI must exist)
        ui_codes = {str(r.get("__creditCode") or "").strip().upper() for r in rows if _is_credit_code(r.get("__creditCode"))}
        for code in excel_rows.keys():
            if code not in ui_codes:
                missing_rows.append(f"{code}: exists in excel sheet {sheet_name} but missing in 明细表")

        # Compare values where both sides have the field header name.
        for r in rows:
            code = str(r.get("__creditCode") or "").strip()
            name = str(r.get("__name") or "").strip()
            if not _is_credit_code(code):
                continue
            code_u = str(code).strip().upper()
            excel_r = excel_rows.get(code_u)
            if not excel_r:
                missing_rows.append(f"{code_u}: not found in excel sheet {sheet_name}")
                continue

            for field, ui_val in r.items():
                if not isinstance(field, str):
                    continue
                if field.startswith("__") or field in ignore:
                    continue
                excel_field = field if field in headers else UI_TO_EXCEL_FIELD.get(field, "")
                if not excel_field or excel_field not in headers:
                    continue
                excel_val = ws.cell(excel_r, headers[excel_field]).value
                if excel_val is None or (isinstance(excel_val, str) and excel_val.strip() == ""):
                    continue

                ui_num = _parse_number(ui_val)
                ex_num = _parse_number(excel_val)

                # Prefer numeric compare when both parse.
                if ui_num is not None or ex_num is not None:
                    ui_num, ex_num = _normalize_rate_pair(field, ui_num, ex_num)
                    if not _close(ui_num, ex_num, eps=_field_eps(field)):
                        mismatches.append(
                            Mismatch(
                                credit_code=code,
                                name=name,
                                sheet=sheet_name,
                                field=f"{field} (excel:{excel_field})" if excel_field != field else field,
                                expected=excel_val,
                                actual=ui_val,
                            )
                        )
                else:
                    if str(ui_val).strip() != str(excel_val).strip():
                        mismatches.append(
                            Mismatch(
                                credit_code=code,
                                name=name,
                                sheet=sheet_name,
                                field=f"{field} (excel:{excel_field})" if excel_field != field else field,
                                expected=excel_val,
                                actual=ui_val,
                            )
                        )

    return missing_rows, mismatches


def _export_index(wb) -> Dict[str, Tuple[Dict[str, int], Dict[str, int]]]:
    idx: Dict[str, Tuple[Dict[str, int], Dict[str, int]]] = {}
    for name in wb.sheetnames:
        ws = wb[name]
        table = _sheet_table(ws)
        if table:
            idx[name] = table
    return idx


def _compare_ui_vs_export(ui_rows: List[Dict[str, Any]], export_wb) -> Tuple[List[str], List[Mismatch]]:
    missing: List[str] = []
    mismatches: List[Mismatch] = []

    idx = _export_index(export_wb)
    preferred_sheets = {
        "批发": ["批发", "批零总表"],
        "零售": ["零售", "批零总表"],
        "住宿": ["住宿", "住餐总表"],
        "餐饮": ["餐饮", "住餐总表"],
    }

    for r in ui_rows:
        code = str(r.get("__creditCode") or "").strip()
        name = str(r.get("__name") or "").strip()
        industry = str(r.get("__industry") or "").strip()
        if not _is_credit_code(code):
            continue
        code_u = str(code).strip().upper()

        candidates = preferred_sheets.get(industry, ["批零总表", "住餐总表", "批发", "零售", "住宿", "餐饮"])
        found_sheet: Optional[str] = None
        ex_headers: Dict[str, int] = {}
        ex_rows: Dict[str, int] = {}
        for s in candidates:
            t = idx.get(s)
            if not t:
                continue
            headers, rows = t
            if code_u in rows:
                found_sheet = s
                ex_headers, ex_rows = headers, rows
                break

        if not found_sheet:
            missing.append(f"{code_u}: not found in exported excel (industry={industry})")
            continue

        ex_r = ex_rows[code_u]
        for field, ui_val in r.items():
            if not isinstance(field, str):
                continue
            if field.startswith("__"):
                continue
            if field in ("规模", "标记", "来源表"):
                continue
            excel_field = field if field in ex_headers else UI_TO_EXCEL_FIELD.get(field, "")
            if not excel_field or excel_field not in ex_headers:
                continue
            excel_val = export_wb[found_sheet].cell(ex_r, ex_headers[excel_field]).value
            ui_num = _parse_number(ui_val)
            ex_num = _parse_number(excel_val)
            if ui_num is not None or ex_num is not None:
                ui_num, ex_num = _normalize_rate_pair(field, ui_num, ex_num)
                if not _close(ui_num, ex_num, eps=_field_eps(field)):
                    mismatches.append(
                        Mismatch(
                            credit_code=code,
                            name=name,
                            sheet=found_sheet,
                            field=f"{field} (export:{excel_field})" if excel_field != field else field,
                            expected=excel_val,
                            actual=ui_val,
                        )
                    )
            else:
                if str(ui_val).strip() != str(excel_val).strip():
                    mismatches.append(
                            Mismatch(
                                credit_code=code,
                                name=name,
                                sheet=found_sheet,
                                field=f"{field} (export:{excel_field})" if excel_field != field else field,
                                expected=excel_val,
                                actual=ui_val,
                            )
                        )

    return missing, mismatches


@dataclass
class CompletenessCase:
    sheet: str
    credit_code: str
    name: str
    field: str
    expected: Any
    actual: Any
    ok: bool
    reason: str
    reproduce: str


def _build_expected_from_derived_sheet(wb, sheet_name: str) -> Dict[str, Dict[str, Any]]:
    # Returns creditCode -> {name, expected{ui_field:value}, sheet}
    if sheet_name not in wb.sheetnames:
        return {}
    ws = wb[sheet_name]
    headers = [ws.cell(1, c).value for c in range(1, ws.max_column + 1)]
    code_col = _get_header_cols(headers, "统一社会信用代码", 1)
    name_col = _get_header_cols(headers, "单位详细名称", 1)
    if not code_col or not name_col:
        return {}

    mapping = DERIVED_SHEET_MAPPINGS.get(sheet_name) or []
    if not mapping:
        return {}

    out: Dict[str, Dict[str, Any]] = {}
    for r in range(2, ws.max_row + 1):
        code = ws.cell(r, code_col).value
        if not _is_credit_code(code):
            continue
        code_u = str(code).strip().upper()
        name = str(ws.cell(r, name_col).value or "").strip()

        expected: Dict[str, Any] = {}
        for ui_field, excel_header, nth in mapping:
            col = _get_header_cols(headers, excel_header, nth)
            if not col:
                continue
            v = ws.cell(r, col).value
            if v is None or (isinstance(v, str) and v.strip() == ""):
                continue
            expected[ui_field] = v

        out[code_u] = {"name": name, "sheet": sheet_name, "expected": expected}
    return out


def _build_completeness_cases(
    input_wb,
    ui_rows: List[Dict[str, Any]],
) -> Tuple[List[CompletenessCase], Dict[str, Any]]:
    # Build UI index by credit code
    ui_by_code: Dict[str, Dict[str, Any]] = {}
    for r in ui_rows:
        code = r.get("__creditCode")
        if not _is_credit_code(code):
            continue
        ui_by_code[str(code).strip().upper()] = r

    cases: List[CompletenessCase] = []
    summary = {
        "sheets": {},
        "totalCompanies": 0,
        "missingCompanies": 0,
        "totalChecks": 0,
        "failedChecks": 0,
        "missingCodesBySheet": {},
    }

    for sheet_name in ["批发", "零售", "住宿", "餐饮"]:
        expected_map = _build_expected_from_derived_sheet(input_wb, sheet_name)
        sheet_summary = {"companies": len(expected_map), "missingCompanies": 0, "checks": 0, "fails": 0}
        summary["totalCompanies"] += len(expected_map)

        ui_codes_for_sheet = {
            str(r.get("__creditCode") or "").strip().upper()
            for r in ui_rows
            if str(r.get("来源表") or "").strip() == sheet_name and _is_credit_code(r.get("__creditCode"))
        }
        missing_codes = sorted([code for code in expected_map.keys() if code not in ui_codes_for_sheet])
        summary["missingCodesBySheet"][sheet_name] = missing_codes
        sheet_summary["missingCompanies"] = len(missing_codes)
        summary["missingCompanies"] += len(missing_codes)

        for code, info in expected_map.items():
            expected_fields: Dict[str, Any] = info.get("expected") or {}
            name = str(info.get("name") or "").strip()
            ui_row = ui_by_code.get(code)
            if not ui_row:
                continue

            ui_sheet = str(ui_row.get("来源表") or "").strip()
            ui_name = str(ui_row.get("__name") or name).strip()

            for field, exp in expected_fields.items():
                act = ui_row.get(field, "-")
                sheet_summary["checks"] += 1
                summary["totalChecks"] += 1

                if ui_sheet and ui_sheet != sheet_name:
                    okv = False
                    reason = f"来源表不一致：UI={ui_sheet}，Excel={sheet_name}"
                elif act in (None, "", "-"):
                    okv = False
                    reason = "Excel 有值，但明细表该字段为空/未展示"
                else:
                    exp_num = _parse_number(exp)
                    act_num = _parse_number(act)
                    if exp_num is not None or act_num is not None:
                        act_num, exp_num = _normalize_rate_pair(field, act_num, exp_num)
                        okv = _close(act_num, exp_num, eps=_field_eps(field))
                        reason = "" if okv else "数值不一致（允许少量格式化/四舍五入容差）"
                    else:
                        okv = str(act).strip() == str(exp).strip()
                        reason = "" if okv else "文本不一致"

                if not okv:
                    sheet_summary["fails"] += 1
                    summary["failedChecks"] += 1

                cases.append(
                    CompletenessCase(
                        sheet=sheet_name,
                        credit_code=code,
                        name=ui_name,
                        field=field,
                        expected=exp,
                        actual=act,
                        ok=okv,
                        reason=reason,
                        reproduce=f"首页明细表搜索 {code} → 展示列 {field} → 对照输入 Excel Sheet「{sheet_name}」的对应列",
                    )
                )

        summary["sheets"][sheet_name] = sheet_summary

    return cases, summary


def _summarize_mismatches(mismatches: List[Mismatch], limit: int = 200) -> str:
    if not mismatches:
        return "<p class='ok'>✅ 未发现差异</p>"
    rows = []
    for m in mismatches[:limit]:
        rows.append(
            "<tr>"
            f"<td>{_safe(m.credit_code)}</td>"
            f"<td>{_safe(m.name)}</td>"
            f"<td>{_safe(m.sheet)}</td>"
            f"<td>{_safe(m.field)}</td>"
            f"<td class='mono'>{_safe(str(m.expected))}</td>"
            f"<td class='mono'>{_safe(str(m.actual))}</td>"
            "</tr>"
        )
    more = ""
    if len(mismatches) > limit:
        more = f"<p class='warn'>仅展示前 {limit} 条，剩余 {len(mismatches)-limit} 条请查看原始数据 JSON。</p>"
    return (
        f"<p class='bad'>❌ 差异数量：{len(mismatches)}</p>"
        + more
        + "<div class='table-wrap'><table><thead><tr>"
        "<th>统一社会信用代码</th><th>企业</th><th>Sheet</th><th>字段</th><th>Excel</th><th>明细表</th>"
        "</tr></thead><tbody>"
        + "".join(rows)
        + "</tbody></table></div>"
    )


def _summarize_missing(missing: List[str], limit: int = 200) -> str:
    if not missing:
        return "<p class='ok'>✅ 覆盖完整</p>"
    items = "".join(f"<li class='mono'>{_safe(x)}</li>" for x in missing[:limit])
    more = ""
    if len(missing) > limit:
        more = f"<p class='warn'>仅展示前 {limit} 条，剩余 {len(missing)-limit} 条已省略。</p>"
    return f"<p class='bad'>❌ 覆盖问题：{len(missing)}</p>{more}<ul>{items}</ul>"

def _suggestions(
    ui_before_ok: bool,
    ui_after_ok: bool,
    before_rows: List[Dict[str, Any]],
    after_rows: List[Dict[str, Any]],
    input_wb,
    export_wb,
    missing_before: List[str],
    mismatches_before: List[Mismatch],
    missing_export: List[str],
    mismatches_export: List[Mismatch],
) -> str:
    hints: List[str] = []

    if not ui_before_ok or not ui_after_ok:
        hints.append("UI 明细表抽取失败：优先确认页面结构是否变化（table/td/input），或导入后是否出现“暂无数据”。")

    if input_wb is not None and ui_before_ok:
        excel_tables = [n for n in input_wb.sheetnames if _sheet_table(input_wb[n])]
        if not excel_tables:
            hints.append("输入 Excel 未识别到表头：检查是否存在“统一社会信用代码”列，或表头行不在前 10 行。")

    if export_wb is not None and ui_after_ok:
        # Heuristic: export may not contain per-company credit code rows.
        tables = [n for n in export_wb.sheetnames if _sheet_table(export_wb[n])]
        if not tables:
            hints.append("导出 Excel 未识别到企业明细表格：导出可能只包含汇总/衍生指标；建议在导出文件中确认是否存在“统一社会信用代码”列。")

    if missing_before:
        hints.append("导入覆盖缺失：若明细表分页/后端 pageSize 限制导致未加载全量企业，需要在 UI 或 API 增加分页拉全量（当前测试会在 report 中列出缺失 code）。")

    if mismatches_before:
        hints.append("导入字段不一致：通常是数值格式化/四舍五入/单位差异导致，建议对比 report 中同一企业同一字段的 Excel vs 明细表值。")

    if missing_export:
        hints.append("导出覆盖缺失：导出 Excel 里找不到信用代码对应行，可能是导出表不含企业级明细或 code 未写入。")

    if mismatches_export:
        hints.append("导出字段不一致：优先排查修改是否真正落库（输入框 blur 自动保存），以及导出是否读取最新数据。")

    if not hints:
        return "<p class='ok'>✅ 暂无额外改动建议</p>"

    items = "".join(f"<li>{_safe(x)}</li>" for x in hints)
    return f"<ul>{items}</ul>"


def _issues_summary(
    missing_before: List[str],
    mismatches_before: List[Mismatch],
    missing_export: List[str],
    mismatches_export: List[Mismatch],
    action_results: List[Dict[str, Any]],
    action_persist: List[Dict[str, Any]],
    completeness_failed: int,
    completeness_total: int,
    derived_unmapped_cols: int,
    derived_missing_ui_cols: int,
) -> str:
    issues: List[str] = []
    action_fail = sum(1 for r in action_results if r.get("ok") is False)
    persist_fail = sum(1 for r in action_persist if r.get("ok") is False)

    if missing_before:
        issues.append(f"导入覆盖缺失：{len(missing_before)}（见“导入一致性/覆盖检查”）")
    if mismatches_before:
        issues.append(f"导入字段不一致：{len(mismatches_before)}（见“导入一致性/字段一致性”）")
    if action_fail:
        issues.append(f"修改动作失败：{action_fail}（见“修改动作覆盖”）")
    if persist_fail:
        issues.append(f"修改持久化失败：{persist_fail}（见“修改动作覆盖”的 UI(刷新后) 列）")
    if missing_export:
        issues.append(f"导出覆盖缺失：{len(missing_export)}（见“导出一致性/覆盖检查”）")
    if mismatches_export:
        issues.append(f"导出字段不一致：{len(mismatches_export)}（见“导出一致性/字段一致性”）")
    if completeness_total > 0 and completeness_failed > 0:
        issues.append(f"明细表完整性断言失败：{completeness_failed}/{completeness_total}（见“明细表数据完整性案例”）")
    if derived_unmapped_cols > 0:
        issues.append(f"衍生 Sheet 有值列未映射：{derived_unmapped_cols}（见“衍生 Sheet 列覆盖”）")
    if derived_missing_ui_cols > 0:
        issues.append(f"衍生 Sheet 映射列在 UI 缺失：{derived_missing_ui_cols}（见“衍生 Sheet 列覆盖”）")

    if not issues:
        return "<p class='ok'>✅ 未发现不符合预期项</p>"
    return "<ul>" + "".join(f"<li>{_safe(x)}</li>" for x in issues) + "</ul>"


def _list_screenshots(dir_path: str) -> List[str]:
    p = Path(dir_path)
    if not p.exists():
        return []
    return [str(x) for x in sorted(p.glob("*.png"))]


def _write_json(path: Path, obj: Any) -> None:
    path.write_text(json.dumps(obj, ensure_ascii=False, indent=2), encoding="utf-8")


def _derived_columns(ws) -> List[Tuple[str, int, int]]:
    # Returns (headerText, nthOccurrence, colIndex) for derived sheets (row 1 headers).
    seen: Dict[str, int] = {}
    out: List[Tuple[str, int, int]] = []
    for col in range(1, ws.max_column + 1):
        h = ws.cell(1, col).value
        if h is None or str(h).strip() == "":
            continue
        name = str(h).strip()
        seen[name] = seen.get(name, 0) + 1
        out.append((name, seen[name], col))
    return out


def _col_non_empty_stats(ws, col: int, max_examples: int = 3) -> Tuple[int, List[Any]]:
    count = 0
    examples: List[Any] = []
    for r in range(2, ws.max_row + 1):
        v = ws.cell(r, col).value
        if v is None or (isinstance(v, str) and v.strip() == ""):
            continue
        count += 1
        if len(examples) < max_examples and v not in examples:
            examples.append(v)
    return count, examples


def _guess_ui_field(excel_header: str) -> str:
    # Best-effort suggestion; used only for report hints.
    h = excel_header.strip()
    if h in {"粮油食品类", "饮料类", "烟酒类", "服装鞋帽针纺类", "日用品类", "汽车类", "吃穿用"}:
        return h
    if h in {"单位规模", "小微企业"}:
        return h
    return h


def _build_derived_column_coverage(input_wb, ui_headers: List[str]) -> Dict[str, Any]:
    # For each derived sheet, list columns that have values and whether they map to a UI column.
    out: Dict[str, Any] = {"sheets": {}}
    for sheet_name in ["批发", "零售", "住宿", "餐饮"]:
        if input_wb is None or sheet_name not in input_wb.sheetnames:
            continue
        ws = input_wb[sheet_name]
        mapping = DERIVED_SHEET_MAPPINGS.get(sheet_name) or []
        mapped: Dict[Tuple[str, int], str] = {(ex, nth): ui for (ui, ex, nth) in mapping}

        items: List[Dict[str, Any]] = []
        missing_ui_cols = 0
        unmapped_cols_with_values = 0
        for header, nth, col in _derived_columns(ws):
            if header in {"序号", "统一社会信用代码", "单位详细名称"}:
                continue
            if header.startswith("[201-1]"):
                continue
            non_empty, examples = _col_non_empty_stats(ws, col)
            if non_empty == 0:
                continue

            ui_field = mapped.get((header, nth), "")
            ui_present = bool(ui_field and ui_field in ui_headers)
            if not ui_field:
                # Unmapped column: still mark if the header itself is present in UI (rare, but helps debugging).
                ui_present = header in ui_headers
            if ui_field and not ui_present:
                missing_ui_cols += 1
            if not ui_field:
                unmapped_cols_with_values += 1

            items.append(
                {
                    "excelHeader": header,
                    "nth": nth,
                    "nonEmpty": non_empty,
                    "examples": examples,
                    "uiField": ui_field,
                    "uiPresent": ui_present,
                    "suggestedUiField": _guess_ui_field(header),
                }
            )

        out["sheets"][sheet_name] = {
            "columnsWithValues": len(items),
            "missingUiColumns": missing_ui_cols,
            "unmappedColumnsWithValues": unmapped_cols_with_values,
            "items": items,
        }
    return out


def _action_export_checks(
    action_results: List[Dict[str, Any]],
    ui_lookup_rows: List[Dict[str, Any]],
    export_wb,
) -> List[Dict[str, Any]]:
    if export_wb is None:
        return []
    ui_by_code = {
        str(r.get("__creditCode") or "").strip().upper(): r
        for r in ui_lookup_rows
        if _is_credit_code(r.get("__creditCode"))
    }
    idx = _export_index(export_wb)
    preferred_sheets = {
        "批发": ["批发", "批零总表"],
        "零售": ["零售", "批零总表"],
        "住宿": ["住宿", "住餐总表"],
        "餐饮": ["餐饮", "住餐总表"],
    }

    out: List[Dict[str, Any]] = []
    for a in action_results:
        cc = str(a.get("creditCode") or "").strip().upper()
        field = str(a.get("field") or "").strip()
        exp = a.get("value")

        r = ui_by_code.get(cc)
        if not r:
            out.append({"creditCode": cc, "field": field, "ok": False, "reason": "修改后 UI 未找到该企业，无法校验导出", "expected": exp})
            continue

        industry = str(r.get("__industry") or r.get("来源表") or "").strip()
        candidates = preferred_sheets.get(industry, []) + [industry]
        found_sheet = ""
        ex_headers: Dict[str, int] = {}
        ex_rows: Dict[str, int] = {}
        for s in candidates:
            if s in idx and cc in idx[s][1]:
                found_sheet = s
                ex_headers, ex_rows = idx[s]
                break
        if not found_sheet:
            out.append(
                {
                    "creditCode": cc,
                    "field": field,
                    "ok": False,
                    "reason": f"导出 Excel 未找到该企业行（industry={industry}）",
                    "expected": exp,
                }
            )
            continue

        excel_field = field if field in ex_headers else UI_TO_EXCEL_FIELD.get(field, "")
        if not excel_field or excel_field not in ex_headers:
            out.append(
                {
                    "creditCode": cc,
                    "field": field,
                    "ok": False,
                    "reason": f"导出 Excel 缺少对应列（field={field}）",
                    "sheet": found_sheet,
                    "expected": exp,
                }
            )
            continue

        ex_r = ex_rows[cc]
        actual = export_wb[found_sheet].cell(ex_r, ex_headers[excel_field]).value
        exp_num = _parse_number(exp)
        act_num = _parse_number(actual)
        if exp_num is not None or act_num is not None:
            act_num, exp_num = _normalize_rate_pair(field, act_num, exp_num)
            okv = _close(act_num, exp_num, eps=_field_eps(field))
        else:
            okv = str(actual).strip() == str(exp).strip()

        out.append(
            {
                "creditCode": cc,
                "field": field,
                "sheet": found_sheet,
                "excelField": excel_field,
                "expected": exp,
                "actual": actual,
                "ok": okv,
                "reason": "" if okv else "导出值与修改后预期不一致",
            }
        )
    return out


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--input-xlsx", required=True)
    ap.add_argument("--export-xlsx", required=True)
    ap.add_argument("--ui-before", required=True)
    ap.add_argument("--ui-after", required=True)
    ap.add_argument("--import-events", required=False, default="")
    ap.add_argument("--tab-counts", required=False, default="")
    ap.add_argument("--steps", required=False, default="")
    ap.add_argument("--actions", required=False, default="")
    ap.add_argument("--console", required=False, default="")
    ap.add_argument("--errors", required=False, default="")
    ap.add_argument("--trace", required=False, default="")
    ap.add_argument("--video", required=False, default="")
    ap.add_argument("--screenshots", required=False, default="")
    ap.add_argument("--out", required=True)
    args = ap.parse_args()

    ui_before = _unwrap_agent_browser_json(_load_json(args.ui_before))
    ui_after = _unwrap_agent_browser_json(_load_json(args.ui_after))
    before_rows = ui_before.get("rows") or []
    after_rows = ui_after.get("rows") or []

    ui_before_error = ui_before.get("error") or ""
    ui_after_error = ui_after.get("error") or ""
    ui_before_ok = isinstance(before_rows, list) and len(before_rows) > 0 and not ui_before_error
    ui_after_ok = isinstance(after_rows, list) and len(after_rows) > 0 and not ui_after_error

    input_wb_error = ""
    export_wb_error = ""
    try:
        input_wb = load_workbook(args.input_xlsx, data_only=True)
    except Exception as e:
        input_wb = None
        input_wb_error = str(e)
    try:
        export_wb = load_workbook(args.export_xlsx, data_only=True)
    except Exception as e:
        export_wb = None
        export_wb_error = str(e)

    missing_before: List[str] = []
    mismatches_before: List[Mismatch] = []
    if input_wb is not None and ui_before_ok:
        missing_before, mismatches_before = _compare_ui_vs_excel(
            ui_rows=before_rows,
            wb=input_wb,
            ignore_fields=["规模", "标记"],
        )
    elif not ui_before_ok:
        missing_before = ["UI 抽取失败：无法进行导入一致性校验"]
    elif input_wb is None:
        missing_before = [f"输入 Excel 打开失败：{input_wb_error}"]

    missing_export: List[str] = []
    mismatches_export: List[Mismatch] = []
    if export_wb is not None and ui_after_ok:
        missing_export, mismatches_export = _compare_ui_vs_export(
            ui_rows=after_rows,
            export_wb=export_wb,
        )
    elif not ui_after_ok:
        missing_export = ["UI 抽取失败：无法进行导出一致性校验"]
    elif export_wb is None:
        missing_export = [f"导出 Excel 打开失败：{export_wb_error}"]

    import_events = {}
    if args.import_events and Path(args.import_events).exists():
        try:
            import_events = _unwrap_agent_browser_json(_load_json(args.import_events))
        except Exception:
            import_events = {}

    tab_counts: Dict[str, Any] = {}
    if args.tab_counts and Path(args.tab_counts).exists():
        try:
            tab_counts = _unwrap_agent_browser_json(_load_json(args.tab_counts))
        except Exception:
            tab_counts = {}

    screenshots = _list_screenshots(args.screenshots) if args.screenshots else []

    started_at = ui_before.get("extractedAt") or ""
    ended_at = ui_after.get("extractedAt") or ""
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    steps: List[Dict[str, Any]] = []
    if args.steps and Path(args.steps).exists():
        try:
            steps = json.loads(Path(args.steps).read_text(encoding="utf-8"))
        except Exception:
            steps = []

    action_results: List[Dict[str, Any]] = []
    action_persist: List[Dict[str, Any]] = []
    if args.actions and Path(args.actions).exists():
        try:
            payload = json.loads(Path(args.actions).read_text(encoding="utf-8")) or {}
            action_results = payload.get("results") or []
            action_persist = payload.get("persist") or []
        except Exception:
            action_results = []
            action_persist = []

    out_dir = Path(args.out).parent
    try:
        _write_json(out_dir / "missing_before.json", missing_before)
        _write_json(out_dir / "mismatches_before.json", [m.__dict__ for m in mismatches_before])
        _write_json(out_dir / "missing_export.json", missing_export)
        _write_json(out_dir / "mismatches_export.json", [m.__dict__ for m in mismatches_export])
    except Exception:
        pass

    completeness_cases: List[CompletenessCase] = []
    completeness_summary: Dict[str, Any] = {}
    completeness_failed = 0
    completeness_total = 0
    missing_codes_by_sheet: Dict[str, List[str]] = {}
    derived_column_coverage: Dict[str, Any] = {}

    if input_wb is not None and ui_before_ok:
        try:
            completeness_cases, completeness_summary = _build_completeness_cases(input_wb, before_rows)
            completeness_total = len(completeness_cases)
            completeness_failed = sum(1 for c in completeness_cases if not c.ok)
            missing_codes_by_sheet = completeness_summary.get("missingCodesBySheet") or {}
            derived_column_coverage = _build_derived_column_coverage(input_wb, ui_before.get("headers") or [])
            _write_json(out_dir / "completeness_cases.json", [c.__dict__ for c in completeness_cases])
            _write_json(out_dir / "completeness_summary.json", completeness_summary)
            _write_json(out_dir / "missing_codes_by_sheet.json", missing_codes_by_sheet)
            _write_json(out_dir / "derived_column_coverage.json", derived_column_coverage)
        except Exception as e:
            completeness_summary = {"error": str(e)}
            completeness_total = 1
            completeness_failed = 1

    derived_unmapped_cols_total = 0
    derived_missing_ui_cols_total = 0
    try:
        for s in (derived_column_coverage.get("sheets") or {}).values():
            derived_unmapped_cols_total += int(s.get("unmappedColumnsWithValues") or 0)
            derived_missing_ui_cols_total += int(s.get("missingUiColumns") or 0)
    except Exception:
        derived_unmapped_cols_total = 0
        derived_missing_ui_cols_total = 0

    lookup_rows: List[Dict[str, Any]] = []
    if isinstance(before_rows, list):
        lookup_rows.extend(before_rows)
    if isinstance(after_rows, list):
        lookup_rows.extend(after_rows)
    action_export_checks = _action_export_checks(action_results, lookup_rows, export_wb)
    try:
        _write_json(out_dir / "action_export_checks.json", action_export_checks)
    except Exception:
        pass

    ok = (
        ui_before_ok
        and ui_after_ok
        and export_wb is not None
        and input_wb is not None
        and not missing_before
        and not mismatches_before
        and not missing_export
        and not mismatches_export
        and completeness_failed == 0
        and all(
            (derived_column_coverage.get("sheets") or {}).get(s, {}).get("unmappedColumnsWithValues", 0) == 0
            for s in ["批发", "零售", "住宿", "餐饮"]
        )
        and all(
            (derived_column_coverage.get("sheets") or {}).get(s, {}).get("missingUiColumns", 0) == 0
            for s in ["批发", "零售", "住宿", "餐饮"]
        )
        and all(s.get("status") != "fail" for s in steps)
        and all(r.get("ok") is not False for r in action_results)
        and all(r.get("ok") is not False for r in action_persist)
    )
    status = "PASS" if ok else "FAIL"

    import_log = ""
    try:
        import_log = "\n".join((import_events.get("items") or [])[:400])
    except Exception:
        import_log = ""

    shots_items: List[str] = []
    for p in screenshots[:30]:
        ps = _safe(p).replace("\\", "/")
        cap = _safe(Path(p).name)
        shots_items.append(
            "<div class=\"shot\">"
            f"<a class=\"shot-link\" href=\"{ps}\" data-cap=\"{cap}\">"
            f"<img class=\"thumb\" src=\"{ps}\" alt=\"shot\" loading=\"lazy\"/>"
            "</a>"
            f"<div class=\"cap mono\">{cap}</div>"
            "</div>"
        )
    shots_html = "".join(shots_items)

    conclusion = "✅ 全量校验通过（导入一致性 + 修改覆盖 + 导出一致性）" if ok else "❌ 存在不符合预期项：请优先查看【执行步骤与不符合预期项】与两段一致性差异表；所有改动建议也写在 report 内。"

    def _steps_html() -> str:
        if not steps:
            return "<p class='warn'>未生成 steps.json（脚本可能在早期退出）</p>"
        rows = []
        for s in steps:
            st = str(s.get("status") or "")
            klass = "ok" if st == "pass" else ("bad" if st == "fail" else "warn")
            rows.append(
                "<tr>"
                f"<td class='mono'>{_safe(str(s.get('ts') or ''))}</td>"
                f"<td class='mono'>{_safe(str(s.get('name') or ''))}</td>"
                f"<td class='{klass}'>{_safe(st)}</td>"
                f"<td>{_safe(str(s.get('detail') or ''))}</td>"
                f"<td>{_safe(str(s.get('reproduce') or ''))}</td>"
                "</tr>"
            )
        return (
            "<div class='table-wrap'><table><thead><tr>"
            "<th>时间</th><th>步骤</th><th>结果</th><th>不符合预期/原因</th><th>如何复现</th>"
            "</tr></thead><tbody>"
            + "".join(rows)
            + "</tbody></table></div>"
        )

    def _actions_html() -> str:
        if not action_results:
            return "<p class='warn'>未执行修改动作或未生成 actions_result.json</p>"
        persist_map: Dict[str, Dict[str, Any]] = {}
        for p in action_persist:
            k = f"{p.get('creditCode')}|{p.get('field')}|{p.get('i')}"
            persist_map[k] = p
        rows = []
        for r in action_results:
            okv = bool(r.get("ok"))
            klass = "ok" if okv else "bad"
            k = f"{r.get('creditCode')}|{r.get('field')}|{r.get('i')}"
            pv = persist_map.get(k) or {}
            v = r.get("value")
            v_s = "" if v is None else str(v)
            ui_s = "" if r.get("uiValue") is None else str(r.get("uiValue"))
            pv_s = "" if pv.get("uiValue") is None else str(pv.get("uiValue"))
            rows.append(
                "<tr>"
                f"<td class='mono'>{_safe(str(r.get('i') or ''))}</td>"
                f"<td class='mono'>{_safe(str(r.get('creditCode') or ''))}</td>"
                f"<td>{_safe(str(r.get('field') or ''))}</td>"
                f"<td class='mono'>{_safe(v_s)}</td>"
                f"<td class='mono'>{_safe(ui_s)}</td>"
                f"<td class='mono'>{_safe(pv_s)}</td>"
                f"<td class='{klass}'>{'PASS' if okv else 'FAIL'}</td>"
                f"<td>{_safe(str(r.get('reason') or ''))}</td>"
                f"<td class='mono'>{_safe(str(r.get('error') or ''))}</td>"
                "</tr>"
            )
        return (
            "<div class='table-wrap'><table><thead><tr>"
            "<th>#</th><th>统一社会信用代码</th><th>字段</th><th>新值</th><th>UI(立即)</th><th>UI(刷新后)</th><th>结果</th><th>覆盖场景</th><th>错误</th>"
            "</tr></thead><tbody>"
            + "".join(rows)
            + "</tbody></table></div>"
        )

    def _completeness_html() -> str:
        if input_wb is None or not ui_before_ok:
            return "<p class='warn'>输入 Excel 或 UI 抽取失败：未执行完整性断言</p>"

        if not completeness_cases and not missing_codes_by_sheet:
            return "<p class='warn'>未生成完整性数据（可能是衍生 Sheet 映射为空）</p>"

        pinned_cc = "914401007RDD76M0RF"
        pinned_fields = ["销售额;上年-本月", "商品销售额;上年-本月"]
        pinned = [c for c in completeness_cases if c.credit_code.upper() == pinned_cc and c.field in pinned_fields]
        fail_cases = [c for c in completeness_cases if not c.ok]
        show_fails = fail_cases[:200]

        def _row(c: CompletenessCase) -> str:
            klass = "ok" if c.ok else "bad"
            return (
                "<tr>"
                f"<td class='mono'>{_safe(c.sheet)}</td>"
                f"<td class='mono'>{_safe(c.credit_code)}</td>"
                f"<td>{_safe(c.name)}</td>"
                f"<td>{_safe(c.field)}</td>"
                f"<td class='mono'>{_safe(str(c.expected))}</td>"
                f"<td class='mono'>{_safe(str(c.actual))}</td>"
                f"<td class='{klass}'>{'PASS' if c.ok else 'FAIL'}</td>"
                f"<td>{_safe(c.reason)}</td>"
                f"<td>{_safe(c.reproduce)}</td>"
                "</tr>"
            )

        pinned_html = ""
        if pinned:
            pinned_html = (
                "<p class='ok'>✅ 关键案例（用户指定）</p>"
                "<div class='table-wrap'><table><thead><tr>"
                "<th>Sheet</th><th>统一社会信用代码</th><th>企业</th><th>字段</th><th>Excel</th><th>明细表</th><th>结果</th><th>原因</th><th>如何复现</th>"
                "</tr></thead><tbody>"
                + "".join(_row(c) for c in pinned[:1])
                + "</tbody></table></div>"
            )
        else:
            pinned_html = (
                "<p class='bad'>❌ 关键案例未命中（未在完整性断言数据中找到该企业字段）</p>"
                "<p class='warn'>复现：导入后在明细表搜索 914401007RDD76M0RF，确认列名是否为“商品销售额;上年-本月”。</p>"
            )

        fail_html = ""
        if not fail_cases:
            fail_html = "<p class='ok'>✅ 字段级完整性断言：未发现失败</p>"
        else:
            more = ""
            if len(fail_cases) > len(show_fails):
                more = f"<p class='warn'>仅展示前 {len(show_fails)} 条失败，完整列表见 completeness_cases.json。</p>"
            fail_html = (
                f"<p class='bad'>❌ 字段级完整性断言失败：{len(fail_cases)}/{len(completeness_cases)}</p>"
                + more
                + "<div class='table-wrap'><table><thead><tr>"
                "<th>Sheet</th><th>统一社会信用代码</th><th>企业</th><th>字段</th><th>Excel</th><th>明细表</th><th>结果</th><th>原因</th><th>如何复现</th>"
                "</tr></thead><tbody>"
                + "".join(_row(c) for c in show_fails)
                + "</tbody></table></div>"
            )

        missing_total = sum(len(v) for v in (missing_codes_by_sheet or {}).values())
        missing_html = f"<p class='{'ok' if missing_total == 0 else 'bad'}'>企业覆盖缺失：{missing_total}</p>"

        return pinned_html + "<div style='height:12px'></div>" + missing_html + "<div style='height:12px'></div>" + fail_html

    def _missing_codes_html() -> str:
        if not missing_codes_by_sheet:
            return "<p class='warn'>未生成缺失企业列表</p>"
        parts: List[str] = []
        for sheet_name in ["批发", "零售", "住宿", "餐饮"]:
            codes = missing_codes_by_sheet.get(sheet_name) or []
            sample = codes[:30]
            parts.append(
                "<div class='card' style='margin-top:12px;'>"
                f"<h2>Sheet「{_safe(sheet_name)}」缺失企业：{_safe(str(len(codes)))}</h2>"
                + ("<p class='ok'>✅ 无缺失</p>" if not codes else "<ul>" + "".join(f"<li class='mono'>{_safe(x)}</li>" for x in sample) + "</ul>")
                + (f"<p class='warn'>仅展示前 {len(sample)} 个，完整见 missing_codes_by_sheet.json。</p>" if len(codes) > len(sample) else "")
                + "</div>"
            )
        return "".join(parts)

    def _derived_coverage_html() -> str:
        sheets = (derived_column_coverage.get("sheets") or {}) if isinstance(derived_column_coverage, dict) else {}
        if not sheets:
            return "<p class='warn'>未生成“衍生 Sheet 列覆盖”信息</p>"
        rows: List[str] = []
        for sheet_name in ["批发", "零售", "住宿", "餐饮"]:
            s = sheets.get(sheet_name) or {}
            items = s.get("items") or []
            for it in items[:200]:
                mapped = bool(it.get("uiField"))
                ui_present = bool(it.get("uiPresent"))
                status_text = "OK" if (mapped and ui_present) else ("MISSING_UI_COL" if mapped else "UNMAPPED")
                klass = "ok" if status_text == "OK" else "bad"
                header = str(it.get("excelHeader") or "")
                nth = int(it.get("nth") or 1)
                nth_txt = f"#{nth}" if nth > 1 else ""
                rows.append(
                    "<tr>"
                    f"<td class='mono'>{_safe(sheet_name)}</td>"
                    f"<td class='mono'>{_safe(header + nth_txt)}</td>"
                    f"<td class='mono'>{_safe(str(it.get('nonEmpty') or ''))}</td>"
                    f"<td class='mono'>{_safe(str(it.get('uiField') or ''))}</td>"
                    f"<td class='{klass}'>{_safe(status_text)}</td>"
                    f"<td class='mono'>{_safe(str(it.get('examples') or ''))}</td>"
                    f"<td class='mono'>{_safe(str(it.get('suggestedUiField') or ''))}</td>"
                    "</tr>"
                )
        return (
            "<div class='table-wrap'><table><thead><tr>"
            "<th>Sheet</th><th>Excel 列</th><th>有值数量</th><th>映射到明细表列</th><th>状态</th><th>示例</th><th>建议 UI 列名</th>"
            "</tr></thead><tbody>"
            + "".join(rows)
            + "</tbody></table></div>"
        )

    def _action_export_checks_html() -> str:
        if export_wb is None or not ui_after_ok:
            return "<p class='warn'>导出 Excel 或 UI 抽取失败：未执行“修改动作 → 导出回归校验”</p>"
        if not action_export_checks:
            return "<p class='warn'>无修改动作或无法生成回归校验</p>"
        fails = [x for x in action_export_checks if not x.get("ok")]
        rows = []
        for x in action_export_checks[:200]:
            okv = bool(x.get("ok"))
            klass = "ok" if okv else "bad"
            rows.append(
                "<tr>"
                f"<td class='mono'>{_safe(str(x.get('creditCode') or ''))}</td>"
                f"<td>{_safe(str(x.get('field') or ''))}</td>"
                f"<td class='mono'>{_safe(str(x.get('sheet') or ''))}</td>"
                f"<td class='mono'>{_safe(str(x.get('excelField') or ''))}</td>"
                f"<td class='mono'>{_safe(str(x.get('expected') or ''))}</td>"
                f"<td class='mono'>{_safe(str(x.get('actual') or ''))}</td>"
                f"<td class='{klass}'>{'PASS' if okv else 'FAIL'}</td>"
                f"<td>{_safe(str(x.get('reason') or ''))}</td>"
                "</tr>"
            )
        more = ""
        if len(action_export_checks) > 200:
            more = "<p class='warn'>仅展示前 200 条，完整见 action_export_checks.json。</p>"
        return (
            f"<p class='{'ok' if not fails else 'bad'}'>"
            + ("✅ 全部 PASS" if not fails else f"❌ FAIL：{len(fails)}/{len(action_export_checks)}")
            + "</p>"
            + more
            + "<div class='table-wrap'><table><thead><tr>"
            "<th>统一社会信用代码</th><th>字段</th><th>Sheet</th><th>导出列</th><th>期望</th><th>实际</th><th>结果</th><th>原因</th>"
            "</tr></thead><tbody>"
            + "".join(rows)
            + "</tbody></table></div>"
        )

    def _tab_counts_html() -> str:
        items = tab_counts.get("items") if isinstance(tab_counts, dict) else None
        if not isinstance(items, list) or not items:
            return "<p class='warn'>未生成 tab 行数统计</p>"
        rows = []
        for it in items:
            okv = bool(it.get("ok"))
            klass = "ok" if okv else "bad"
            rows.append(
                "<tr>"
                f"<td class='mono'>{_safe(str(it.get('tab') or ''))}</td>"
                f"<td class='{klass}'>{'PASS' if okv else 'FAIL'}</td>"
                f"<td class='mono'>{_safe(str(it.get('rows') or ''))}</td>"
                f"<td class='mono'>{_safe(str(it.get('totalText') or ''))}</td>"
                f"<td class='mono'>{_safe(str(it.get('error') or ''))}</td>"
                "</tr>"
            )
        return (
            "<div class='table-wrap'><table><thead><tr>"
            "<th>Tab</th><th>结果</th><th>表格行数</th><th>底部总数文案</th><th>错误</th>"
            "</tr></thead><tbody>"
            + "".join(rows)
            + "</tbody></table></div>"
        )

    html = f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Northstar E2E (agent-browser) - {status}</title>
  <style>
    :root {{
      --bg: #0b1220;
      --panel: #0f1b33;
      --text: #e6edf3;
      --muted: #a6b3c3;
      --ok: #4ade80;
      --bad: #fb7185;
      --warn: #fbbf24;
      --border: rgba(255,255,255,.10);
      --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      --sans: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial;
    }}
    body {{ margin: 0; background: var(--bg); color: var(--text); font-family: var(--sans); }}
    .wrap {{ max-width: 1200px; margin: 0 auto; padding: 28px 18px 48px; }}
    .title {{ display: flex; align-items: baseline; justify-content: space-between; gap: 12px; }}
    h1 {{ font-size: 22px; margin: 0; }}
    .pill {{ padding: 4px 10px; border-radius: 999px; font-weight: 700; font-size: 12px; border: 1px solid var(--border); }}
    .pill.ok {{ color: #052e15; background: var(--ok); border-color: rgba(74,222,128,.35); }}
    .pill.bad {{ color: #2b0b11; background: var(--bad); border-color: rgba(251,113,133,.35); }}
    .grid {{ display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 16px; }}
    .card {{ background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 14px 14px 12px; }}
    .card h2 {{ font-size: 14px; margin: 0 0 8px; color: var(--muted); font-weight: 700; }}
    .kv {{ display: grid; grid-template-columns: 160px 1fr; gap: 6px 10px; font-size: 13px; }}
    .kv div {{ padding: 2px 0; }}
    .k {{ color: var(--muted); }}
    .v {{ font-family: var(--mono); }}
    .mono {{ font-family: var(--mono); }}
    .ok {{ color: var(--ok); font-weight: 700; }}
    .bad {{ color: var(--bad); font-weight: 700; }}
    .warn {{ color: var(--warn); font-weight: 700; }}
    .section {{ margin-top: 14px; }}
    .section h3 {{ margin: 0 0 10px; font-size: 16px; }}
    a {{ color: #93c5fd; text-decoration: none; }}
    a:hover {{ text-decoration: underline; }}
    ul {{ margin: 8px 0 0 18px; }}
    li {{ margin: 3px 0; }}
    .table-wrap {{ overflow: auto; border: 1px solid var(--border); border-radius: 10px; }}
    table {{ border-collapse: collapse; width: 100%; min-width: 900px; font-size: 12px; }}
    th, td {{ border-bottom: 1px solid var(--border); padding: 8px 10px; text-align: left; vertical-align: top; }}
    th {{ position: sticky; top: 0; background: rgba(15,27,51,.95); backdrop-filter: blur(8px); }}
    .shots {{ display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }}
    @media (max-width: 1100px) {{ .shots {{ grid-template-columns: repeat(3, 1fr); }} }}
    @media (max-width: 820px) {{ .shots {{ grid-template-columns: repeat(2, 1fr); }} }}
    @media (max-width: 520px) {{ .shots {{ grid-template-columns: 1fr; }} }}
    .shot {{ border: 1px solid var(--border); border-radius: 10px; overflow: hidden; background: rgba(255,255,255,.02); }}
    .shot img.thumb {{ width: 100%; height: 120px; object-fit: cover; display: block; }}
    .shot .cap {{ padding: 8px 10px; font-size: 12px; color: var(--muted); }}
    pre {{ background: rgba(255,255,255,.04); border: 1px solid var(--border); border-radius: 10px; padding: 10px 12px; overflow: auto; }}

    .lightbox {{ position: fixed; inset: 0; background: rgba(0,0,0,.76); display: none; z-index: 9999; }}
    .lightbox.open {{ display: block; }}
    .lightbox-inner {{ position: absolute; inset: 28px 20px; display: grid; grid-template-rows: auto 1fr auto; gap: 10px; }}
    .lightbox-top {{ display: flex; align-items: center; justify-content: space-between; gap: 10px; }}
    .lightbox-title {{ color: var(--muted); font-size: 12px; font-family: var(--mono); }}
    .lightbox-btn {{ border: 1px solid rgba(255,255,255,.18); background: rgba(255,255,255,.06); color: var(--text); border-radius: 10px; padding: 8px 10px; cursor: pointer; }}
    .lightbox-btn:hover {{ background: rgba(255,255,255,.10); }}
    .lightbox-body {{ position: relative; border: 1px solid rgba(255,255,255,.14); border-radius: 12px; overflow: hidden; background: rgba(0,0,0,.20); }}
    .lightbox-body img {{ width: 100%; height: 100%; object-fit: contain; display: block; }}
    .lightbox-nav {{ position: absolute; inset: 0; display: flex; align-items: center; justify-content: space-between; pointer-events: none; }}
    .lightbox-nav button {{ pointer-events: all; width: 44px; height: 44px; border-radius: 999px; }}
    .lightbox-bottom {{ color: var(--muted); font-size: 12px; font-family: var(--mono); }}
  </style>
</head>
<body>
  <div class="wrap">
    <div class="title">
      <h1>Northstar E2E（agent-browser）</h1>
      <div class="pill {'ok' if ok else 'bad'}">{status}</div>
    </div>
    <div class="grid">
      <div class="card">
        <h2>运行信息</h2>
        <div class="kv">
          <div class="k">时间</div><div class="v">{_safe(now)}</div>
          <div class="k">BASE_URL</div><div class="v">{_safe(args.base_url)}</div>
          <div class="k">输入 Excel</div><div class="v"><a href="{_safe(args.input_xlsx)}">{_safe(args.input_xlsx)}</a></div>
          <div class="k">导出 Excel</div><div class="v"><a href="{_safe(args.export_xlsx)}">{_safe(args.export_xlsx)}</a></div>
          <div class="k">抽取(导入后)</div><div class="v">{_safe(str(started_at))}</div>
          <div class="k">抽取(修改后)</div><div class="v">{_safe(str(ended_at))}</div>
        </div>
      </div>
      <div class="card">
        <h2>附件</h2>
        <ul>
          <li><a class="mono" href="{_safe(args.ui_before)}">ui_companies_before.json</a></li>
          <li><a class="mono" href="{_safe(args.ui_after)}">ui_companies_after.json</a></li>
          <li><a class="mono" href="{_safe(args.import_events)}">import_events.json</a></li>
          <li><a class="mono" href="{_safe(args.tab_counts)}">tab_counts.json</a></li>
          <li><a class="mono" href="{_safe(args.console)}">browser_console.txt</a></li>
          <li><a class="mono" href="{_safe(args.errors)}">browser_errors.txt</a></li>
          <li><a class="mono" href="{_safe(args.trace)}">trace.zip</a></li>
          <li><a class="mono" href="{_safe(args.video)}">run.webm</a></li>
        </ul>
      </div>
    </div>

    <div class="section">
      <h3>执行步骤与不符合预期项</h3>
      <div class="card">
        <h2>步骤明细</h2>
        {_steps_html()}
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>不符合预期项总览</h2>
        {_issues_summary(missing_before, mismatches_before, missing_export, mismatches_export, action_results, action_persist, completeness_failed, completeness_total, derived_unmapped_cols_total, derived_missing_ui_cols_total)}
        <p class="warn">复现入口：打开 <span class="mono">{_safe(args.base_url)}</span> → 导入 → 明细表搜索信用代码 → 修改/对照 → 导出后打开 Excel 对照。</p>
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>修改动作覆盖</h2>
        <p class="warn">复现：在首页输入框按信用代码搜索，修改对应字段输入框，失焦（blur）触发自动保存，然后导出检查。</p>
        {_actions_html()}
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>Tab 行数与总数</h2>
        <p class="warn">目的：覆盖“全部/批发/零售/住宿/餐饮”tab 切换与数据是否加载。</p>
        {_tab_counts_html()}
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>UI 抽取状态</h2>
        <div class="kv">
          <div class="k">导入后抽取</div><div class="v">{'OK' if ui_before_ok else 'FAIL'}</div>
          <div class="k">导入后错误</div><div class="v">{_safe(str(ui_before_error))}</div>
          <div class="k">修改后抽取</div><div class="v">{'OK' if ui_after_ok else 'FAIL'}</div>
          <div class="k">修改后错误</div><div class="v">{_safe(str(ui_after_error))}</div>
          <div class="k">输入 Excel 打开</div><div class="v">{'OK' if input_wb is not None else 'FAIL'}</div>
          <div class="k">导出 Excel 打开</div><div class="v">{'OK' if export_wb is not None else 'FAIL'}</div>
        </div>
      </div>
    </div>

    <div class="section">
      <h3>明细表数据完整性案例（输入 Excel → 明细表）</h3>
      <div class="card">
        <h2>字段级完整性（大量案例）</h2>
        <p class="warn">完整列表：<a class="mono" href="completeness_cases.json">completeness_cases.json</a> / <a class="mono" href="completeness_summary.json">completeness_summary.json</a></p>
        {_completeness_html()}
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>缺失企业（Excel 有，但明细表缺）</h2>
        <p class="warn">完整列表：<a class="mono" href="missing_codes_by_sheet.json">missing_codes_by_sheet.json</a></p>
        {_missing_codes_html()}
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>衍生 Sheet 列覆盖（有值列 → UI 列是否存在）</h2>
        <p class="warn">完整列表：<a class="mono" href="derived_column_coverage.json">derived_column_coverage.json</a></p>
        {_derived_coverage_html()}
      </div>
    </div>

    <div class="section">
      <h3>改动建议（仅写在报告内）</h3>
      <div class="card">
        <h2>建议汇总</h2>
        {_suggestions(ui_before_ok, ui_after_ok, before_rows if isinstance(before_rows, list) else [], after_rows if isinstance(after_rows, list) else [], input_wb, export_wb, missing_before, mismatches_before, missing_export, mismatches_export)}
      </div>
    </div>

    <div class="section">
      <h3>导入一致性（明细表 vs 输入 Excel）</h3>
      <div class="card">
        <h2>覆盖检查</h2>
        {_summarize_missing(missing_before)}
        <p class="warn">复现：打开输入 Excel 的对应 Sheet（来源表），在“统一社会信用代码”列定位企业，对比明细表同字段显示。</p>
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>字段一致性</h2>
        {_summarize_mismatches(mismatches_before)}
        <p class="warn">复现：首页明细表搜索信用代码，打开“列”展示相关字段，对照输入 Excel 的同字段值。</p>
      </div>
    </div>

    <div class="section">
      <h3>导出一致性（导出 Excel vs 明细表[修改后]）</h3>
      <div class="card">
        <h2>覆盖检查</h2>
        {_summarize_missing(missing_export)}
        <p class="warn">复现：点击“导出”下载 xlsx，打开后在对应 Sheet 用“统一社会信用代码”定位行，对照明细表修改后的值。</p>
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>字段一致性</h2>
        {_summarize_mismatches(mismatches_export)}
        <p class="warn">复现：先确认修改动作已保存（输入框失焦），再导出并对照字段。</p>
      </div>
      <div class="card" style="margin-top: 12px;">
        <h2>修改动作 → 导出回归校验</h2>
        <p class="warn">完整列表：<a class="mono" href="action_export_checks.json">action_export_checks.json</a></p>
        {_action_export_checks_html()}
      </div>
    </div>

    <div class="section">
      <h3>导入日志（UI）</h3>
      <div class="card">
        <h2>进度事件</h2>
        <p class="mono">count={_safe(str(import_events.get('count', '')))}</p>
        <pre>{_safe(import_log)}</pre>
      </div>
    </div>

    <div class="section">
      <h3>浏览器日志</h3>
      <div class="grid">
        <div class="card">
          <h2>Console</h2>
          <pre>{_safe(_read_text(args.console)[:20000])}</pre>
        </div>
        <div class="card">
          <h2>Errors</h2>
          <pre>{_safe(_read_text(args.errors)[:20000])}</pre>
        </div>
      </div>
    </div>

    <div class="section">
      <h3>关键截图</h3>
      <div class="shots">
        {shots_html}
      </div>
    </div>

    <div class="section">
      <h3>结论</h3>
      <div class="card">
        <p>{_safe(conclusion)}</p>
      </div>
    </div>
  </div>

  <div id="lightbox" class="lightbox" aria-hidden="true">
    <div class="lightbox-inner">
      <div class="lightbox-top">
        <div id="lightbox-title" class="lightbox-title">screenshot</div>
        <div style="display:flex; gap:8px;">
          <button id="lightbox-open" class="lightbox-btn" type="button">在新标签页打开</button>
          <button id="lightbox-close" class="lightbox-btn" type="button">关闭 (Esc)</button>
        </div>
      </div>
      <div class="lightbox-body" id="lightbox-body">
        <img id="lightbox-img" alt="preview" />
        <div class="lightbox-nav">
          <button id="lightbox-prev" class="lightbox-btn" type="button">‹</button>
          <button id="lightbox-next" class="lightbox-btn" type="button">›</button>
        </div>
      </div>
      <div id="lightbox-cap" class="lightbox-bottom"></div>
    </div>
  </div>

  <script>
    (() => {{
      const links = Array.from(document.querySelectorAll('.shot-link'));
      const lb = document.getElementById('lightbox');
      const img = document.getElementById('lightbox-img');
      const title = document.getElementById('lightbox-title');
      const cap = document.getElementById('lightbox-cap');
      const btnClose = document.getElementById('lightbox-close');
      const btnOpen = document.getElementById('lightbox-open');
      const btnPrev = document.getElementById('lightbox-prev');
      const btnNext = document.getElementById('lightbox-next');
      const body = document.getElementById('lightbox-body');
      if (!lb || !img || links.length === 0) return;

      let idx = 0;
      function setIndex(i) {{
        if (i < 0) i = links.length - 1;
        if (i >= links.length) i = 0;
        idx = i;
        const a = links[idx];
        const href = a.getAttribute('href') || '';
        const c = a.getAttribute('data-cap') || '';
        img.setAttribute('src', href);
        title.textContent = c || href;
        cap.textContent = href;
        btnOpen.onclick = () => window.open(href, '_blank');
      }}

      function openAt(i) {{
        setIndex(i);
        lb.classList.add('open');
        lb.setAttribute('aria-hidden', 'false');
      }}
      function close() {{
        lb.classList.remove('open');
        lb.setAttribute('aria-hidden', 'true');
        img.setAttribute('src', '');
      }}

      links.forEach((a, i) => {{
        a.addEventListener('click', (e) => {{
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
          e.preventDefault();
          openAt(i);
        }});
      }});

      btnClose && btnClose.addEventListener('click', close);
      btnPrev && btnPrev.addEventListener('click', () => setIndex(idx - 1));
      btnNext && btnNext.addEventListener('click', () => setIndex(idx + 1));
      lb.addEventListener('click', (e) => {{
        if (e.target === lb) close();
      }});
      body && body.addEventListener('dblclick', close);
      document.addEventListener('keydown', (e) => {{
        if (!lb.classList.contains('open')) return;
        if (e.key === 'Escape') close();
        if (e.key === 'ArrowLeft') setIndex(idx - 1);
        if (e.key === 'ArrowRight') setIndex(idx + 1);
      }});
    }})();
  </script>
</body>
</html>
"""

    Path(args.out).write_text(html, encoding="utf-8")

    if not ok:
        raise SystemExit(1)


if __name__ == "__main__":
    main()

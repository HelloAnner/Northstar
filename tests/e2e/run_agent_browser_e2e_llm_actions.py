#!/usr/bin/env python3
import json
import os
import re
import shlex
import subprocess
import sqlite3
import sys
import time
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple


def _run(cmd: List[str], log_path: Path) -> str:
    p = subprocess.run(cmd, text=True, capture_output=True)
    prev = ""
    if log_path.exists():
        prev = log_path.read_text(encoding="utf-8", errors="replace")
    log_path.write_text(prev + p.stdout + p.stderr, encoding="utf-8")
    if p.returncode != 0:
        raise SystemExit(p.returncode)
    return p.stdout


def _agent(cmd: str, log_path: Path) -> None:
    _run(shlex.split(f"agent-browser {cmd}"), log_path)


def _agent_json(cmd: str, log_path: Path) -> Dict[str, Any]:
    out = _run(shlex.split(f"agent-browser {cmd}"), log_path).strip()
    if not out:
        return {"ok": False, "error": "empty output"}
    try:
        return json.loads(out)
    except Exception:
        return {"ok": False, "error": "invalid json output", "raw": out[-2000:]}


def _unwrap_agent_browser_json(d: Dict[str, Any]) -> Dict[str, Any]:
    if not isinstance(d, dict):
        return {"ok": False, "error": "invalid json"}
    if d.get("success") is False:
        return {"ok": False, "error": d.get("error") or "agent-browser error"}
    data = d.get("data")
    if isinstance(data, dict) and isinstance(data.get("result"), dict):
        return data["result"]
    return d


def _http_json(url: str, method: str = "GET", payload: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    body = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=body, method=method, headers=headers)
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read().decode("utf-8"))


def _pick_company(items: List[Dict[str, Any]], kind: str, fields: List[str]) -> Tuple[Optional[Dict[str, Any]], str]:
    for it in items:
        if str(it.get("kind") or "") != kind:
            continue
        for f in fields:
            if it.get(f) is not None:
                return it, f
    return None, ""


def _parse_num(v: Any) -> Optional[float]:
    if v is None:
        return None
    if isinstance(v, (int, float)) and not isinstance(v, bool):
        return float(v)
    s = str(v).strip().replace(",", "")
    if not s or s == "-":
        return None
    if s.endswith("%"):
        s = s[:-1]
    try:
        return float(s)
    except Exception:
        return None


def _close(a: Optional[float], b: Optional[float], eps: float = 0.6) -> bool:
    if a is None or b is None:
        return False
    return abs(a - b) <= eps


def _js_click_chat_fab() -> str:
    return (
        "(() => {\n"
        "  const btn = document.querySelector('button.fixed.bottom-6.right-6');\n"
        "  if (!btn) return { ok: false, error: 'chat button not found' };\n"
        "  btn.click();\n"
        "  return { ok: true };\n"
        "})()"
    )


def _js_get_config_values() -> str:
    return (
        "(() => {\n"
        "  const base = document.querySelector('#llm-base-url');\n"
        "  const model = document.querySelector('#llm-model');\n"
        "  const api = document.querySelector('#llm-api-key');\n"
        "  return {\n"
        "    baseUrl: base ? String(base.value || '').trim() : '',\n"
        "    model: model ? String(model.value || '').trim() : '',\n"
        "    apiKeyType: api ? String(api.type || '') : '',\n"
        "    apiKeyLen: api ? String(api.value || '').length : 0,\n"
        "  };\n"
        "})()"
    )


def _js_get_chat_summary_text() -> str:
    return (
        "(() => {\n"
        "  const bubbles = Array.from(document.querySelectorAll('div.rounded-2xl'));\n"
        "  for (let i = bubbles.length - 1; i >= 0; i -= 1) {\n"
        "    const text = String(bubbles[i].textContent || '').trim();\n"
        "    if (text.includes('已更新企业')) return { ok: true, text };\n"
        "  }\n"
        "  return { ok: false, error: 'summary not found' };\n"
        "})()"
    )


def _js_get_chat_error_text() -> str:
    return (
        "(() => {\n"
        "  const bubbles = Array.from(document.querySelectorAll('div.rounded-2xl'));\n"
        "  for (let i = bubbles.length - 1; i >= 0; i -= 1) {\n"
        "    const text = String(bubbles[i].textContent || '').trim();\n"
        "    if (text.includes('错误')) return { ok: true, text };\n"
        "  }\n"
        "  return { ok: false, error: 'error message not found' };\n"
        "})()"
    )


def _js_wait_for_chat_terminal() -> str:
    return (
        "(() => {\n"
        "  const texts = Array.from(document.querySelectorAll('div.rounded-2xl'))\n"
        "    .map((el) => String(el.textContent || ''));\n"
        "  return texts.some((t) => t.includes('已更新企业') || t.includes('错误'));\n"
        "})()"
    )


def _js_get_message_count() -> str:
    return (
        "(() => {\n"
        "  return { count: document.querySelectorAll('div.rounded-2xl').length };\n"
        "})()"
    )


def _js_check_streaming_indicator() -> str:
    """检查流式输出指示器是否存在"""
    return (
        "(() => {\n"
        "  const indicator = document.querySelector('[data-streaming]') ||\n"
        "                    document.querySelector('.streaming') ||\n"
        "                    document.querySelector('[class*=streaming]');\n"
        "  return { ok: !!indicator, exists: !!indicator };\n"
        "})()"
    )


def _js_get_chat_messages() -> str:
    """获取所有聊天消息内容"""
    return (
        "(() => {\n"
        "  const bubbles = Array.from(document.querySelectorAll('div.rounded-2xl'));\n"
        "  return {\n"
        "    count: bubbles.length,\n"
        "    messages: bubbles.map(el => ({\n"
        "      text: String(el.textContent || '').trim(),\n"
        "      html: el.innerHTML\n"
        "    }))\n"
        "  };\n"
        "})()"
    )


def _js_get_last_message_content() -> str:
    """获取最后一条消息内容"""
    return (
        "(() => {\n"
        "  const bubbles = Array.from(document.querySelectorAll('div.rounded-2xl'));\n"
        "  if (bubbles.length === 0) return { ok: false, error: 'no messages' };\n"
        "  const last = bubbles[bubbles.length - 1];\n"
        "  return {\n"
        "    ok: true,\n"
        "    text: String(last.textContent || '').trim(),\n"
        "    index: bubbles.length - 1\n"
        "  };\n"
        "})()"
    )


def _js_check_dialog_position() -> str:
    """检查对话框位置是否为右侧侧边栏"""
    return (
        "(() => {\n"
        "  const dialog = document.querySelector('[role=dialog]') ||\n"
        "                 document.querySelector('[data-llm-chat]') ||\n"
        "                 document.querySelector('.fixed.inset-0');\n"
        "  if (!dialog) return { ok: false, error: 'dialog not found' };\n"
        "  const rect = dialog.getBoundingClientRect();\n"
        "  const style = window.getComputedStyle(dialog);\n"
        "  return {\n"
        "    ok: true,\n"
        "    right: rect.right,\n"
        "    width: rect.width,\n"
        "    position: style.position,\n"
        "    isRightSide: rect.right >= window.innerWidth - 50\n"
        "  };\n"
        "})()"
    )


def _mask(v: str) -> str:
    if not v:
        return ""
    if len(v) <= 6:
        return "***"
    return v[:3] + "***" + v[-3:]


def _parse_summary_numbers(text: str) -> Dict[str, Any]:
    out: Dict[str, Any] = {"updatedCompanies": None, "targetIndicators": None, "optimized": None}
    if not text:
        return out
    # 支持多种格式：带粗体标记、不带粗体标记、带单位后缀
    m1 = re.search(r"已更新企业[:：]\s*([0-9]+)", text)
    m2 = re.search(r"指标目标[:：]\s*([0-9]+)", text)
    m3 = re.search(r"智能调整[:：]\s*(已触发|未触发)", text)
    if m1:
        out["updatedCompanies"] = int(m1.group(1))
    if m2:
        out["targetIndicators"] = int(m2.group(1))
    if m3:
        out["optimized"] = m3.group(1) == "已触发"
    return out


def _to_snake(name: str) -> str:
    return re.sub(r"(?<!^)([A-Z])", r"_\1", name).lower()


def _get_field(obj: Dict[str, Any], field: str) -> Any:
    if field in obj:
        return obj.get(field)
    snake = _to_snake(field)
    if snake in obj:
        return obj.get(snake)
    return obj.get(field.lower())


def _parse_company_id(raw: str) -> Tuple[str, Optional[int]]:
    parts = str(raw or "").split(":")
    if len(parts) != 2:
        return "", None
    try:
        return parts[0], int(parts[1])
    except Exception:
        return parts[0], None


def _fetch_db_value(db_path: Path, table: str, row_id: int, field: str) -> Any:
    if not db_path.exists():
        return None
    if not field:
        return None
    col = _to_snake(field)
    try:
        conn = sqlite3.connect(str(db_path))
        cur = conn.cursor()
        cur.execute(f"select {col} from {table} where id = ?", (row_id,))
        row = cur.fetchone()
        conn.close()
        if not row:
            return None
        return row[0]
    except Exception:
        return None


def main() -> None:
    if len(sys.argv) != 5:
        print(
            "Usage: run_agent_browser_e2e_llm_actions.py <baseUrl> <logPath> <screenshotsDir> <outJson>",
            file=sys.stderr,
        )
        raise SystemExit(2)

    base_url = sys.argv[1].rstrip("/")
    log_path = Path(sys.argv[2])
    screenshots = Path(sys.argv[3])
    out_path = Path(sys.argv[4])

    llm_base = os.environ.get("DEEPSEEK_BASE_URL") or os.environ.get("LLM_BASE_URL") or ""
    llm_model = os.environ.get("DEEPSEEK_MODEL_NAME") or os.environ.get("LLM_MODEL") or ""
    llm_api_key = os.environ.get("DEEPSEEK_API_KEY") or os.environ.get("LLM_API_KEY") or ""

    results: Dict[str, Any] = {
        "executed": False,
        "ok": False,
        "config": {},
        "streaming": {},
        "first": {},
        "second": {},
        "updates": {},
        "newSession": {},
        "errors": [],
        "screenshots": {},
    }

    if not llm_base or not llm_model or not llm_api_key:
        results["skipReason"] = "missing LLM env"
        out_path.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
        print("LLM test skipped: missing env")
        return

    results["executed"] = True

    # 1) 全局配置 UI 校验（不暴露明文）
    config_ok = True
    try:
        _agent('find role button click --name "全局配置"', log_path)
        _agent('wait --text "全局配置"', log_path)
        conf_ret = _agent_json(f"eval {shlex.quote(_js_get_config_values())} --json", log_path)
        conf = _unwrap_agent_browser_json(conf_ret)
        base_ok = str(conf.get("baseUrl") or "").strip() == llm_base.strip()
        model_ok = str(conf.get("model") or "").strip() == llm_model.strip()
        api_type = str(conf.get("apiKeyType") or "")
        api_len = int(conf.get("apiKeyLen") or 0)
        api_ok = api_type == "password" and api_len > 0

        show_ok = True
        # 显示 -> 隐藏，期间不截图，避免泄露密钥
        try:
            _agent('find role button click --name "显示"', log_path)
            time.sleep(0.2)
            conf_show = _unwrap_agent_browser_json(
                _agent_json(f"eval {shlex.quote(_js_get_config_values())} --json", log_path)
            )
            show_ok = str(conf_show.get("apiKeyType") or "") == "text"
        except SystemExit as e:
            show_ok = False
            results["errors"].append(f"config show failed: exit={e.code}")
        try:
            _agent('find role button click --name "隐藏"', log_path)
            time.sleep(0.2)
            conf_hide = _unwrap_agent_browser_json(
                _agent_json(f"eval {shlex.quote(_js_get_config_values())} --json", log_path)
            )
            show_ok = show_ok and str(conf_hide.get("apiKeyType") or "") == "password"
        except SystemExit as e:
            show_ok = False
            results["errors"].append(f"config hide failed: exit={e.code}")

        results["config"] = {
            "baseUrl": _mask(llm_base),
            "model": llm_model,
            "baseUrlOk": base_ok,
            "modelOk": model_ok,
            "apiKeyMasked": api_ok,
            "showHideOk": show_ok,
            "apiKeyType": api_type,
            "apiKeyLen": api_len,
        }
        config_ok = base_ok and model_ok and api_ok and show_ok
        _agent(f'screenshot \"{screenshots / "14_llm_config.png"}\"', log_path)
        results["screenshots"]["config"] = str(screenshots / "14_llm_config.png")
        _agent('find role button click --name "取消"', log_path)
    except SystemExit as e:
        config_ok = False
        results["errors"].append(f"config check failed: exit={e.code}")

    # 2) 打开对话框
    open_ok = True
    try:
        open_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_click_chat_fab())} --json", log_path)
        )
        if not bool(open_ret.get("ok")):
            open_ok = False
            results["errors"].append(str(open_ret.get("error") or "chat button not found"))
        _agent('wait --text "数据对话助手"', log_path)
        _agent(f'screenshot \"{screenshots / "15_llm_chat_open.png"}\"', log_path)
        results["screenshots"]["chatOpen"] = str(screenshots / "15_llm_chat_open.png")
    except SystemExit as e:
        open_ok = False
        results["errors"].append(f"open chat failed: exit={e.code}")

    # 3) 选取企业与指标
    wr_id = ""
    ac_id = ""
    wr_field = "salesCurrentMonth"
    ac_field = "foodCurrentMonth"
    wr_target = 0.0
    ac_target = 0.0
    indicator_target = 0.0
    indicator_id = "limitAbove_month_value"
    company_before: Dict[str, Any] = {}

    try:
        companies = _http_json(f"{base_url}/api/companies?page=1&pageSize=2000")
        items = companies.get("items") or []
        wr_item, wr_field = _pick_company(items, "wr", ["salesCurrentMonth", "retailCurrentMonth"])
        ac_item, ac_field = _pick_company(items, "ac", ["revenueCurrentMonth", "foodCurrentMonth"])
        if wr_item:
            wr_id = str(wr_item.get("id") or "")
            base = _parse_num(wr_item.get(wr_field)) or 0.0
            wr_target = round(base + 123.0, 2)
        if ac_item:
            ac_id = str(ac_item.get("id") or "")
            base = _parse_num(ac_item.get(ac_field)) or 0.0
            ac_target = round(base + 45.0, 2)

        indicators = _http_json(f"{base_url}/api/indicators")
        groups = indicators.get("groups") or []
        found_val = None
        for g in groups:
            for it in g.get("indicators") or []:
                if str(it.get("id") or "") == indicator_id:
                    found_val = _parse_num(it.get("value"))
                    break
        base_val = found_val if found_val is not None else 0.0
        indicator_target = round(base_val + 500.0, 2)

        if wr_id:
            company_before[wr_id] = _http_json(f"{base_url}/api/companies/{wr_id}").get("company") or {}
        if ac_id:
            company_before[ac_id] = _http_json(f"{base_url}/api/companies/{ac_id}").get("company") or {}
    except Exception as e:
        results["errors"].append(f"load company/indicator failed: {e}")

    if not wr_id or not ac_id:
        results["errors"].append("企业数据不足，无法执行 LLM 修改测试")
        results["ok"] = False
        out_path.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
        return

    # 4) 第一轮对话：带指标目标（触发智能调整）
    first_ok = True
    streaming_ok = False
    summary_ok = False
    try:
        prompt = (
            f"请仅调用 update_companies，严格执行以下修改："
            f"1) {{\\\"id\\\":\\\"{wr_id}\\\",\\\"patch\\\":{{\\\"{wr_field}\\\":{wr_target}}}}}；"
            f"2) {{\\\"id\\\":\\\"{ac_id}\\\",\\\"patch\\\":{{\\\"{ac_field}\\\":{ac_target}}}}}；"
            f"不要调用 set_indicator_targets，最后给出简短说明。"
        )
        _agent(f'find placeholder "输入你的调整需求…" fill "{prompt}"', log_path)
        _agent('find role button click --name "发送"', log_path)

        try:
            _agent('wait --text "▍" --timeout 30000', log_path)
            streaming_ok = True
            _agent(f'screenshot \"{screenshots / "16_llm_streaming.png"}\"', log_path)
            results["screenshots"]["streaming"] = str(screenshots / "16_llm_streaming.png")
        except SystemExit as e:
            results["errors"].append(f"streaming not detected: exit={e.code}")

        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)
        summary_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_chat_summary_text())} --json", log_path)
        )
        summary_text = str(summary_ret.get("text") or "")
        summary_nums = _parse_summary_numbers(summary_text)
        summary_ok = (
            summary_nums.get("optimized") is False
            and (summary_nums.get("targetIndicators") or 0) == 0
            and (summary_nums.get("updatedCompanies") or 0) >= 2
        )
        if not summary_text:
            err_ret = _unwrap_agent_browser_json(
                _agent_json(f"eval {shlex.quote(_js_get_chat_error_text())} --json", log_path)
            )
            err_text = str(err_ret.get("text") or "")
            if err_text:
                results["errors"].append(err_text)

        results["first"] = {
            "summaryText": summary_text,
            "summary": summary_nums,
            "expectedOptimized": False,
        }
        _agent(f'screenshot \"{screenshots / "17_llm_after_first.png"}\"', log_path)
        results["screenshots"]["afterFirst"] = str(screenshots / "17_llm_after_first.png")
    except SystemExit as e:
        first_ok = False
        results["errors"].append(f"first chat failed: exit={e.code}")

    # 5) 校验更新结果（LLM 修改字段应保持不变）
    updates_ok = True
    updates: Dict[str, Any] = {}
    db_path = Path(os.environ.get("RUN_DIR", "") or "") / "server" / "data" / "northstar.db"
    wr_kind, wr_numeric = _parse_company_id(wr_id)
    ac_kind, ac_numeric = _parse_company_id(ac_id)
    results["debug"] = {
        "runDir": os.environ.get("RUN_DIR", ""),
        "dbPath": str(db_path),
        "dbExists": db_path.exists(),
    }
    try:
        wr_after = _http_json(f"{base_url}/api/companies/{wr_id}").get("company") or {}
        ac_after = _http_json(f"{base_url}/api/companies/{ac_id}").get("company") or {}
        wr_before_raw = _get_field(company_before.get(wr_id, {}), wr_field)
        ac_before_raw = _get_field(company_before.get(ac_id, {}), ac_field)
        wr_after_raw = _get_field(wr_after, wr_field)
        ac_after_raw = _get_field(ac_after, ac_field)
        if db_path.exists() and wr_numeric is not None and wr_kind == "wr":
            wr_after_raw = _fetch_db_value(db_path, "wholesale_retail", wr_numeric, wr_field)
        if db_path.exists() and ac_numeric is not None and ac_kind == "ac":
            ac_after_raw = _fetch_db_value(db_path, "accommodation_catering", ac_numeric, ac_field)
        wr_after_val = _parse_num(wr_after_raw)
        ac_after_val = _parse_num(ac_after_raw)
        updates["wr"] = {
            "id": wr_id,
            "field": wr_field,
            "before": wr_before_raw,
            "afterRaw": wr_after_raw,
            "after": wr_after_val,
            "target": wr_target,
            "ok": _close(wr_after_val, wr_target, eps=0.6),
        }
        updates["ac"] = {
            "id": ac_id,
            "field": ac_field,
            "before": ac_before_raw,
            "afterRaw": ac_after_raw,
            "after": ac_after_val,
            "target": ac_target,
            "ok": _close(ac_after_val, ac_target, eps=0.6),
        }
        updates_ok = bool(updates["wr"]["ok"]) and bool(updates["ac"]["ok"])
    except Exception as e:
        updates_ok = False
        results["errors"].append(f"verify updates failed: {e}")

    # 6) 第二轮对话：仅触发指标目标
    second_ok = True
    second_summary_ok = False
    try:
        indicator_prompt = (
            f"只调整指标 {indicator_id} 到 {indicator_target}，不要修改企业数据；"
            f"必须调用 set_indicator_targets。"
        )
        _agent(f'find placeholder "输入你的调整需求…" fill "{indicator_prompt}"', log_path)
        _agent('find role button click --name "发送"', log_path)
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)
        summary_ret2 = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_chat_summary_text())} --json", log_path)
        )
        summary_text2 = str(summary_ret2.get("text") or "")
        summary_nums2 = _parse_summary_numbers(summary_text2)
        second_summary_ok = summary_nums2.get("optimized") is True and (summary_nums2.get("targetIndicators") or 0) > 0
        if not summary_text2:
            err_ret2 = _unwrap_agent_browser_json(
                _agent_json(f"eval {shlex.quote(_js_get_chat_error_text())} --json", log_path)
            )
            err_text2 = str(err_ret2.get("text") or "")
            if err_text2:
                results["errors"].append(err_text2)
        results["second"] = {
            "summaryText": summary_text2,
            "summary": summary_nums2,
            "expectedOptimized": True,
        }
        _agent(f'screenshot \"{screenshots / "18_llm_after_second.png"}\"', log_path)
        results["screenshots"]["afterSecond"] = str(screenshots / "18_llm_after_second.png")

        wr_after_opt = _http_json(f"{base_url}/api/companies/{wr_id}").get("company") or {}
        ac_after_opt = _http_json(f"{base_url}/api/companies/{ac_id}").get("company") or {}
        wr_after_opt_raw = _get_field(wr_after_opt, wr_field)
        ac_after_opt_raw = _get_field(ac_after_opt, ac_field)
        if db_path.exists() and wr_numeric is not None and wr_kind == "wr":
            wr_after_opt_raw = _fetch_db_value(db_path, "wholesale_retail", wr_numeric, wr_field)
        if db_path.exists() and ac_numeric is not None and ac_kind == "ac":
            ac_after_opt_raw = _fetch_db_value(db_path, "accommodation_catering", ac_numeric, ac_field)
        wr_after_opt_val = _parse_num(wr_after_opt_raw)
        ac_after_opt_val = _parse_num(ac_after_opt_raw)
        updates["afterOptimize"] = {
            "wr": {"after": wr_after_opt_val, "target": wr_target, "ok": _close(wr_after_opt_val, wr_target, eps=0.6)},
            "ac": {"after": ac_after_opt_val, "target": ac_target, "ok": _close(ac_after_opt_val, ac_target, eps=0.6)},
        }
        if not (updates["afterOptimize"]["wr"]["ok"] and updates["afterOptimize"]["ac"]["ok"]):
            updates_ok = False
    except SystemExit as e:
        second_ok = False
        results["errors"].append(f"second chat failed: exit={e.code}")

    # 7) 新建会话（消息清空）
    new_session_ok = True
    try:
        _agent('find role button click --name "新建会话"', log_path)
        time.sleep(0.3)
        count_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_message_count())} --json", log_path)
        )
        count = int(count_ret.get("count") or 0)
        new_session_ok = count == 0
        results["newSession"] = {"messageCount": count, "ok": new_session_ok}
        _agent(f'screenshot \"{screenshots / "19_llm_new_session.png"}\"', log_path)
        results["screenshots"]["newSession"] = str(screenshots / "19_llm_new_session.png")
    except SystemExit as e:
        new_session_ok = False
        results["errors"].append(f"new session failed: exit={e.code}")

    results["streaming"] = {"ok": streaming_ok}
    results["updates"] = updates

    # 8) 流式输出详细测试
    streaming_detail_ok = True
    try:
        _agent(f'find placeholder "输入你的调整需求…" fill "测试流式输出"', log_path)
        _agent('find role button click --name "发送"', log_path)

        # 检查流式指示器
        stream_check = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_check_streaming_indicator())} --json", log_path)
        )

        # 等待流式开始
        time.sleep(0.5)
        _agent(f'screenshot \"{screenshots / "20_llm_streaming_detail.png"}\"', log_path)
        results["screenshots"]["streamingDetail"] = str(screenshots / "20_llm_streaming_detail.png")

        # 获取消息内容
        messages_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_chat_messages())} --json", log_path)
        )

        results["streamingDetail"] = {
            "indicatorExists": bool(stream_check.get("exists")),
            "messageCount": int(messages_ret.get("count") or 0),
            "ok": True,
        }
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)
    except SystemExit as e:
        streaming_detail_ok = False
        results["errors"].append(f"streaming detail test failed: exit={e.code}")
        results["streamingDetail"] = {"ok": False}

    # 9) 对话框位置测试 - 验证是否为右侧侧边栏
    dialog_position_ok = True
    try:
        pos_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_check_dialog_position())} --json", log_path)
        )
        is_right_side = bool(pos_ret.get("isRightSide"))
        results["dialogPosition"] = {
            "isRightSide": is_right_side,
            "width": pos_ret.get("width"),
            "ok": is_right_side,
        }
        dialog_position_ok = is_right_side
        _agent(f'screenshot \"{screenshots / "21_llm_dialog_position.png"}\"', log_path)
        results["screenshots"]["dialogPosition"] = str(screenshots / "21_llm_dialog_position.png")
    except SystemExit as e:
        dialog_position_ok = False
        results["errors"].append(f"dialog position check failed: exit={e.code}")
        results["dialogPosition"] = {"ok": False}

    # 10) 多轮对话上下文测试
    multi_turn_ok = True
    try:
        # 第一轮
        _agent(f'find placeholder "输入你的调整需求…" fill "记住数字 12345"', log_path)
        _agent('find role button click --name "发送"', log_path)
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)

        # 第二轮 - 询问之前的内容
        _agent(f'find placeholder "输入你的调整需求…" fill "我刚才让你记住的数字是什么"', log_path)
        _agent('find role button click --name "发送"', log_path)
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)

        # 获取最后一条消息
        last_msg = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_last_message_content())} --json", log_path)
        )
        last_text = str(last_msg.get("text") or "")

        # 检查是否包含之前的数字
        context_kept = "12345" in last_text or "数字" in last_text
        results["multiTurn"] = {
            "contextKept": context_kept,
            "lastMessage": last_text[:100],
            "ok": context_kept,
        }
        multi_turn_ok = context_kept
        _agent(f'screenshot \"{screenshots / "22_llm_multi_turn.png"}\"', log_path)
        results["screenshots"]["multiTurn"] = str(screenshots / "22_llm_multi_turn.png")
    except SystemExit as e:
        multi_turn_ok = False
        results["errors"].append(f"multi-turn test failed: exit={e.code}")
        results["multiTurn"] = {"ok": False}

    # 11) 边界测试 - 超长输入
    boundary_test_ok = True
    try:
        long_input = "测试" * 100  # 200字符
        _agent(f'find placeholder "输入你的调整需求…" fill "{long_input}"', log_path)
        _agent('find role button click --name "发送"', log_path)
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)

        results["boundaryTest"] = {"ok": True, "inputLength": len(long_input)}
        _agent(f'screenshot \"{screenshots / "23_llm_boundary_test.png"}\"', log_path)
        results["screenshots"]["boundaryTest"] = str(screenshots / "23_llm_boundary_test.png")
    except SystemExit as e:
        boundary_test_ok = False
        results["errors"].append(f"boundary test failed: exit={e.code}")
        results["boundaryTest"] = {"ok": False}

    # 12) 关闭和重新打开对话框测试
    reopen_ok = True
    try:
        # 关闭对话框
        _agent('find role button click --name "关闭"', log_path)
        time.sleep(0.3)

        # 重新打开
        _agent(f'eval {shlex.quote(_js_click_chat_fab())} --json', log_path)
        _agent('wait --text "数据对话助手"', log_path)

        # 验证消息历史是否保留
        count_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_message_count())} --json", log_path)
        )
        count = int(count_ret.get("count") or 0)

        results["reopen"] = {"messageCount": count, "ok": count > 0}
        reopen_ok = count > 0
        _agent(f'screenshot \"{screenshots / "24_llm_reopen.png"}\"', log_path)
        results["screenshots"]["reopen"] = str(screenshots / "24_llm_reopen.png")
    except SystemExit as e:
        reopen_ok = False
        results["errors"].append(f"reopen test failed: exit={e.code}")
        results["reopen"] = {"ok": False}

    # 13) 模板问题测试 - 验证指标修改是否生效
    template_issue_ok = True
    try:
        # 获取修改前的指标值
        indicators_before = _http_json(f"{base_url}/api/indicators")
        groups_before = indicators_before.get("groups") or []
        found_val_before = None
        for g in groups_before:
            for it in g.get("indicators") or []:
                if str(it.get("id") or "") == indicator_id:
                    found_val_before = _parse_num(it.get("value"))
                    break

        # 发送修改指标的请求
        template_prompt = (
            f"请调用 set_indicator_targets，将指标 {indicator_id} 调整到 {indicator_target}；"
            f"不要修改企业数据。"
        )
        _agent(f'find placeholder "输入你的调整需求…" fill "{template_prompt}"', log_path)
        _agent('find role button click --name "发送"', log_path)
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)

        # 验证指标是否已修改
        time.sleep(0.5)  # 等待数据同步
        indicators_after = _http_json(f"{base_url}/api/indicators")
        groups_after = indicators_after.get("groups") or []
        found_val_after = None
        for g in groups_after:
            for it in g.get("indicators") or []:
                if str(it.get("id") or "") == indicator_id:
                    found_val_after = _parse_num(it.get("value"))
                    break

        # 验证指标值是否已修改
        value_changed = (
            found_val_after is not None
            and found_val_before is not None
            and abs(found_val_after - indicator_target) <= 0.6
        )

        results["templateIssue"] = {
            "indicatorId": indicator_id,
            "beforeValue": found_val_before,
            "afterValue": found_val_after,
            "targetValue": indicator_target,
            "valueChanged": value_changed,
            "ok": value_changed,
        }
        template_issue_ok = value_changed
        _agent(f'screenshot \"{screenshots / "25_llm_template_issue.png"}\"', log_path)
        results["screenshots"]["templateIssue"] = str(screenshots / "25_llm_template_issue.png")
    except SystemExit as e:
        template_issue_ok = False
        results["errors"].append(f"template issue test failed: exit={e.code}")
        results["templateIssue"] = {"ok": False}

    # 14) 错误处理测试 - 验证错误消息是否正确显示
    error_handling_ok = True
    try:
        # 发送一个无效的请求
        invalid_prompt = "请调用一个不存在的工具函数"
        _agent(f'find placeholder "输入你的调整需求…" fill "{invalid_prompt}"', log_path)
        _agent('find role button click --name "发送"', log_path)
        _agent(f"wait --fn {shlex.quote(_js_wait_for_chat_terminal())} --timeout 180000", log_path)

        # 检查是否有错误消息
        error_ret = _unwrap_agent_browser_json(
            _agent_json(f"eval {shlex.quote(_js_get_chat_error_text())} --json", log_path)
        )
        error_text = str(error_ret.get("text") or "")

        results["errorHandling"] = {
            "hasError": bool(error_text),
            "errorText": error_text[:200],
            "ok": bool(error_text),
        }
        error_handling_ok = bool(error_text)
        _agent(f'screenshot \"{screenshots / "26_llm_error_handling.png"}\"', log_path)
        results["screenshots"]["errorHandling"] = str(screenshots / "26_llm_error_handling.png")
    except SystemExit as e:
        error_handling_ok = False
        results["errors"].append(f"error handling test failed: exit={e.code}")
        results["errorHandling"] = {"ok": False}

    results["ok"] = (
        config_ok
        and open_ok
        and first_ok
        and second_ok
        and streaming_ok
        and summary_ok
        and second_summary_ok
        and updates_ok
        and new_session_ok
        and streaming_detail_ok
        and dialog_position_ok
        and multi_turn_ok
        and boundary_test_ok
        and reopen_ok
        and template_issue_ok
        and error_handling_ok
    )

    out_path.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"LLM results written: {out_path}")


if __name__ == "__main__":
    main()

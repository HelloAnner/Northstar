#!/usr/bin/env python3
# 联动预览高亮 E2E 测试脚本
# @author Anner
# Created on 2026/2/3
# Updated on 2026/2/4

import json
import shlex
import subprocess
import sys
import time
from pathlib import Path
from typing import Any, Dict, List


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


def _js_pick_derived_cell() -> str:
    return (
        "(() => {\n"
        "  const table = document.querySelector('table');\n"
        "  if (!table) return { ok: false, error: 'table not found' };\n"
        "  const headers = Array.from(table.querySelectorAll('thead th')).map(th => (th.textContent || '').trim());\n"
        "  const candidates = [\n"
        "    { label: '同比增速(当月)', parents: ['本年-本月', '上年-本月'] },\n"
        "    { label: '同比增量(当月)', parents: ['本年-本月', '上年-本月'] },\n"
        "    { label: '零售额;同比增速(当月)', parents: ['零售额;本年-本月', '零售额;上年-本月'] },\n"
        "    { label: '零售额;同比增量(当月)', parents: ['零售额;本年-本月', '零售额;上年-本月'] },\n"
        "  ];\n"
        "  let chosen = null;\n"
        "  for (const cand of candidates) {\n"
        "    const idx = headers.indexOf(cand.label);\n"
        "    if (idx < 0) continue;\n"
        "    const parentIdx = cand.parents.map(p => headers.indexOf(p));\n"
        "    if (parentIdx.some(i => i < 0)) continue;\n"
        "    chosen = { label: cand.label, parents: cand.parents, idx, parentIdx };\n"
        "    break;\n"
        "  }\n"
        "  if (!chosen) return { ok: false, error: 'derived header not found', headers };\n"
        "  const rows = Array.from(table.querySelectorAll('tbody tr'));\n"
        "  for (let r = 0; r < rows.length; r++) {\n"
        "    const tds = Array.from(rows[r].querySelectorAll('td'));\n"
        "    if (tds.length <= chosen.idx) continue;\n"
        "    const cell = tds[chosen.idx];\n"
        "    if (!cell) continue;\n"
        "    const name = (tds[0].querySelector('.truncate.font-medium')?.textContent || '').trim();\n"
        "    const credit = (tds[0].querySelector('.font-mono')?.textContent || '').trim();\n"
        "    try { cell.scrollIntoView({ block: 'center', inline: 'center' }); } catch (_) {}\n"
        "    cell.dispatchEvent(new MouseEvent('click', { bubbles: true }));\n"
        "    window.__ns_linkage_preview = { rowIndex: r, colIndex: chosen.idx, header: chosen.label, name, credit, parents: chosen.parents, parentIdx: chosen.parentIdx };\n"
        "    return { ok: true, rowIndex: r, colIndex: chosen.idx, header: chosen.label, name, credit, parents: chosen.parents, parentIdx: chosen.parentIdx };\n"
        "  }\n"
        "  return { ok: false, error: 'no data row found', headers, chosen };\n"
        "})()"
    )


def _js_check_derived_highlight() -> str:
    return (
        "(() => {\n"
        "  const target = window.__ns_linkage_preview;\n"
        "  if (!target) return { ok: false, error: 'missing target' };\n"
        "  const rows = Array.from(document.querySelectorAll('tbody tr'));\n"
        "  const row = rows[target.rowIndex];\n"
        "  const cells = row ? Array.from(row.querySelectorAll('td')) : [];\n"
        "  const cell = cells[target.colIndex];\n"
        "  const highlighted = !!(cell && cell.classList.contains('ring-yellow-400/80'));\n"
        "  const parentStatus = [];\n"
        "  const parentIdx = Array.isArray(target.parentIdx) ? target.parentIdx : [];\n"
        "  const parentLabels = Array.isArray(target.parents) ? target.parents : [];\n"
        "  for (let i = 0; i < parentIdx.length; i++) {\n"
        "    const idx = parentIdx[i];\n"
        "    const label = parentLabels[i] || String(idx);\n"
        "    const c = cells[idx];\n"
        "    const ok = !!(c && c.classList.contains('ring-yellow-400/80'));\n"
        "    parentStatus.push({ label, idx, highlighted: ok });\n"
        "  }\n"
        "  const all = Array.from(document.querySelectorAll('tbody td'));\n"
        "  const highlightCount = all.filter(td => td.classList.contains('ring-yellow-400/80')).length;\n"
        "  const parentsOk = parentStatus.length > 0 && parentStatus.every(p => p.highlighted);\n"
        "  return { ok: highlighted && parentsOk && highlightCount > 0, highlighted, parentStatus, highlightCount, target };\n"
        "})()"
    )


def _js_click_indicator() -> str:
    return (
        "(() => {\n"
        "  const inputs = Array.from(document.querySelectorAll('input.rounded-full'));\n"
        "  if (!inputs.length) return { ok: false, error: 'indicator inputs not found' };\n"
        "  const pick = (label) => {\n"
        "    for (const input of inputs) {\n"
        "      const row = input.closest('div.flex.items-center.gap-3') || input.closest('div.flex.items-center') || input.parentElement?.parentElement;\n"
        "      if (!row) continue;\n"
        "      const labelEl = row.querySelector('div.min-w-0') || row.querySelector('div');\n"
        "      const text = labelEl ? String(labelEl.textContent || '').trim() : '';\n"
        "      if (text.includes(label)) return { input, text };\n"
        "    }\n"
        "    return null;\n"
        "  };\n"
        "  let picked = pick('批发') || pick('零售') || pick('住宿') || pick('餐饮');\n"
        "  if (!picked) picked = { input: inputs[0], text: '' };\n"
        "  const input = picked.input;\n"
        "  try { input.scrollIntoView({ block: 'center', inline: 'center' }); } catch (_) {}\n"
        "  input.click();\n"
        "  window.__ns_linkage_indicator = { label: picked.text || '' };\n"
        "  return { ok: true, label: picked.text || '' };\n"
        "})()"
    )


def _js_check_indicator_highlight() -> str:
    return (
        "(() => {\n"
        "  const target = window.__ns_linkage_indicator || {};\n"
        "  const inputs = Array.from(document.querySelectorAll('input.rounded-full'));\n"
        "  const highlightedInputs = inputs.filter(input => input.classList.contains('border-yellow-400') || input.classList.contains('ring-yellow-400/50'));\n"
        "  const cells = Array.from(document.querySelectorAll('tbody td'));\n"
        "  const highlightedCells = cells.filter(td => td.classList.contains('ring-yellow-400/80'));\n"
        "  const ok = highlightedInputs.length > 0 && highlightedCells.length > 0;\n"
        "  return { ok, highlightIndicators: highlightedInputs.length, highlightCells: highlightedCells.length, target };\n"
        "})()"
    )


def _js_clear_highlight() -> str:
    return (
        "(() => {\n"
        "  document.body.click();\n"
        "  return { ok: true };\n"
        "})()"
    )


def main() -> None:
    if len(sys.argv) != 4:
        print(
            "Usage: run_agent_browser_e2e_linkage_preview.py <logPath> <screenshotsDir> <outJson>",
            file=sys.stderr,
        )
        raise SystemExit(2)

    log_path = Path(sys.argv[1])
    screenshots = Path(sys.argv[2])
    out_json = Path(sys.argv[3])

    result: Dict[str, Any] = {"ok": False}

    try:
        _agent("wait --load networkidle", log_path)

        pick_ret = _agent_json(f"eval {shlex.quote(_js_pick_derived_cell())} --json", log_path)
        pick_res = _unwrap_agent_browser_json(pick_ret)
        if not pick_res.get("ok"):
            result = {"ok": False, "error": pick_res.get("error") or "pick derived cell failed", "pick": pick_ret}
            out_json.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
            raise SystemExit(1)

        time.sleep(0.6)
        try:
            _agent("wait --fn \"(() => { return Array.from(document.querySelectorAll('tbody td')).some(td => td.classList.contains('ring-yellow-400/80')); })()\"", log_path)
        except SystemExit:
            pass

        check_ret = _agent_json(f"eval {shlex.quote(_js_check_derived_highlight())} --json", log_path)
        check_res = _unwrap_agent_browser_json(check_ret)
        derived_ok = bool(check_res.get("ok"))

        _agent_json(f"eval {shlex.quote(_js_clear_highlight())} --json", log_path)
        time.sleep(0.2)

        indicator_pick = _agent_json(f"eval {shlex.quote(_js_click_indicator())} --json", log_path)
        indicator_pick_res = _unwrap_agent_browser_json(indicator_pick)
        time.sleep(0.4)
        try:
            _agent(
                "wait --fn \"(() => { return document.querySelectorAll('input.rounded-full.border-yellow-400').length > 0 || document.querySelectorAll('tbody td.ring-yellow-400/80').length > 0; })()\"",
                log_path,
            )
        except SystemExit:
            pass
        indicator_check = _agent_json(f"eval {shlex.quote(_js_check_indicator_highlight())} --json", log_path)
        indicator_check_res = _unwrap_agent_browser_json(indicator_check)

        indicator_ok = bool(indicator_check_res.get("ok"))
        result = {
            "ok": bool(derived_ok and indicator_ok),
            "derived": {"pick": pick_res, "check": check_res},
            "indicator": {"pick": indicator_pick_res, "check": indicator_check_res},
        }
        out_json.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")

        if not result["ok"]:
            raise SystemExit(1)

        _agent(f"screenshot \"{screenshots / '08_linkage_preview.png'}\"", log_path)
    except SystemExit:
        _agent(f"screenshot \"{screenshots / '08_linkage_preview_failed.png'}\"", log_path)
        raise


if __name__ == "__main__":
    main()

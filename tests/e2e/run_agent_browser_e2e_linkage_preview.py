#!/usr/bin/env python3
# 联动预览高亮 E2E 测试脚本
# @author Anner
# Created on 2026/2/3

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


def _js_pick_cell() -> str:
    return (
        "(() => {\n"
        "  const table = document.querySelector('table');\n"
        "  if (!table) return { ok: false, error: 'table not found' };\n"
        "  const headers = Array.from(table.querySelectorAll('thead th')).map(th => (th.textContent || '').trim());\n"
        "  const rows = Array.from(table.querySelectorAll('tbody tr'));\n"
        "  for (let r = 0; r < rows.length; r++) {\n"
        "    const tds = Array.from(rows[r].querySelectorAll('td'));\n"
        "    if (tds.length < 2) continue;\n"
        "    for (let c = 1; c < tds.length; c++) {\n"
        "      const cell = tds[c];\n"
        "      if (!cell) continue;\n"
        "      const name = (tds[0].querySelector('.truncate.font-medium')?.textContent || '').trim();\n"
        "      const credit = (tds[0].querySelector('.font-mono')?.textContent || '').trim();\n"
        "      try { cell.scrollIntoView({ block: 'center', inline: 'center' }); } catch (_) {}\n"
        "      cell.dispatchEvent(new MouseEvent('click', { bubbles: true }));\n"
        "      window.__ns_linkage_preview = { rowIndex: r, colIndex: c, header: headers[c] || '', name, credit };\n"
        "      return { ok: true, rowIndex: r, colIndex: c, header: headers[c] || '', name, credit };\n"
        "    }\n"
        "  }\n"
        "  return { ok: false, error: 'no data cell found' };\n"
        "})()"
    )


def _js_check_highlight() -> str:
    return (
        "(() => {\n"
        "  const target = window.__ns_linkage_preview;\n"
        "  if (!target) return { ok: false, error: 'missing target' };\n"
        "  const rows = Array.from(document.querySelectorAll('tbody tr'));\n"
        "  const row = rows[target.rowIndex];\n"
        "  const cells = row ? Array.from(row.querySelectorAll('td')) : [];\n"
        "  const cell = cells[target.colIndex];\n"
        "  const highlighted = !!(cell && cell.classList.contains('ring-yellow-400/80'));\n"
        "  const all = Array.from(document.querySelectorAll('tbody td'));\n"
        "  const highlightCount = all.filter(td => td.classList.contains('ring-yellow-400/80')).length;\n"
        "  return { ok: highlighted && highlightCount > 0, highlighted, highlightCount, target };\n"
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
        pick_ret = _agent_json(f"eval {shlex.quote(_js_pick_cell())} --json", log_path)
        pick_res = _unwrap_agent_browser_json(pick_ret)
        if not pick_res.get("ok"):
            result = {"ok": False, "error": pick_res.get("error") or "pick cell failed", "pick": pick_ret}
            out_json.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
            raise SystemExit(1)

        time.sleep(0.6)
        try:
            _agent("wait --fn \"(() => { return Array.from(document.querySelectorAll('tbody td')).some(td => td.classList.contains('ring-yellow-400/80')); })()\"", log_path)
        except SystemExit:
            pass

        check_ret = _agent_json(f"eval {shlex.quote(_js_check_highlight())} --json", log_path)
        check_res = _unwrap_agent_browser_json(check_ret)
        result = {"ok": bool(check_res.get("ok")), "pick": pick_res, "check": check_res}
        out_json.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")

        if not result["ok"]:
            raise SystemExit(1)

        _agent(f"screenshot \"{screenshots / '08_linkage_preview.png'}\"", log_path)
        _agent_json(f"eval {shlex.quote(_js_clear_highlight())} --json", log_path)
        time.sleep(0.2)
    except SystemExit:
        _agent(f"screenshot \"{screenshots / '08_linkage_preview_failed.png'}\"", log_path)
        raise


if __name__ == "__main__":
    main()

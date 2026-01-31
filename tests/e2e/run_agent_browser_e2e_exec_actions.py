#!/usr/bin/env python3
import json
import shlex
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional


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


def _js_set_value(credit_code: str, column_label: str, new_value: float) -> str:
    # Finds a row by credit code and sets the editable input for the given column label.
    # Relies on headers being visible (we pre-enable all columns).
    return (
        "(() => {\n"
        f"  const cc = {json.dumps(credit_code)};\n"
        f"  const label = {json.dumps(column_label)};\n"
        f"  const value = {json.dumps(str(new_value))};\n"
        "  const table = document.querySelector('table');\n"
        "  if (!table) return { ok: false, error: 'table not found' };\n"
        "  const headers = Array.from(table.querySelectorAll('thead th')).map(th => (th.textContent||'').trim());\n"
        "  const idx = headers.findIndex(h => h === label);\n"
        "  if (idx < 0) return { ok: false, error: 'header not found', headers };\n"
        "  const rows = Array.from(table.querySelectorAll('tbody tr'));\n"
        "  for (const tr of rows) {\n"
        "    const ccEl = tr.querySelector('td:nth-child(1) .font-mono');\n"
        "    const ccText = (ccEl?.textContent || '').trim();\n"
        "    if (ccText !== cc) continue;\n"
        "    const cell = tr.querySelector(`td:nth-child(${idx+1})`);\n"
        "    if (!cell) return { ok: false, error: 'cell not found', idx };\n"
        "    const input = cell.querySelector('input');\n"
        "    if (!input) return { ok: false, error: 'input not found (not editable?)', idx };\n"
        "    input.focus();\n"
        "    input.value = value;\n"
        "    input.dispatchEvent(new Event('input', { bubbles: true }));\n"
        "    input.dispatchEvent(new Event('change', { bubbles: true }));\n"
        "    input.blur();\n"
        "    return { ok: true, idx, cc, label, value };\n"
        "  }\n"
        "  return { ok: false, error: 'row not found', cc };\n"
        "})()"
    )


def _js_get_value(credit_code: str, column_label: str) -> str:
    return (
        "(() => {\n"
        f"  const cc = {json.dumps(credit_code)};\n"
        f"  const label = {json.dumps(column_label)};\n"
        "  const table = document.querySelector('table');\n"
        "  if (!table) return { ok: false, error: 'table not found' };\n"
        "  const headers = Array.from(table.querySelectorAll('thead th')).map(th => (th.textContent||'').trim());\n"
        "  const idx = headers.findIndex(h => h === label);\n"
        "  if (idx < 0) return { ok: false, error: 'header not found', headers };\n"
        "  const rows = Array.from(table.querySelectorAll('tbody tr'));\n"
        "  for (const tr of rows) {\n"
        "    const ccEl = tr.querySelector('td:nth-child(1) .font-mono');\n"
        "    const ccText = (ccEl?.textContent || '').trim();\n"
        "    if (ccText !== cc) continue;\n"
        "    const cell = tr.querySelector(`td:nth-child(${idx+1})`);\n"
        "    if (!cell) return { ok: false, error: 'cell not found', idx };\n"
        "    const input = cell.querySelector('input');\n"
        "    if (input) return { ok: true, value: String(input.value || '') };\n"
        "    return { ok: true, value: String((cell.textContent||'').trim()) };\n"
        "  }\n"
        "  return { ok: false, error: 'row not found', cc };\n"
        "})()"
    )


def main() -> None:
    if len(sys.argv) != 4:
        print("Usage: run_agent_browser_e2e_exec_actions.py <actions.json> <logPath> <screenshotsDir>", file=sys.stderr)
        raise SystemExit(2)

    actions_path = Path(sys.argv[1])
    log_path = Path(sys.argv[2])
    screenshots = Path(sys.argv[3])

    if not actions_path.exists():
        print(f"No actions file: {actions_path}")
        return

    payload = json.loads(actions_path.read_text(encoding="utf-8"))
    actions = payload.get("actions") or []
    if not actions:
        print("No actions to execute.")
        (screenshots.parent / "actions_result.json").write_text(
            json.dumps({"results": []}, ensure_ascii=False, indent=2), encoding="utf-8"
        )
        return

    # Ensure table is in a stable state: select "全部" tab and clear search.
    _agent('find role tab click --name "全部"', log_path)
    _agent('find placeholder "按企业名称/信用代码搜索…" fill ""', log_path)
    _agent("wait --load networkidle", log_path)
    _agent(f'screenshot "{screenshots / "04_before_actions.png"}"', log_path)

    results: List[Dict[str, Any]] = []
    for i, a in enumerate(actions, start=1):
        cc = str(a["credit_code"])
        label = str(a["column_label"])
        new_value = float(a["new_value"])
        reason = str(a.get("reason") or "")

        try:
            _agent(f'find placeholder "按企业名称/信用代码搜索…" fill "{cc}"', log_path)
            _agent("wait --load networkidle", log_path)
            _agent(f'wait --text "{cc}"', log_path)
            set_ret = _agent_json(f'eval {shlex.quote(_js_set_value(cc, label, new_value))} --json', log_path)
            _agent("wait --load networkidle", log_path)
            get_ret = _agent_json(f'eval {shlex.quote(_js_get_value(cc, label))} --json', log_path)
            set_res = _unwrap_agent_browser_json(set_ret)
            get_res = _unwrap_agent_browser_json(get_ret)
            if not bool(set_res.get("ok")):
                _agent(f'screenshot "{screenshots / f"05_action_{i}.png"}"', log_path)
                results.append(
                    {
                        "i": i,
                        "creditCode": cc,
                        "field": label,
                        "value": new_value,
                        "ok": False,
                        "reason": reason,
                        "error": str(set_res.get("error") or "set failed"),
                        "setResult": set_ret,
                        "uiValue": get_res.get("value"),
                    }
                )
                continue
            _agent(f'screenshot "{screenshots / f"05_action_{i}.png"}"', log_path)
            results.append(
                {
                    "i": i,
                    "creditCode": cc,
                    "field": label,
                    "value": new_value,
                    "ok": True,
                    "reason": reason,
                    "setResult": set_ret,
                    "uiValue": get_res.get("value"),
                }
            )
        except SystemExit as e:
            results.append({"i": i, "creditCode": cc, "field": label, "value": new_value, "ok": False, "reason": reason, "error": f"agent-browser failed: exit={e.code}"})
        finally:
            try:
                _agent('find placeholder "按企业名称/信用代码搜索…" fill ""', log_path)
                _agent("wait --load networkidle", log_path)
            except SystemExit:
                pass

        # Add a lightweight marker into the log for the report.
        log_path.write_text(
            log_path.read_text(encoding="utf-8")
            + f"\n[ACTION] {i}/{len(actions)} {cc} {label}={new_value} ({reason})\n",
            encoding="utf-8",
        )

    # Reload once and verify persisted UI values for each action.
    persist: List[Dict[str, Any]] = []
    try:
        _agent("reload", log_path)
        _agent("wait --load networkidle", log_path)
        _agent('find role tab click --name "全部"', log_path)
        _agent('find placeholder "按企业名称/信用代码搜索…" fill ""', log_path)
        _agent("wait --load networkidle", log_path)
        _agent(f'screenshot "{screenshots / "09_after_reload.png"}"', log_path)
        for i, a in enumerate(actions, start=1):
            cc = str(a["credit_code"])
            label = str(a["column_label"])
            try:
                _agent(f'find placeholder "按企业名称/信用代码搜索…" fill "{cc}"', log_path)
                _agent("wait --load networkidle", log_path)
                _agent(f'wait --text "{cc}"', log_path)
                get_ret = _agent_json(f'eval {shlex.quote(_js_get_value(cc, label))} --json', log_path)
                get_res = _unwrap_agent_browser_json(get_ret)
                _agent(f'screenshot "{screenshots / f"09_persist_{i}.png"}"', log_path)
                persist.append(
                    {
                        "i": i,
                        "creditCode": cc,
                        "field": label,
                        "ok": True,
                        "uiValue": get_res.get("value"),
                    }
                )
            except SystemExit as e:
                persist.append({"i": i, "creditCode": cc, "field": label, "ok": False, "error": f"agent-browser failed: exit={e.code}"})
            finally:
                try:
                    _agent('find placeholder "按企业名称/信用代码搜索…" fill ""', log_path)
                    _agent("wait --load networkidle", log_path)
                except SystemExit:
                    pass
    except SystemExit:
        persist.append({"i": 0, "ok": False, "error": "failed to reload/verify persistence"})

    (screenshots.parent / "actions_result.json").write_text(
        json.dumps({"results": results, "persist": persist}, ensure_ascii=False, indent=2), encoding="utf-8"
    )


if __name__ == "__main__":
    main()

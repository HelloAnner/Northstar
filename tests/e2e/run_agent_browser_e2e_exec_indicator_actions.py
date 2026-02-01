#!/usr/bin/env python3
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


def _js_set_indicator_value(label: str, new_value: float) -> str:
    return (
        "(() => {\n"
        f"  const label = {json.dumps(label)};\n"
        f"  const value = {json.dumps(str(new_value))};\n"
        "  const norm = (s) => String(s || '').replace(/\\s+/g, '').trim();\n"
        "  const inputs = Array.from(document.querySelectorAll('input.rounded-full'));\n"
        "  for (const input of inputs) {\n"
        "    const row = input.closest('div.flex.items-center') || input.parentElement?.parentElement;\n"
        "    if (!row) continue;\n"
        "    const labelEl = row.querySelector('div.min-w-0') || row.querySelector('div');\n"
        "    const text = labelEl ? String(labelEl.textContent || '').trim() : '';\n"
        "    if (norm(text) !== norm(label)) continue;\n"
        "    try { input.scrollIntoView({ block: 'center', inline: 'center' }); } catch (_) {}\n"
        "    input.focus();\n"
        "    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set;\n"
        "    if (setter) setter.call(input, value); else input.value = value;\n"
        "    input.dispatchEvent(new Event('input', { bubbles: true }));\n"
        "    input.dispatchEvent(new Event('change', { bubbles: true }));\n"
        "    input.blur();\n"
        "    return { ok: true, label: text, value };\n"
        "  }\n"
        "  return { ok: false, error: 'indicator not found', label, count: inputs.length };\n"
        "})()"
    )


def _js_get_indicator_value(label: str) -> str:
    return (
        "(() => {\n"
        f"  const label = {json.dumps(label)};\n"
        "  const norm = (s) => String(s || '').replace(/\\s+/g, '').trim();\n"
        "  const inputs = Array.from(document.querySelectorAll('input.rounded-full'));\n"
        "  for (const input of inputs) {\n"
        "    const row = input.closest('div.flex.items-center') || input.parentElement?.parentElement;\n"
        "    if (!row) continue;\n"
        "    const labelEl = row.querySelector('div.min-w-0') || row.querySelector('div');\n"
        "    const text = labelEl ? String(labelEl.textContent || '').trim() : '';\n"
        "    if (norm(text) !== norm(label)) continue;\n"
        "    return { ok: true, label: text, value: String(input.value || '') };\n"
        "  }\n"
        "  return { ok: false, error: 'indicator not found', label, count: inputs.length };\n"
        "})()"
    )


def main() -> None:
    if len(sys.argv) != 4:
        print(
            "Usage: run_agent_browser_e2e_exec_indicator_actions.py <indicator_actions.json> <logPath> <screenshotsDir>",
            file=sys.stderr,
        )
        raise SystemExit(2)

    actions_path = Path(sys.argv[1])
    log_path = Path(sys.argv[2])
    screenshots = Path(sys.argv[3])

    if not actions_path.exists():
        print(f"No indicator actions file: {actions_path}")
        return

    payload = json.loads(actions_path.read_text(encoding="utf-8"))
    actions = payload.get("actions") or []
    if not actions:
        print("No indicator actions to execute.")
        (screenshots.parent / "indicator_actions_result.json").write_text(
            json.dumps({"results": []}, ensure_ascii=False, indent=2), encoding="utf-8"
        )
        return

    results: List[Dict[str, Any]] = []
    for i, a in enumerate(actions, start=1):
        label = str(a.get("label") or "")
        target = float(a.get("target") or 0.0)
        reason = str(a.get("reason") or "")

        try:
            set_ret = _agent_json(f'eval {shlex.quote(_js_set_indicator_value(label, target))} --json', log_path)
            time.sleep(0.2)
            get_ret = _agent_json(f'eval {shlex.quote(_js_get_indicator_value(label))} --json', log_path)
            set_res = _unwrap_agent_browser_json(set_ret)
            get_res = _unwrap_agent_browser_json(get_ret)
            if not bool(set_res.get("ok")):
                results.append(
                    {
                        "i": i,
                        "label": label,
                        "value": target,
                        "ok": False,
                        "reason": reason,
                        "error": str(set_res.get("error") or "set failed"),
                        "setResult": set_ret,
                        "uiValue": get_res.get("value"),
                    }
                )
                continue
            results.append(
                {
                    "i": i,
                    "label": label,
                    "value": target,
                    "ok": True,
                    "reason": reason,
                    "setResult": set_ret,
                    "uiValue": get_res.get("value"),
                }
            )
        except SystemExit as e:
            results.append(
                {
                    "i": i,
                    "label": label,
                    "value": target,
                    "ok": False,
                    "reason": reason,
                    "error": f"agent-browser failed: exit={e.code}",
                }
            )

        log_path.write_text(
            log_path.read_text(encoding="utf-8") + f"\n[INDICATOR] {i}/{len(actions)} {label}={target} ({reason})\n",
            encoding="utf-8",
        )

    # Apply all draft targets via Smart Adjust
    try:
        _agent('find role button click --name "智能调整"', log_path)
        time.sleep(1.0)
        _agent("wait --load networkidle", log_path)
    except SystemExit as e:
        results.append({"i": 0, "label": "_smart_adjust", "ok": False, "error": f"smart adjust failed: exit={e.code}"})

    _agent(f'screenshot "{screenshots / "12_indicator_adjust.png"}"', log_path)

    (screenshots.parent / "indicator_actions_result.json").write_text(
        json.dumps({"results": results}, ensure_ascii=False, indent=2), encoding="utf-8"
    )


if __name__ == "__main__":
    main()

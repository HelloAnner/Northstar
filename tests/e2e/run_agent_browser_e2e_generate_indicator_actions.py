#!/usr/bin/env python3
import json
import math
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional


def _die(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(2)


def _load_json(path: Path) -> Dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def _unwrap_agent_browser_json(d: Dict[str, Any]) -> Dict[str, Any]:
    if not isinstance(d, dict):
        return {"error": "invalid json"}
    if d.get("success") is False:
        return {"error": d.get("error") or "agent-browser error"}
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


def _is_rate(label: str, unit: str) -> bool:
    if unit and "%" in unit:
        return True
    if "增速" in label or "%" in label:
        return True
    return False


def _next_value(base: float, label: str, unit: str) -> float:
    # Deterministic pseudo-random tweak based on label hash.
    seed = sum(ord(c) for c in label) % 7
    if _is_rate(label, unit):
        delta = (seed - 3) * 1.5
        if delta == 0:
            delta = 2.0
        return round(base + delta, 2)
    # Value indicators: bump by 3%~12%.
    ratio = 1.0 + (seed + 1) * 0.015
    if base == 0:
        return round(10.0 + seed * 3.0, 2)
    return round(base * ratio, 2)


def _force_notice_action(actions: List[Dict[str, Any]]) -> bool:
    def _norm(s: str) -> str:
        return str(s or "").replace(" ", "").replace("\u3000", "")

    for act in actions:
        label = _norm(act.get("label") or "")
        if "限上社零额增速" in label and "累计" in label:
            act["target"] = -200.0
            act["reason"] = "智能调整失败提示校验（强制负增速）"
            act["forceNotice"] = True
            return True

    for act in actions:
        if _is_rate(str(act.get("label") or ""), str(act.get("unit") or "")):
            act["target"] = -200.0
            act["reason"] = "智能调整失败提示校验（强制负增速）"
            act["forceNotice"] = True
            return True
    return False


def main() -> None:
    if len(sys.argv) != 3:
        print(
            "Usage: run_agent_browser_e2e_generate_indicator_actions.py <indicators_before.json> <indicator_actions.json>",
            file=sys.stderr,
        )
        raise SystemExit(2)

    in_path = Path(sys.argv[1])
    out_path = Path(sys.argv[2])

    payload = _unwrap_agent_browser_json(_load_json(in_path))
    indicators = payload.get("indicators") or []
    if not isinstance(indicators, list) or not indicators:
        out_path.write_text(
            json.dumps(
                {
                    "generatedAt": datetime.now().isoformat(timespec="seconds"),
                    "actions": [],
                    "error": "indicators_before.json has no indicators",
                },
                ensure_ascii=False,
                indent=2,
            ),
            encoding="utf-8",
        )
        print("indicator actions: 0 (no indicators)")
        return

    actions: List[Dict[str, Any]] = []
    for it in indicators:
        label = str(it.get("label") or "").strip()
        if not label:
            continue
        raw = it.get("value")
        unit = str(it.get("unit") or "").strip()
        base = _parse_number(raw)
        if base is None:
            base = 0.0
        target = _next_value(base, label, unit)
        actions.append(
            {
                "label": label,
                "unit": unit,
                "before": base,
                "target": target,
                "reason": f"指标联动/DAG：调整 {label}",
            }
        )

    forced_notice = _force_notice_action(actions)

    out = {
        "generatedAt": datetime.now().isoformat(timespec="seconds"),
        "actions": actions,
        "count": len(actions),
        "expectedCount": 16,
        "forcedNotice": forced_notice,
    }
    out_path.write_text(json.dumps(out, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"indicator actions: {len(actions)} -> {out_path}")


if __name__ == "__main__":
    main()

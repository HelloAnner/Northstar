#!/usr/bin/env python3
import json
import math
import random
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional


def _die(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(2)


def _load_json(path: Path) -> Dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def _parse_number(v: Any) -> Optional[float]:
    if v is None:
        return None
    if isinstance(v, (int, float)) and not isinstance(v, bool):
        if math.isfinite(float(v)):
            return float(v)
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


@dataclass(frozen=True)
class Action:
    credit_code: str
    column_label: str
    new_value: float
    reason: str


def _pick(rows: List[Dict[str, Any]], industry: str) -> Dict[str, Any]:
    cands = [r for r in rows if (r.get("__industry") or "").strip() == industry]
    if not cands:
        _die(f"no company found for industry={industry}")
    # Prefer rows that have a credit code
    for r in cands:
        if str(r.get("__creditCode") or "").strip() and str(r.get("__creditCode") or "").strip() != "-":
            return r
    return cands[0]


def main() -> None:
    if len(sys.argv) != 3:
        print("Usage: run_agent_browser_e2e_generate_actions.py <ui_before.json> <actions.json>", file=sys.stderr)
        raise SystemExit(2)

    ui_path = Path(sys.argv[1])
    out_path = Path(sys.argv[2])

    payload = _load_json(ui_path)
    # agent-browser --json wraps as {success,data:{result:...}}
    if isinstance(payload, dict) and isinstance(payload.get("data"), dict) and "result" in payload["data"]:
        payload = payload["data"]["result"]
    rows = payload.get("rows") or []
    if not isinstance(rows, list) or not rows:
        # Keep a deterministic output file so the runner can continue and report failures.
        out_path.write_text(
            json.dumps(
                {
                    "generatedAt": payload.get("extractedAt"),
                    "baseRowCount": payload.get("rowCount"),
                    "actions": [],
                    "picked": {},
                    "error": "ui_before.json has no rows",
                },
                ensure_ascii=False,
                indent=2,
            ),
            encoding="utf-8",
        )
        print("actions: 0 (ui_before has no rows)")
        return

    # Pick 1 company per industry to cover scenarios.
    wholesale = _pick(rows, "批发")
    retail = _pick(rows, "零售")
    accommodation = _pick(rows, "住宿")
    catering = _pick(rows, "餐饮")

    def cc(r: Dict[str, Any]) -> str:
        v = str(r.get("__creditCode") or "").strip()
        if not v or v == "-":
            _die("picked row has empty credit code")
        return v

    # Choose columns based on export template header labels (also present in table headers).
    # Keep action set small but representative:
    # - decimal
    # - zero
    # - negative
    # - multi-field across WR and AC
    actions: List[Action] = []

    def bump(r: Dict[str, Any], label: str, delta: float, reason: str) -> None:
        old = _parse_number(r.get(label))
        if old is None:
            old = 0.0
        actions.append(Action(credit_code=cc(r), column_label=label, new_value=round(old + delta, 2), reason=reason))

    def setv(r: Dict[str, Any], label: str, v: float, reason: str) -> None:
        actions.append(Action(credit_code=cc(r), column_label=label, new_value=round(v, 2), reason=reason))

    bump(wholesale, "商品销售额;本年-本月", 123.45, "批发：小数微调，覆盖 decimal 保存")
    setv(wholesale, "商品销售额;增速(当月)", 12.34, "批发：修改增速字段，覆盖 rate 保存")
    setv(retail, "零售额;本年-本月", 0.0, "零售：置 0，覆盖 zero 保存")
    setv(retail, "零售额;增速(当月)", -3.21, "零售：负增速，覆盖 rate/negative")
    bump(accommodation, "营业额;本年-本月", 50.0, "住宿：营业额微调，覆盖住餐主指标")
    setv(accommodation, "营业额;增速(当月)", 8.88, "住宿：修改增速字段，覆盖 rate 保存")
    bump(catering, "本月餐费收入", -10.0, "餐饮：负数微调，覆盖 negative 保存/校验")

    out = {
        "generatedAt": payload.get("extractedAt"),
        "baseRowCount": payload.get("rowCount"),
        "actions": [a.__dict__ for a in actions],
        "picked": {
            "批发": {"creditCode": cc(wholesale), "name": wholesale.get("__name")},
            "零售": {"creditCode": cc(retail), "name": retail.get("__name")},
            "住宿": {"creditCode": cc(accommodation), "name": accommodation.get("__name")},
            "餐饮": {"creditCode": cc(catering), "name": catering.get("__name")},
        },
    }

    out_path.write_text(json.dumps(out, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"actions: {len(actions)} -> {out_path}")


if __name__ == "__main__":
    main()

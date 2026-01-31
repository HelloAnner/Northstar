#!/usr/bin/env python3
import json
import math
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


def _pick_by_code(rows: List[Dict[str, Any]], credit_code: str) -> Optional[Dict[str, Any]]:
    cc = credit_code.strip()
    for r in rows:
        if str(r.get("__creditCode") or "").strip() == cc:
            return r
    return None


def _pick(rows: List[Dict[str, Any]], industry: str) -> Dict[str, Any]:
    cands = [r for r in rows if (r.get("__industry") or "").strip() == industry]
    if not cands:
        _die(f"no company found for industry={industry}")
    # Prefer rows that have a credit code; keep deterministic by sorting.
    cands = sorted(cands, key=lambda x: str(x.get("__creditCode") or ""))
    for r in cands:
        v = str(r.get("__creditCode") or "").strip()
        if v and v != "-":
            return r
    return cands[0]


def _pick_label(row: Dict[str, Any], candidates: List[str]) -> str:
    for c in candidates:
        if c in row:
            return c
    return candidates[0]


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
    retail = _pick_by_code(rows, "914401007RDD76M0RF") or _pick(rows, "零售")
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

    # 批发/零售口径优先用“销售额”列（可编辑）；“商品销售额”通常用于住餐口径
    sales_cur = _pick_label(wholesale, ["销售额;本年-本月", "商品销售额;本年-本月"])
    sales_last = _pick_label(wholesale, ["销售额;上年-本月", "商品销售额;上年-本月"])
    sales_rate = _pick_label(wholesale, ["销售额;增速(当月)", "商品销售额;增速(当月)"])
    sales_cum = _pick_label(wholesale, ["销售额;本年-1—本月", "商品销售额;本年-1—本月"])

    retail_cur = _pick_label(retail, ["零售额;本年-本月"])
    retail_last = _pick_label(retail, ["零售额;上年-本月"])
    retail_rate = _pick_label(retail, ["零售额;增速(当月)"])

    rev_cur = _pick_label(accommodation, ["营业额;本年-本月", "销售额;本年-本月", "商品销售额;本年-本月"])
    rev_rate = _pick_label(accommodation, ["营业额;增速(当月)", "销售额;增速(当月)", "商品销售额;增速(当月)"])
    room_cur = _pick_label(accommodation, ["本月客房收入", "客房收入;本年-本月"])

    food_cur = _pick_label(catering, ["本月餐费收入", "餐费收入;本年-本月"])
    goods_cur = _pick_label(catering, ["本月商品销售额", "商品销售额;本年-本月", "销售额;本年-本月"])

    bump(wholesale, sales_cur, 123.45, f"批发：小数微调（{sales_cur}），覆盖 decimal 保存")
    bump(wholesale, sales_last, 10.0, f"批发：上年同期微调（{sales_last}），覆盖 lastYearMonth editable 保存")
    setv(wholesale, sales_rate, 12.34, f"批发：修改增速字段（{sales_rate}），覆盖 rate 保存")
    bump(wholesale, sales_cum, 50.0, f"批发：累计微调（{sales_cum}），覆盖 cumulative editable 保存")

    setv(retail, retail_cur, 0.0, f"零售：置 0（{retail_cur}），覆盖 zero 保存")
    bump(retail, retail_last, 1.0, f"零售：上年同期微调（{retail_last}），覆盖 lastYearMonth editable 保存")
    setv(retail, retail_rate, -3.21, f"零售：负增速（{retail_rate}），覆盖 rate/negative 保存")

    bump(accommodation, rev_cur, 50.0, f"住宿：营业额微调（{rev_cur}），覆盖住餐主指标 editable 保存")
    setv(accommodation, rev_rate, 8.88, f"住宿：修改增速字段（{rev_rate}），覆盖 rate editable 保存")
    bump(accommodation, room_cur, 10.0, f"住宿：客房收入微调（{room_cur}），覆盖子指标 editable 保存")

    bump(catering, food_cur, -10.0, f"餐饮：负数微调（{food_cur}），覆盖 negative 保存/校验")
    bump(catering, goods_cur, 5.55, f"餐饮：商品销售额微调（{goods_cur}），覆盖 goods editable 保存/校验")

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

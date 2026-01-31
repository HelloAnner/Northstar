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

def _pick_many(rows: List[Dict[str, Any]], industry: str, n: int) -> List[Dict[str, Any]]:
    cands = [r for r in rows if (r.get("__industry") or "").strip() == industry]
    cands = sorted(cands, key=lambda x: str(x.get("__creditCode") or ""))
    out: List[Dict[str, Any]] = []
    seen: set[str] = set()
    for r in cands:
        code = str(r.get("__creditCode") or "").strip()
        if not code or code == "-" or code in seen:
            continue
        seen.add(code)
        out.append(r)
        if len(out) >= n:
            break
    if not out:
        _die(f"no company found for industry={industry}")
    return out


def _pick_label(row: Dict[str, Any], candidates: List[str]) -> str:
    for c in candidates:
        if c in row:
            return c
    # Fallback: candidate like "销售额;本年-本月" but UI might render just "本年-本月"
    for c in candidates:
        if ";" in c:
            s = c.split(";", 1)[1].strip()
            if s and s in row:
                return s
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
    wholesale_list = _pick_many(rows, "批发", 2)
    retail_main = _pick_by_code(rows, "914401007RDD76M0RF")
    retail_list = [retail_main] if retail_main else []
    for r in _pick_many(rows, "零售", 2):
        if retail_main and str(r.get("__creditCode") or "").strip() == str(retail_main.get("__creditCode") or "").strip():
            continue
        retail_list.append(r)
    retail_list = retail_list[:2]

    accommodation_list = _pick_many(rows, "住宿", 2)
    catering_list = _pick_many(rows, "餐饮", 2)

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

    # 批发/零售口径：UI 主销售额列为“本年-本月/上年-本月/同比增速(当月)/本年-1—本月/累计同比增速”
    # 最新 UI（web/src/components/CompaniesTable.tsx）主销售额列名为：本年-本月 / 上年-本月 / 同比增速(当月) / 本年-1—本月 / 累计同比增速
    wholesale1 = wholesale_list[0]
    wholesale2 = wholesale_list[-1]
    sales_cur = _pick_label(wholesale1, ["本年-本月", "销售额;本年-本月", "商品销售额;本年-本月"])
    sales_last = _pick_label(wholesale1, ["上年-本月", "销售额;上年-本月", "商品销售额;上年-本月"])
    sales_rate = _pick_label(wholesale1, ["同比增速(当月)", "销售额;增速(当月)", "商品销售额;增速(当月)"])
    sales_cum = _pick_label(wholesale1, ["本年-1—本月", "销售额;本年-1—本月", "商品销售额;本年-1—本月"])
    sales_cum_rate = _pick_label(wholesale1, ["累计同比增速", "销售额;累计增速", "商品销售额;累计增速"])

    retail1 = retail_list[0]
    retail2 = retail_list[-1]
    retail_cur = _pick_label(retail1, ["零售额;本年-本月"])
    retail_last = _pick_label(retail1, ["零售额;上年-本月"])
    retail_rate = _pick_label(retail1, ["零售额;同比增速(当月)", "零售额;增速(当月)"])

    # 住宿/餐饮在 UI 中复用“本年-本月”等列（内部映射到 revenue* 字段）；不依赖客房/餐费/商品销售额子指标列（当前 UI 未展示）
    accommodation1 = accommodation_list[0]
    accommodation2 = accommodation_list[-1]
    rev_cur = _pick_label(accommodation1, ["本年-本月", "营业额;本年-本月", "销售额;本年-本月"])
    rev_rate = _pick_label(accommodation1, ["同比增速(当月)", "营业额;增速(当月)", "销售额;增速(当月)"])

    catering1 = catering_list[0]
    catering2 = catering_list[-1]
    catering_sales_cur = _pick_label(catering1, ["本年-本月", "营业额;本年-本月", "销售额;本年-本月"])
    catering_rate = _pick_label(catering1, ["同比增速(当月)", "营业额;增速(当月)", "销售额;增速(当月)"])

    bump(wholesale1, sales_cur, 123.0, f"批发(1)：微调（{sales_cur}），覆盖保存")
    bump(wholesale1, sales_last, 10.0, f"批发(1)：上年同期微调（{sales_last}），覆盖 lastYearMonth editable 保存")
    setv(wholesale1, sales_rate, 12.0, f"批发(1)：修改增速字段（{sales_rate}），覆盖 rate 保存")
    bump(wholesale1, sales_cum, 50.0, f"批发(1)：累计微调（{sales_cum}），覆盖 cumulative editable 保存")
    setv(wholesale1, sales_cum_rate, 6.0, f"批发(1)：累计增速修改（{sales_cum_rate}），覆盖 cumulative rate 保存")
    # 第二个批发企业：覆盖 0 / 大数
    sales_cur2 = _pick_label(wholesale2, ["本年-本月", "销售额;本年-本月"])
    setv(wholesale2, sales_cur2, 0.0, f"批发(2)：本年-本月置 0（{sales_cur2}），覆盖 zero 保存")

    setv(retail1, retail_cur, 0.0, f"零售(1)：置 0（{retail_cur}），覆盖 zero 保存")
    bump(retail1, retail_last, 1.0, f"零售(1)：上年同期微调（{retail_last}），覆盖 lastYearMonth editable 保存")
    setv(retail1, retail_rate, -3.0, f"零售(1)：负增速（{retail_rate}），覆盖 rate/negative 保存")
    # 第二个零售企业：覆盖大值 & 搜索路径（仍用信用代码）
    retail_cur2 = _pick_label(retail2, ["零售额;本年-本月"])
    bump(retail2, retail_cur2, 999.0, f"零售(2)：本年-本月大幅微调（{retail_cur2}），覆盖大数保存")

    bump(accommodation1, rev_cur, 50.0, f"住宿(1)：营业额微调（{rev_cur}），覆盖住餐主指标 editable 保存")
    setv(accommodation1, rev_rate, 9.0, f"住宿(1)：修改增速字段（{rev_rate}），覆盖 rate editable 保存")
    rev_cur2 = _pick_label(accommodation2, ["本年-本月", "营业额;本年-本月"])
    bump(accommodation2, rev_cur2, -20.0, f"住宿(2)：负数微调（{rev_cur2}），覆盖 negative 保存")

    bump(catering1, catering_sales_cur, 20.0, f"餐饮(1)：营业额微调（{catering_sales_cur}），覆盖保存")
    setv(catering1, catering_rate, -7.0, f"餐饮(1)：负增速（{catering_rate}），覆盖 rate/negative 保存")
    catering_sales_cur2 = _pick_label(catering2, ["本年-本月", "营业额;本年-本月"])
    bump(catering2, catering_sales_cur2, 333.0, f"餐饮(2)：本年-本月大幅微调（{catering_sales_cur2}），覆盖大数保存")

    out = {
        "generatedAt": payload.get("extractedAt"),
        "baseRowCount": payload.get("rowCount"),
        "actions": [a.__dict__ for a in actions],
        "picked": {
            "批发": {"creditCode": cc(wholesale1), "name": wholesale1.get("__name")},
            "批发2": {"creditCode": cc(wholesale2), "name": wholesale2.get("__name")},
            "零售": {"creditCode": cc(retail1), "name": retail1.get("__name")},
            "零售2": {"creditCode": cc(retail2), "name": retail2.get("__name")},
            "住宿": {"creditCode": cc(accommodation1), "name": accommodation1.get("__name")},
            "住宿2": {"creditCode": cc(accommodation2), "name": accommodation2.get("__name")},
            "餐饮": {"creditCode": cc(catering1), "name": catering1.get("__name")},
            "餐饮2": {"creditCode": cc(catering2), "name": catering2.get("__name")},
        },
    }

    out_path.write_text(json.dumps(out, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"actions: {len(actions)} -> {out_path}")


if __name__ == "__main__":
    main()

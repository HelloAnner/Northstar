#!/usr/bin/env python3
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, Optional
from urllib import request, parse


def _read_json(url: str) -> Dict[str, Any]:
    with request.urlopen(url, timeout=60) as resp:
        data = resp.read().decode("utf-8")
    return json.loads(data)


def _post_json(url: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    body = json.dumps(payload).encode("utf-8")
    req = request.Request(url, data=body, headers={"Content-Type": "application/json"}, method="POST")
    with request.urlopen(req, timeout=60) as resp:
        data = resp.read().decode("utf-8")
    return json.loads(data)


def _download(url: str, out_path: Path) -> Dict[str, Any]:
    req = request.Request(url, data=b"", method="POST")
    with request.urlopen(req, timeout=600) as resp:
        out_path.write_bytes(resp.read())
        headers = dict(resp.headers.items())
    return headers


def _parse_filename(headers: Dict[str, Any]) -> str:
    cd = str(headers.get("Content-Disposition") or "")
    if not cd:
        return ""
    # Try filename*= first
    parts = cd.split(";")
    for part in parts:
        part = part.strip()
        if part.lower().startswith("filename*="):
            val = part.split("=", 1)[1]
            if "''" in val:
                return parse.unquote(val.split("''", 1)[1])
            return parse.unquote(val)
    for part in parts:
        part = part.strip()
        if part.lower().startswith("filename="):
            val = part.split("=", 1)[1].strip()
            return val.strip('"')
    return ""


def main() -> None:
    if len(sys.argv) != 4:
        print("Usage: run_agent_browser_e2e_export_param.py <baseUrl> <exportAltPath> <metaPath>", file=sys.stderr)
        raise SystemExit(2)

    base_url = sys.argv[1].rstrip("/")
    export_path = Path(sys.argv[2])
    meta_path = Path(sys.argv[3])

    meta: Dict[str, Any] = {
        "generatedAt": datetime.now().isoformat(timespec="seconds"),
        "ok": False,
        "skipped": False,
        "reason": "",
    }

    try:
        months = _read_json(f"{base_url}/api/months")
        current_year = int(months.get("currentYear") or 0)
        current_month = int(months.get("currentMonth") or 0)
        items = months.get("items") or []
        alt = None
        for it in items:
            y = int(it.get("year") or 0)
            m = int(it.get("month") or 0)
            total = int(it.get("total") or 0)
            if y == current_year and m == current_month:
                continue
            if total <= 0:
                continue
            alt = {"year": y, "month": m}
            break

        meta["primary"] = {"year": current_year, "month": current_month}

        if not alt:
            meta.update({"skipped": True, "reason": "可用月份不足(需要至少2个月)"})
            meta_path.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")
            print("export param: skipped (not enough months)")
            return

        _post_json(f"{base_url}/api/months/select", alt)

        headers = _download(f"{base_url}/api/export", export_path)
        file_name = _parse_filename(headers)
        meta["alternate"] = {"year": alt["year"], "month": alt["month"], "fileName": file_name}

        # switch back
        _post_json(f"{base_url}/api/months/select", {"year": current_year, "month": current_month})

        meta["ok"] = True
        meta_path.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")
        print(f"export param: ok -> {export_path}")
    except Exception as e:
        meta["reason"] = str(e)
        meta_path.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")
        print(f"export param: fail ({e})")
        raise SystemExit(1)


if __name__ == "__main__":
    main()

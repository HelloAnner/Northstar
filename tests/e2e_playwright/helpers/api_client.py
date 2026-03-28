"""NorthStar HTTP API 客户端，用于测试数据准备和验证"""
from __future__ import annotations

import time
from pathlib import Path
from typing import Any

import httpx


class ApiClient:
    """封装 NorthStar REST API 调用"""

    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")
        self.client = httpx.Client(
            base_url=self.base_url, timeout=30,
            transport=httpx.HTTPTransport(proxy=None),
        )

    def close(self) -> None:
        self.client.close()

    # ---- 状态 ----
    def status(self) -> dict:
        return self.client.get("/api/status").json()

    def is_initialized(self) -> bool:
        return self.status().get("initialized", False)

    # ---- 配置 ----
    def get_config(self) -> dict:
        return self.client.get("/api/config").json()

    def patch_config(self, updates: dict) -> dict:
        return self.client.patch("/api/config", json={"updates": updates}).json()

    # ---- 导入 ----
    def import_excel(self, file_path: str | Path, clear_existing: bool = True) -> list[dict]:
        """通过 API 导入 Excel，收集 SSE 事件"""
        path = Path(file_path)
        events = []
        with open(path, "rb") as f:
            with self.client.stream(
                "POST",
                "/api/import",
                files={"file": (path.name, f, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")},
                data={"clearExisting": str(clear_existing).lower(), "updateConfigYM": "true"},
                timeout=120,
            ) as resp:
                for line in resp.iter_lines():
                    if line.startswith("data: "):
                        import json
                        try:
                            events.append(json.loads(line[6:]))
                        except json.JSONDecodeError:
                            pass
        return events

    def import_preview(self) -> dict:
        return self.client.get("/api/import/preview").json()

    # ---- 企业 ----
    def companies(self, industry_type: str = "all", page: int = 1, page_size: int = 200, keyword: str = "") -> dict:
        params = {"industryType": industry_type, "page": page, "pageSize": page_size}
        if keyword:
            params["keyword"] = keyword
        return self.client.get("/api/companies", params=params).json()

    def patch_company(self, company_id: str, patch: dict) -> dict:
        return self.client.patch(f"/api/companies/{company_id}", json=patch).json()

    def reset_companies(self, company_ids: list[str] | None = None) -> dict:
        return self.client.post("/api/companies/reset", json={"companyIds": company_ids or []}).json()

    # ---- 指标 ----
    def indicators(self) -> dict:
        return self.client.get("/api/indicators").json()

    # ---- 规则 ----
    def get_rules(self) -> list[dict]:
        return self.client.get("/api/rules").json()

    def add_rule(self, text: str) -> list[dict]:
        return self.client.post("/api/rules", json={"text": text}).json()

    def update_rule(self, index: int, text: str) -> list[dict]:
        return self.client.put(f"/api/rules/{index}", json={"text": text}).json()

    def delete_rule(self, index: int) -> list[dict]:
        return self.client.delete(f"/api/rules/{index}").json()

    def convert_rules(self) -> dict:
        return self.client.post("/api/rules/convert").json()

    def rules_status(self) -> dict:
        return self.client.get("/api/rules/status").json()

    def wait_rules_convert(self, timeout: int = 30) -> dict:
        """等待规则转换完成"""
        deadline = time.time() + timeout
        while time.time() < deadline:
            st = self.rules_status()
            if st.get("status") in ("ok", "error", "idle"):
                return st
            time.sleep(1)
        raise TimeoutError("Rules conversion timed out")

    def clear_all_rules(self) -> None:
        """清空全部规则"""
        rules = self.get_rules()
        for i in range(len(rules), 0, -1):
            self.delete_rule(i)

    # ---- 优化 ----
    def optimize(self, targets: dict) -> dict:
        return self.client.post("/api/optimize", json={"targets": targets}).json()

    # ---- 导出 ----
    def export_stream_download(self, output_path: str | Path) -> dict:
        """通过 SSE 导出并下载文件"""
        import json
        events = []
        download_url = None
        with self.client.stream("POST", "/api/export/stream", timeout=120) as resp:
            for line in resp.iter_lines():
                if line.startswith("data: "):
                    try:
                        evt = json.loads(line[6:])
                        events.append(evt)
                        if evt.get("type") == "done" and evt.get("data", {}).get("downloadUrl"):
                            download_url = evt["data"]["downloadUrl"]
                    except json.JSONDecodeError:
                        pass
        if download_url:
            dl = self.client.get(download_url)
            Path(output_path).write_bytes(dl.content)
        return {"events": events, "downloaded": download_url is not None}

    # ---- 设置 ----
    def get_user_prompt(self) -> str:
        return self.client.get("/api/settings/user-prompt").json().get("content", "")

    def set_user_prompt(self, content: str) -> dict:
        return self.client.put("/api/settings/user-prompt", json={"content": content}).json()

    # ---- LLM Chat ----
    def chat_stream(self, messages: list[dict], mode: str = "chat", session_id: str = "test") -> list[dict]:
        """发送聊天请求，收集 SSE 事件"""
        import json
        events = []
        with self.client.stream(
            "POST",
            "/api/llm/chat/stream",
            json={"sessionId": session_id, "mode": mode, "messages": messages},
            timeout=60,
        ) as resp:
            for line in resp.iter_lines():
                if line.startswith("data: "):
                    try:
                        events.append(json.loads(line[6:]))
                    except json.JSONDecodeError:
                        pass
        return events

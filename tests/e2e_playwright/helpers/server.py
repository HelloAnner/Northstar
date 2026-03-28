"""NorthStar 服务器生命周期管理"""
from __future__ import annotations

import os
import shutil
import signal
import subprocess
import tempfile
import time
from pathlib import Path

import httpx


class NorthstarServer:
    """构建、启动、停止 NorthStar 服务器"""

    def __init__(self, port: int = 18080):
        self.port = port
        self.base_url = f"http://localhost:{port}"
        self.repo_root = Path(__file__).resolve().parents[3]
        self.work_dir = Path(tempfile.mkdtemp(prefix="northstar_e2e_"))
        self.data_dir = self.work_dir / "data"
        self.binary = self.work_dir / "northstar"
        self.process: subprocess.Popen | None = None

    def build(self) -> None:
        """编译 NorthStar binary"""
        env = os.environ.copy()
        result = subprocess.run(
            ["go", "build", "-o", str(self.binary), "./cmd/northstar"],
            cwd=self.repo_root,
            env=env,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            raise RuntimeError(f"go build failed:\n{result.stderr}")

    def _write_config(self) -> None:
        """写入 config.toml"""
        self.data_dir.mkdir(parents=True, exist_ok=True)
        config = self.work_dir / "config.toml"
        config.write_text(
            f'[server]\nport = {self.port}\ndev_mode = false\n\n'
            f'[data]\ndata_dir = "data"\nauto_backup = false\n\n'
            f'[business]\ndefault_month = 1\nmax_growth = 0.5\nmin_growth = -0.3\n\n'
            f'[excel]\ntemplate_path = ""\n',
            encoding="utf-8",
        )

    def _http_client(self) -> httpx.Client:
        """创建不走代理的 HTTP 客户端"""
        return httpx.Client(timeout=5, proxy=None, transport=httpx.HTTPTransport(proxy=None))

    def start(self, timeout: int = 30) -> None:
        """启动服务器并等待就绪"""
        self._write_config()
        env = os.environ.copy()
        env["NORTHSTAR_PORT"] = str(self.port)
        self.process = subprocess.Popen(
            [str(self.binary)],
            cwd=self.work_dir,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )
        # 等待服务器就绪
        deadline = time.time() + timeout
        client = self._http_client()
        try:
            while time.time() < deadline:
                try:
                    r = client.get(f"{self.base_url}/api/status")
                    if r.status_code == 200:
                        return
                except (httpx.ConnectError, httpx.ReadTimeout):
                    pass
                time.sleep(0.5)
        finally:
            client.close()
        # 获取进程输出帮助调试
        stdout = ""
        if self.process and self.process.stdout:
            import select
            if select.select([self.process.stdout], [], [], 0)[0]:
                stdout = self.process.stdout.read(4096).decode("utf-8", errors="replace")
        exit_code = self.process.poll() if self.process else "no process"
        self.stop()
        raise RuntimeError(
            f"Server failed to start within {timeout}s. "
            f"Exit code: {exit_code}, Output: {stdout[:1000]}"
        )

    def configure_llm(self) -> bool:
        """从环境变量配置 LLM，返回是否配置成功"""
        base_url = os.environ.get("DEEPSEEK_BASE_URL") or os.environ.get("LLM_BASE_URL")
        model = os.environ.get("DEEPSEEK_MODEL_NAME") or os.environ.get("LLM_MODEL_NAME") or os.environ.get("LLM_MODEL")
        api_key = os.environ.get("DEEPSEEK_API_KEY") or os.environ.get("LLM_API_KEY")
        if not all([base_url, model, api_key]):
            return False
        client = self._http_client()
        try:
            r = client.patch(
                f"{self.base_url}/api/config",
                json={"updates": {"llm_base_url": base_url, "llm_model": model, "llm_api_key": api_key}},
            )
        finally:
            client.close()
        return r.status_code == 200

    def stop(self) -> None:
        """停止服务器"""
        if self.process and self.process.poll() is None:
            self.process.send_signal(signal.SIGTERM)
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=3)
        self.process = None

    def cleanup(self) -> None:
        """清理临时目录"""
        self.stop()
        if self.work_dir.exists():
            shutil.rmtree(self.work_dir, ignore_errors=True)

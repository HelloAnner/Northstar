"""截图辅助函数"""
from __future__ import annotations

import time
from pathlib import Path

from playwright.sync_api import Page


def take_screenshot(page: Page, output_dir: Path, name: str) -> Path:
    """截图并返回路径"""
    output_dir.mkdir(parents=True, exist_ok=True)
    ts = int(time.time() * 1000)
    path = output_dir / f"{name}_{ts}.png"
    page.screenshot(path=str(path), full_page=False)
    return path


def take_step_screenshot(page: Page, output_dir: Path, test_name: str, step: str) -> Path:
    """关键步骤截图"""
    return take_screenshot(page, output_dir, f"{test_name}_{step}")

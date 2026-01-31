#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
SESSION="${AGENT_BROWSER_SESSION:-default}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RESULTS_ROOT="${RESULTS_ROOT:-$REPO_ROOT/tests/e2e-result}"

INPUT_XLSX="${INPUT_XLSX:-$REPO_ROOT/prd/12月月报（预估）_补全企业名称社会代码_20260129.xlsx}"

timestamp="$(date +%Y%m%d-%H%M%S)"
RUN_DIR="$RESULTS_ROOT/run-$timestamp"
SCREENSHOTS_DIR="$RUN_DIR/screenshots"
mkdir -p "$SCREENSHOTS_DIR"

LOG="$RUN_DIR/agent-browser.log"
META="$RUN_DIR/meta.json"
UI_BEFORE="$RUN_DIR/ui_companies_before.json"
UI_AFTER="$RUN_DIR/ui_companies_after.json"
IMPORT_EVENTS="$RUN_DIR/import_events.json"
EXPORT_XLSX="$RUN_DIR/export.xlsx"
CONSOLE_LOG="$RUN_DIR/browser_console.txt"
ERRORS_LOG="$RUN_DIR/browser_errors.txt"
TRACE_ZIP="$RUN_DIR/trace.zip"
VIDEO_WEBM="$RUN_DIR/run.webm"
REPORT_HTML="$RUN_DIR/report.html"
ACTION_RESULTS_JSON="$RUN_DIR/actions_result.json"
STEPS_JSON="$RUN_DIR/steps.json"
STEPS_JSONL="$RUN_DIR/steps.jsonl"

if [[ ! -f "$INPUT_XLSX" ]]; then
  echo "ERROR: INPUT_XLSX not found: $INPUT_XLSX" | tee -a "$LOG" >&2
  exit 2
fi

export BASE_URL
export AGENT_BROWSER_SESSION="$SESSION"
export RUN_DIR
export INPUT_XLSX

python3 - <<'PY' >"$META"
import json, os, datetime
print(json.dumps({
  "baseUrl": os.environ.get("BASE_URL", ""),
  "session": os.environ.get("AGENT_BROWSER_SESSION", ""),
  "inputXlsx": os.environ.get("INPUT_XLSX", ""),
  "runDir": os.environ.get("RUN_DIR", ""),
  "startedAt": datetime.datetime.now().isoformat(timespec="seconds"),
}, ensure_ascii=False, indent=2))
PY

{
  echo "=== Northstar agent-browser e2e ==="
  echo "BASE_URL=$BASE_URL"
  echo "INPUT_XLSX=$INPUT_XLSX"
  echo "RUN_DIR=$RUN_DIR"
  echo "SESSION=$SESSION"
  echo ""
} | tee -a "$LOG"

record_step() {
  local name="$1"
  local status="$2"
  local detail="${3:-}"
  local reproduce="${4:-}"
  python3 - "$name" "$status" "$detail" "$reproduce" <<'PY' >>"$STEPS_JSONL"
import json, sys, datetime
name, status, detail, reproduce = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
print(json.dumps({
  "ts": datetime.datetime.now().isoformat(timespec="seconds"),
  "name": name,
  "status": status,
  "detail": detail,
  "reproduce": reproduce,
}, ensure_ascii=False))
PY
}

finalize_steps() {
  python3 - "$STEPS_JSONL" "$STEPS_JSON" <<'PY'
import json, sys, pathlib
src = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])
items = []
if src.exists():
  for line in src.read_text(encoding="utf-8", errors="replace").splitlines():
    line = line.strip()
    if not line:
      continue
    try:
      items.append(json.loads(line))
    except Exception:
      items.append({"ts":"", "name":"_parse_error", "status":"fail", "detail": line, "reproduce": ""})
out.write_text(json.dumps(items, ensure_ascii=False, indent=2), encoding="utf-8")
PY
}

on_exit() {
  finalize_steps || true
}
trap on_exit EXIT

# Ensure required browser is available for agent-browser (playwright-core@1.58.1 expects cft v1208).
if [[ ! -d "$HOME/Library/Caches/ms-playwright/chromium_headless_shell-1208" ]]; then
  echo ">>> Installing Playwright chromium-headless-shell v1208 (needed by agent-browser)..." | tee -a "$LOG"
  if npx -y playwright@1.58.1 install chromium-headless-shell | tee -a "$LOG"; then
    record_step "install_playwright_browsers" "pass" "" ""
  else
    record_step "install_playwright_browsers" "fail" "npx playwright install failed" "在本机执行：npx -y playwright@1.58.1 install chromium-headless-shell"
  fi
fi

echo ">>> Ensuring agent-browser browser binaries..." | tee -a "$LOG"
if agent-browser install | tee -a "$LOG"; then
  record_step "agent_browser_install" "pass" "" ""
else
  record_step "agent_browser_install" "fail" "agent-browser install failed" "在本机执行：agent-browser install"
fi

echo ">>> Cleaning stale agent-browser sockets..." | tee -a "$LOG"
rm -f "$HOME/.agent-browser/default.sock" "$HOME/.agent-browser/default.pid" 2>/dev/null || true
rm -f "$HOME/.agent-browser/${SESSION}.sock" "$HOME/.agent-browser/${SESSION}.pid" 2>/dev/null || true

echo ">>> Opening $BASE_URL ..." | tee -a "$LOG"
if agent-browser open "$BASE_URL" | tee -a "$LOG"; then
  record_step "open_home" "pass" "" ""
else
  record_step "open_home" "fail" "failed to open BASE_URL" "确认服务可访问后重试：BASE_URL=http://localhost:8080 make test-e2e"
fi
agent-browser set viewport 1440 900 | tee -a "$LOG" || true
agent-browser wait --load networkidle | tee -a "$LOG" || true
agent-browser record start "$VIDEO_WEBM" "$BASE_URL" | tee -a "$LOG" || true
agent-browser trace start | tee -a "$LOG" || true
agent-browser screenshot "$SCREENSHOTS_DIR/00_home.png" | tee -a "$LOG" || true

echo ">>> Preparing table columns (enable all)..." | tee -a "$LOG"
if agent-browser eval "(() => { const keys = ['companyScale','flags','salesCurrentMonth','salesLastYearMonth','salesMonthRate','salesCurrentCumulative','salesLastYearCumulative','salesCumulativeRate','retailCurrentMonth','retailLastYearMonth','retailMonthRate','retailCurrentCumulative','retailLastYearCumulative','retailCumulativeRate','retailRatio','revenueCurrentMonth','revenueLastYearMonth','revenueMonthRate','revenueCurrentCumulative','revenueLastYearCumulative','revenueCumulativeRate','roomCurrentMonth','foodCurrentMonth','goodsCurrentMonth','sourceSheet']; localStorage.setItem('northstar.visibleColumns', JSON.stringify(keys)); return {visibleColumns: keys.length}; })()" | tee -a "$LOG"; then
  record_step "prepare_columns" "pass" "" ""
else
  record_step "prepare_columns" "fail" "failed to set visible columns in localStorage" "打开页面后 F12 Console 执行：localStorage.getItem('northstar.visibleColumns')"
fi
agent-browser reload | tee -a "$LOG" || true
agent-browser wait --load networkidle | tee -a "$LOG" || true
agent-browser screenshot "$SCREENSHOTS_DIR/01_all_columns.png" | tee -a "$LOG" || true

echo ">>> Importing Excel via UI..." | tee -a "$LOG"
if agent-browser find role button click --name "导入" | tee -a "$LOG"; then
  record_step "open_import_dialog" "pass" "" ""
else
  record_step "open_import_dialog" "fail" "找不到/无法点击 导入 按钮" "打开首页后点击右上角“导入”"
fi
agent-browser wait --text "导入数据" | tee -a "$LOG" || true
if agent-browser upload "input[type=file]" "$INPUT_XLSX" | tee -a "$LOG"; then
  record_step "upload_excel" "pass" "" ""
else
  record_step "upload_excel" "fail" "上传文件失败" "在导入弹窗中选择文件并点击开始导入"
fi
agent-browser find role button click --name "开始导入" | tee -a "$LOG" || true
if agent-browser wait --text "完成" | tee -a "$LOG"; then
  record_step "import_done" "pass" "" ""
else
  record_step "import_done" "fail" "导入未在预期时间内完成（UI未出现“完成”）" "导入后观察导入弹窗进度与日志"
fi
agent-browser screenshot "$SCREENSHOTS_DIR/02_import_done.png" | tee -a "$LOG" || true

echo ">>> Capturing import progress events..." | tee -a "$LOG"
agent-browser eval "(() => { const items = Array.from(document.querySelectorAll('[role=\"dialog\"] [data-radix-scroll-area-viewport] .text-sm')).map(el => el.textContent?.trim()).filter(Boolean); return {count: items.length, items}; })()" --json >"$IMPORT_EVENTS" || true

agent-browser find role button click --name "完成" | tee -a "$LOG" || true
agent-browser wait --load networkidle | tee -a "$LOG" || true
agent-browser screenshot "$SCREENSHOTS_DIR/03_after_import.png" | tee -a "$LOG" || true

echo ">>> Extracting companies table (before modifications)..." | tee -a "$LOG"
agent-browser find role tab click --name "全部" | tee -a "$LOG" || true
agent-browser find placeholder "按企业名称/信用代码搜索…" fill "" | tee -a "$LOG" || true
agent-browser wait --fn "document.querySelectorAll('tbody tr td .font-mono').length >= 10" | tee -a "$LOG" || true
if agent-browser eval "(() => { const table = document.querySelector('table'); if (!table) return { error: 'table not found' }; const headers = Array.from(table.querySelectorAll('thead th')).map(th => (th.textContent || '').trim()); const bodyRows = Array.from(table.querySelectorAll('tbody tr')); const rows = []; for (const tr of bodyRows) { const tds = Array.from(tr.querySelectorAll('td')); if (tds.length < 2) continue; const name = (tds[0].querySelector('.truncate.font-medium')?.textContent || '').trim(); const credit = (tds[0].querySelector('.font-mono')?.textContent || '').trim(); if (!name && !credit) continue; const badge = (tds[0].querySelector('.h-5')?.textContent || '').trim(); const row = { __name: name, __creditCode: credit, __industry: badge }; for (let i = 1; i < tds.length && i < headers.length; i++) { const header = headers[i] || ('col_' + String(i)); const cell = tds[i]; const input = cell.querySelector('input'); let v = ''; if (input) v = String(input.value || ''); else v = String((cell.textContent || '').trim()); row[header] = v; } rows.push(row); } return { extractedAt: new Date().toISOString(), headers, rowCount: rows.length, rows }; })()" --json >"$UI_BEFORE"; then
  record_step "extract_ui_before" "pass" "" ""
else
  record_step "extract_ui_before" "fail" "UI 明细表抽取失败（可能是页面 JS 异常/结构变化）" "导入后在首页打开明细表，确认表格可见后重试"
fi

echo ">>> Applying modifications (cover multiple scenarios)..." | tee -a "$LOG"
if python3 "$REPO_ROOT/tests/e2e/run_agent_browser_e2e_generate_actions.py" "$UI_BEFORE" "$RUN_DIR/actions.json" | tee -a "$LOG"; then
  record_step "generate_actions" "pass" "" ""
else
  record_step "generate_actions" "fail" "生成修改动作失败（可能是明细表为空/字段缺失）" "查看 ui_companies_before.json 是否包含 rows"
fi

echo ">>> Executing modification actions via browser..." | tee -a "$LOG"
if [[ -f "$RUN_DIR/actions.json" ]]; then
  if python3 "$REPO_ROOT/tests/e2e/run_agent_browser_e2e_exec_actions.py" "$RUN_DIR/actions.json" "$LOG" "$SCREENSHOTS_DIR" | tee -a "$LOG"; then
    record_step "execute_actions" "pass" "" ""
  else
    record_step "execute_actions" "fail" "执行修改动作时出现失败（详见日志与截图）" "按 actions.json 中的信用代码搜索企业，修改指定字段后 blur 保存"
  fi
else
  record_step "execute_actions" "fail" "actions.json 不存在（跳过修改动作）" "先修复导入后明细表抽取/生成动作逻辑，再重跑 make test-e2e"
fi

echo ">>> Extracting companies table (after modifications)..." | tee -a "$LOG"
agent-browser wait --load networkidle | tee -a "$LOG" || true
agent-browser screenshot "$SCREENSHOTS_DIR/10_after_modifications.png" | tee -a "$LOG" || true
agent-browser reload | tee -a "$LOG" || true
agent-browser wait --load networkidle | tee -a "$LOG" || true
agent-browser find role tab click --name "全部" | tee -a "$LOG" || true
agent-browser find placeholder "按企业名称/信用代码搜索…" fill "" | tee -a "$LOG" || true
agent-browser wait --fn "document.querySelectorAll('tbody tr td .font-mono').length >= 10" | tee -a "$LOG" || true
if agent-browser eval "(() => { const table = document.querySelector('table'); if (!table) return { error: 'table not found' }; const headers = Array.from(table.querySelectorAll('thead th')).map(th => (th.textContent || '').trim()); const bodyRows = Array.from(table.querySelectorAll('tbody tr')); const rows = []; for (const tr of bodyRows) { const tds = Array.from(tr.querySelectorAll('td')); if (tds.length < 2) continue; const name = (tds[0].querySelector('.truncate.font-medium')?.textContent || '').trim(); const credit = (tds[0].querySelector('.font-mono')?.textContent || '').trim(); if (!name && !credit) continue; const badge = (tds[0].querySelector('.h-5')?.textContent || '').trim(); const row = { __name: name, __creditCode: credit, __industry: badge }; for (let i = 1; i < tds.length && i < headers.length; i++) { const header = headers[i] || ('col_' + String(i)); const cell = tds[i]; const input = cell.querySelector('input'); let v = ''; if (input) v = String(input.value || ''); else v = String((cell.textContent || '').trim()); row[header] = v; } rows.push(row); } return { extractedAt: new Date().toISOString(), headers, rowCount: rows.length, rows }; })()" --json >"$UI_AFTER"; then
  record_step "extract_ui_after" "pass" "" ""
else
  record_step "extract_ui_after" "fail" "UI 明细表抽取失败（修改后）" "修改后刷新页面，确认明细表可见"
fi

echo ">>> Exporting Excel via UI..." | tee -a "$LOG"
if agent-browser download "text=导出" "$EXPORT_XLSX" 2>>"$LOG"; then
  record_step "export_download" "pass" "" ""
else
  echo "WARN: download selector failed, fallback to click + wait --download" | tee -a "$LOG"
  agent-browser find role button click --name "导出" | tee -a "$LOG" || true
  if agent-browser wait --download "$EXPORT_XLSX" --timeout 60000 | tee -a "$LOG"; then
    record_step "export_download" "pass" "fallback via wait --download" ""
  else
    record_step "export_download" "fail" "导出下载失败（浏览器未捕获下载）" "手动点击“导出”，观察是否下载 xlsx 文件"
  fi
fi
agent-browser screenshot "$SCREENSHOTS_DIR/11_after_export.png" | tee -a "$LOG" || true

echo ">>> Capturing browser console/errors..." | tee -a "$LOG"
agent-browser console >"$CONSOLE_LOG" || true
agent-browser errors >"$ERRORS_LOG" || true

echo ">>> Finalizing trace/video..." | tee -a "$LOG"
agent-browser trace stop "$TRACE_ZIP" | tee -a "$LOG" || true
agent-browser record stop | tee -a "$LOG" || true
agent-browser close | tee -a "$LOG" || true

echo ">>> Generating HTML report..." | tee -a "$LOG"
set +e
python3 "$REPO_ROOT/tests/e2e/run_agent_browser_e2e_report.py" \
  --base-url "$BASE_URL" \
  --input-xlsx "$INPUT_XLSX" \
  --export-xlsx "$EXPORT_XLSX" \
  --ui-before "$UI_BEFORE" \
  --ui-after "$UI_AFTER" \
  --import-events "$IMPORT_EVENTS" \
  --steps "$STEPS_JSON" \
  --actions "$ACTION_RESULTS_JSON" \
  --console "$CONSOLE_LOG" \
  --errors "$ERRORS_LOG" \
  --trace "$TRACE_ZIP" \
  --video "$VIDEO_WEBM" \
  --screenshots "$SCREENSHOTS_DIR" \
  --out "$REPORT_HTML"
REPORT_RC=$?
set -e

cp -f "$REPORT_HTML" "$RESULTS_ROOT/report.html"
cp -f "$META" "$RESULTS_ROOT/meta.json"
for f in \
  missing_before.json \
  mismatches_before.json \
  missing_export.json \
  mismatches_export.json \
  completeness_cases.json \
  completeness_summary.json \
  missing_codes_by_sheet.json \
  derived_column_coverage.json \
  action_export_checks.json
do
  if [[ -f "$RUN_DIR/$f" ]]; then
    cp -f "$RUN_DIR/$f" "$RESULTS_ROOT/$f"
  fi
done
ln -snf "$(basename "$RUN_DIR")" "$RESULTS_ROOT/latest"

echo "" | tee -a "$LOG"
echo "OK: Report: $RESULTS_ROOT/report.html" | tee -a "$LOG"
echo "OK: Run dir: $RUN_DIR" | tee -a "$LOG"

exit "$REPORT_RC"

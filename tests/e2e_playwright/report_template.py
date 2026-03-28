"""自包含 HTML 报告模板（内联 CSS/JS），商务灰蓝色调

使用 string.Template ($variable) 避免 CSS/JS 花括号与 Python format 冲突。
"""
from string import Template

REPORT_HTML = Template("""\
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>NorthStar E2E Test Report</title>
<style>
  :root {
    --bg: #f8fafc; --card: #ffffff; --border: #e2e8f0;
    --text: #1e293b; --text-muted: #64748b; --text-light: #94a3b8;
    --pass: #10b981; --pass-bg: #ecfdf5; --fail: #ef4444; --fail-bg: #fef2f2;
    --skip: #f59e0b; --skip-bg: #fffbeb; --blue: #3b82f6; --blue-bg: #eff6ff;
    --radius: 8px; --shadow: 0 1px 3px rgba(0,0,0,.06), 0 1px 2px rgba(0,0,0,.04);
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         background: var(--bg); color: var(--text); line-height: 1.5; padding: 24px; }
  .container { max-width: 1200px; margin: 0 auto; }
  h1 { font-size: 1.5rem; font-weight: 600; margin-bottom: 4px; }
  .subtitle { color: var(--text-muted); font-size: 0.875rem; margin-bottom: 24px; }
  .stats { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; margin-bottom: 24px; }
  .stat-card { background: var(--card); border: 1px solid var(--border); border-radius: var(--radius);
               padding: 16px; box-shadow: var(--shadow); }
  .stat-card .label { font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;
                      letter-spacing: 0.05em; margin-bottom: 4px; }
  .stat-card .value { font-size: 1.5rem; font-weight: 700; }
  .stat-card.pass .value { color: var(--pass); }
  .stat-card.fail .value { color: var(--fail); }
  .stat-card.skip .value { color: var(--skip); }
  .progress-bar { height: 8px; background: var(--border); border-radius: 4px;
                  margin-bottom: 24px; overflow: hidden; }
  .progress-fill { height: 100%; border-radius: 4px; transition: width .3s; }
  .progress-fill.good { background: var(--pass); }
  .progress-fill.warn { background: var(--skip); }
  .progress-fill.bad { background: var(--fail); }
  .filters { display: flex; gap: 8px; margin-bottom: 16px; }
  .filter-btn { padding: 6px 16px; border: 1px solid var(--border); border-radius: 6px;
                background: var(--card); cursor: pointer; font-size: 0.8125rem;
                color: var(--text-muted); transition: all .15s; }
  .filter-btn:hover { border-color: var(--blue); color: var(--blue); }
  .filter-btn.active { background: var(--blue); color: white; border-color: var(--blue); }
  .test-group { background: var(--card); border: 1px solid var(--border);
                border-radius: var(--radius); margin-bottom: 12px; box-shadow: var(--shadow); }
  .group-header { padding: 12px 16px; cursor: pointer; display: flex;
                  align-items: center; justify-content: space-between; user-select: none; }
  .group-header:hover { background: #f1f5f9; }
  .group-title { font-weight: 600; font-size: 0.9375rem; }
  .group-badge { display: flex; gap: 6px; }
  .badge { padding: 2px 8px; border-radius: 10px; font-size: 0.75rem; font-weight: 500; }
  .badge.pass { background: var(--pass-bg); color: var(--pass); }
  .badge.fail { background: var(--fail-bg); color: var(--fail); }
  .badge.skip { background: var(--skip-bg); color: var(--skip); }
  .group-body { display: none; border-top: 1px solid var(--border); }
  .group-body.open { display: block; }
  .test-item { padding: 12px 16px; border-bottom: 1px solid var(--border);
               display: flex; align-items: flex-start; gap: 12px; }
  .test-item:last-child { border-bottom: none; }
  .status-dot { width: 8px; height: 8px; border-radius: 50%; margin-top: 6px; flex-shrink: 0; }
  .status-dot.pass { background: var(--pass); }
  .status-dot.fail { background: var(--fail); }
  .status-dot.skip { background: var(--skip); }
  .test-info { flex: 1; min-width: 0; }
  .test-name { font-weight: 500; font-size: 0.875rem; word-break: break-word; }
  .test-doc { color: var(--text-muted); font-size: 0.8125rem; margin-top: 2px; }
  .test-time { color: var(--text-light); font-size: 0.75rem; margin-top: 2px; }
  .test-error { background: var(--fail-bg); border: 1px solid #fecaca; border-radius: 6px;
                padding: 8px 12px; margin-top: 8px; font-family: monospace;
                font-size: 0.75rem; white-space: pre-wrap; overflow-x: auto;
                max-height: 200px; overflow-y: auto; color: #991b1b; }
  .screenshots { display: flex; gap: 8px; margin-top: 8px; flex-wrap: wrap; }
  .thumb { width: 120px; height: 72px; object-fit: cover; border: 1px solid var(--border);
           border-radius: 4px; cursor: pointer; transition: transform .15s; }
  .thumb:hover { transform: scale(1.05); }
  .lightbox { display: none; position: fixed; inset: 0; background: rgba(0,0,0,.8);
              z-index: 1000; align-items: center; justify-content: center; cursor: pointer; }
  .lightbox.show { display: flex; }
  .lightbox img { max-width: 90vw; max-height: 90vh; border-radius: 8px; }
  @media (max-width: 768px) {
    .stats { grid-template-columns: repeat(2, 1fr); }
    body { padding: 12px; }
  }
</style>
</head>
<body>
<div class="container">
  <h1>NorthStar E2E Test Report</h1>
  <div class="subtitle">$timestamp | Duration: $duration</div>
  <div class="stats">
    <div class="stat-card"><div class="label">Total</div><div class="value">$total</div></div>
    <div class="stat-card pass"><div class="label">Passed</div><div class="value">$passed</div></div>
    <div class="stat-card fail"><div class="label">Failed</div><div class="value">$failed</div></div>
    <div class="stat-card skip"><div class="label">Skipped</div><div class="value">$skipped</div></div>
    <div class="stat-card"><div class="label">Pass Rate</div><div class="value">$pass_rate%</div></div>
  </div>
  <div class="progress-bar">
    <div class="progress-fill $rate_class" style="width: $pass_rate%"></div>
  </div>
  <div class="filters">
    <button class="filter-btn active" onclick="filterTests('all')">All ($total)</button>
    <button class="filter-btn" onclick="filterTests('pass')">Passed ($passed)</button>
    <button class="filter-btn" onclick="filterTests('fail')">Failed ($failed)</button>
    <button class="filter-btn" onclick="filterTests('skip')">Skipped ($skipped)</button>
  </div>
  <div id="test-groups">$groups_html</div>
</div>
<div class="lightbox" id="lightbox" onclick="this.classList.remove('show')">
  <img id="lightbox-img" src="" alt="screenshot">
</div>
<script>
function toggleGroup(el) {
  el.nextElementSibling.classList.toggle('open');
}
function filterTests(status) {
  document.querySelectorAll('.filter-btn').forEach(function(b) { b.classList.remove('active'); });
  event.target.classList.add('active');
  document.querySelectorAll('.test-item').forEach(function(el) {
    el.style.display = (status === 'all' || el.dataset.status === status) ? '' : 'none';
  });
  document.querySelectorAll('.test-group').forEach(function(g) {
    var visible = g.querySelectorAll('.test-item:not([style*="display: none"])');
    g.style.display = visible.length ? '' : 'none';
  });
}
function showImg(src) {
  var lb = document.getElementById('lightbox');
  document.getElementById('lightbox-img').src = src;
  lb.classList.add('show');
  event.stopPropagation();
}
document.querySelectorAll('.test-group').forEach(function(g) {
  if (g.querySelector('.status-dot.fail')) { g.querySelector('.group-body').classList.add('open'); }
});
</script>
</body>
</html>
""")

GROUP_TEMPLATE = Template("""\
<div class="test-group">
  <div class="group-header" onclick="toggleGroup(this)">
    <span class="group-title">$group_name</span>
    <div class="group-badge">$badges</div>
  </div>
  <div class="group-body">$items_html</div>
</div>
""")

ITEM_TEMPLATE = Template("""\
<div class="test-item" data-status="$status">
  <div class="status-dot $status"></div>
  <div class="test-info">
    <div class="test-name">$name</div>
    $doc_html
    <div class="test-time">$duration</div>
    $error_html
    $screenshots_html
  </div>
</div>
""")

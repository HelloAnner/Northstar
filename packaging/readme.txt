Northstar 发布包

1) 运行
- macOS/Linux: ./northstar
- Windows: 双击 start.bat（窗口保持运行表示服务在运行，关闭窗口停止服务）
  - 或者直接运行 northstar.exe

2) 配置文件
- 程序启动时会自动读取同目录下的 config.toml
- 可通过命令行参数覆盖：
  -port 8080 -dev -dataDir data

3) 数据目录
- 默认会在同目录创建 data/（可通过 config.toml 或 -dataDir 覆盖）

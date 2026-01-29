@rem Northstar Windows 启动入口
@rem
@rem @author Anner
@rem Created on 2026/1/29
@echo off
setlocal
cd /d "%~dp0"

if not exist "northstar.exe" (
  echo ERROR: northstar.exe not found in current directory.
  echo Please keep start.bat, northstar.exe, config.toml, readme.txt in the same folder.
  pause
  exit /b 1
)

echo Starting Northstar... Close this window to stop the service.
northstar.exe
endlocal

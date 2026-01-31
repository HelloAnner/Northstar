@rem Northstar Windows 启动入口
@rem
@rem @author Anner
@rem Created on 2026/1/29
@rem Updated on 2026/1/31
@echo off
setlocal
cd /d "%~dp0"

chcp 65001 >nul

set "LOG_FILE=%~dp0northstar.log"

if not exist "northstar.exe" (
  echo ERROR: northstar.exe not found in current directory.
  echo Please keep start.bat, northstar.exe, config.toml, readme.txt in the same folder.
  pause
  exit /b 1
)

echo Logs: %LOG_FILE%
>> "%LOG_FILE%" echo.
>> "%LOG_FILE%" echo ===== START %DATE% %TIME% =====

echo Starting Northstar... Close this window to stop the service.
northstar.exe %* 1>>"%LOG_FILE%" 2>&1
set "NS_EXIT_CODE=%ERRORLEVEL%"
if not "%NS_EXIT_CODE%"=="0" (
  echo.
  echo ERROR: Northstar exited with code %NS_EXIT_CODE%.
  echo --- Log tail (last 200 lines) ---
  where powershell.exe >nul 2>nul
  if "%ERRORLEVEL%"=="0" (
    powershell.exe -NoProfile -Command "Get-Content -Path '%LOG_FILE%' -Tail 200"
  ) else (
    type "%LOG_FILE%"
  )
  echo --- Error lines (best effort) ---
  findstr /i /r "error fatal panic exception" "%LOG_FILE%" >nul 2>nul && findstr /i /r "error fatal panic exception" "%LOG_FILE%"
  echo Tip: This is often caused by "port already in use". Try: northstar.exe -port 18080
  echo.
  pause
)
endlocal & exit /b %NS_EXIT_CODE%

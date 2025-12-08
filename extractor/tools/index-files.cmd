@echo off
setlocal enabledelayedexpansion
set "LIST=%~1"
set "EXE=%~dp0..\extractor\dart-extractor.exe"
echo [index-files] CWD: "%CD%"
echo [index-files] Using extractor: "%EXE%"

if not exist "%EXE%" (
  echo ERROR: extractor exe not found: "%EXE%"
  exit /b 1
)
if not exist "%LIST%" (
  echo ERROR: file list not found: "%LIST%"
  exit /b 1
)

for /f "usebackq delims=" %%F in ("%LIST%") do (
  REM Skip empty lines
  if not "%%F"=="" (
    "%EXE%" --index "%%~fF"
    if errorlevel 1 exit /b !errorlevel!
  )
)

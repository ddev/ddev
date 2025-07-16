@echo "Running Windows installer tests..."

TASKKILL /T /F /IM mutagen.exe 2>NUL

make testwininstaller

if %ERRORLEVEL% EQU 0 (
   @echo Successful installer test
) else (
   @echo Installer test failed with error code %errorlevel%
   exit /b %errorlevel%
)
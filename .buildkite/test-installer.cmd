@echo "Running Windows installer tests..."

TASKKILL /T /F /IM mutagen.exe 2>NUL

"C:\Program Files\git\bin\bash" .buildkite/test-installer.sh

if %ERRORLEVEL% EQU 0 (
   @echo Successful installer test
) else (
   @echo Installer test failed with error code %errorlevel%
   exit /b %errorlevel%
)

@echo "Running nightly performance benchmark via bash and perf.sh"

TASKKILL /T /F /IM mutagen.exe

"C:\Program Files\git\bin\bash" .buildkite/perf.sh

if %ERRORLEVEL% EQU 0 (
   @echo Successful benchmark run
) else (
   @echo Failure Reason Given is %errorlevel%
   exit /b %errorlevel%
)

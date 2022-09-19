@echo "Building using bash and build.sh"

TASKKILL /T /F /IM mutagen.exe

"C:\Program Files\git\bin\bash" .buildkite/test.sh

if %ERRORLEVEL% EQU 0 (
   @echo Successful build
) else (
   @echo Failure Reason Given is %errorlevel%
   exit /b %errorlevel%
)

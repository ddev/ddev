@echo "Running Windows installer test using bash"

"C:\Program Files\git\bin\bash" .buildkite/installer-test.sh

if %ERRORLEVEL% EQU 0 (
   @echo Successful installer test
) else (
   @echo Failure Reason Given is %errorlevel%
   exit /b %errorlevel%
)

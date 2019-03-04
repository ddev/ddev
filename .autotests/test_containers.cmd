@echo "Building using bash and build.sh"
"C:\Program Files\git\bin\bash" .autotests/test_containers.sh

if %ERRORLEVEL% EQU 0 (
   @echo Successful build
) else (
   @echo Failure Reason Given is %errorlevel%
   exit /b %errorlevel%
)

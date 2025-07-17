@echo "Building using bash and build.sh"

@echo "DEBUG: Concurrency group debug value: %CONCURRENCY_GROUP_DEBUG%"
@echo "DEBUG: Agent ID: %BUILDKITE_AGENT_ID%"
@echo "DEBUG: Agent Name: %BUILDKITE_AGENT_NAME%"
@echo "DEBUG: Computer Name: %COMPUTERNAME%"
@echo "DEBUG: Expected concurrency group: windows-tests-%BUILDKITE_AGENT_ID%-%BUILDKITE_AGENT_NAME%-%COMPUTERNAME%"

TASKKILL /T /F /IM mutagen.exe

"C:\Program Files\git\bin\bash" .buildkite/test.sh

if %ERRORLEVEL% EQU 0 (
   @echo Successful build
) else (
   @echo Failure Reason Given is %errorlevel%
   exit /b %errorlevel%
)

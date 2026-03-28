# Review notes

TODO: This file should have been removed

* Does it compare the actual cert found installed in WSL2 to the one found in Windows?
* The "X" in report showing "mkcert CA NOT found in Windows certificate store" is not very visible, should be red probably.
* The suggestion "run mkcert as admin in PowerShell" is incorrect, should be as normal user.
* It should check whether mkcert.exe is found on Windows as well as looking in WSL2.
* Still need to test this on Traditional Windows.
* I think when mkcert is not working it says it can't connect to the site from windows side, but the reality is it's a failure of cert. It should have better response than "connectitivity error"
* Test with WSLg
* On Linux, firefox usually works if configured with mkcert. But we have to check for libnss or whatever to know. So "Firefox detected" where it says do "mkcert -install" seems to contradict the statement above saying that libnss is available etc. Instead, this should probably be a warning that some versions of firefox don't respect CA from mkcert so you have to manually import the CA.
* I happened to test in traditional Windows, and WSL2 ddev was still running. Got "TLS verified: localhost:443 with SNI d11.ddev.site", but in reality 443 is the WSL2 ddev-router. We should use the actual DDEV port that is in use for the test.
* The traditional Windows path does not seem to check for Firefox, and it needs to, with the same result and such as when using firefox on Windows with wsl2



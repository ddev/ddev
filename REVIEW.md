TODO: Delete this file, only for reviewing on train

* I tried this on WSL2 when using `sudo nc -l -p 80` and it properly detected the port problem but it couldn't figure out that it was nc that was the problem. It would be better if it found a way to do that. Might need to elevate for port 80, but tried on 8142 and got same response. I don't think it's trying hard enough to figure out the process. (This was mirrored mode)
* It should insist on a poweroff if anything is running, because otherwise it incorrectly reports all the ports legitimately in use as a problem.
* The output can be much more terse. How about one-line results, like `Port 8142 (XHGui HTTPS): ... <result>`

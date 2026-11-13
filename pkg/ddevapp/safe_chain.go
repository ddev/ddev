package ddevapp

import "github.com/ddev/ddev/pkg/globalconfig"

// safeChainInstallDockerfile installs Aikido Safe Chain for safe_chain_enable.
// The CI shims wrap npm, yarn, pnpm, pip, and the other package managers in
// every shell, including the non-interactive ones `ddev exec` starts; ENV PATH
// covers processes started without bash. safe-chain caches its malware lists in
// the install dir, so the project user (group 0) must be able to write there.
// The build fails when the install fails, instead of running unprotected.
const safeChainInstallDockerfile = `
### DDEV-injected Aikido Safe Chain, see https://github.com/AikidoSec/safe-chain
ENV PATH=/usr/local/safe-chain/shims:/usr/local/safe-chain/bin:$PATH
RUN <<EOF
set -eu -o pipefail
# Unset BASH_ENV, or setup-ci appends its PATH line to /etc/bash.nointeractive.bashrc
curl -fsSL --retry 3 https://github.com/AikidoSec/safe-chain/releases/latest/download/install-safe-chain.sh | env -u BASH_ENV sh -s -- --ci --install-dir /usr/local/safe-chain
rm -rf /root/.safe-chain
chgrp -R 0 /usr/local/safe-chain && chmod -R g+rwX,o-w /usr/local/safe-chain
cat >/etc/bashrc/05-safe-chain.bashrc <<'BASHRC'
# Keep the safe-chain shims ahead of the dirs 00-path-setup.bashrc prepends,
# such as vendor/bin, and restore them where /etc/profile reset PATH.
_sc_path= _sc_ifs=$IFS IFS=:
for _sc_p in $PATH; do
  case "$_sc_p" in
    /usr/local/safe-chain/shims|/usr/local/safe-chain/bin) ;;
    *) _sc_path="${_sc_path:+$_sc_path:}$_sc_p" ;;
  esac
done
IFS=$_sc_ifs
PATH="/usr/local/safe-chain/shims:/usr/local/safe-chain/bin${_sc_path:+:$_sc_path}"
unset _sc_p _sc_path _sc_ifs
export PATH
BASHRC
safe-chain --version
test -x /usr/local/safe-chain/shims/npm
EOF
`

// IsSafeChainEnabled reports whether the project or the global config sets
// safe_chain_enable. A project can't opt out of a global setting.
func (app *DdevApp) IsSafeChainEnabled() bool {
	return app.SafeChainEnable || globalconfig.DdevGlobalConfig.SafeChainEnable
}

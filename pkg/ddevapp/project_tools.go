package ddevapp

import "github.com/ddev/ddev/pkg/nodeps"

func projectToolsInstallDockerfile(projectType string) string {
	switch projectType {
	case nodeps.AppTypeWordPress, nodeps.AppTypeWPBedrock:
		return wpCLIInstallDockerfile
	case nodeps.AppTypeMagento:
		return magerunInstallDockerfile
	case nodeps.AppTypeMagento2:
		return magerun2InstallDockerfile
	case nodeps.AppTypeDrupal6, nodeps.AppTypeDrupal7:
		return drush8InstallDockerfile
	case nodeps.AppTypeBackdrop:
		return drush8InstallDockerfile + backdropDrushInstallDockerfile
	case nodeps.AppTypeSymfony:
		return symfonyCLIInstallDockerfile
	case nodeps.AppTypeShopware6:
		return shopwareCLIInstallDockerfile
	default:
		return ""
	}
}

const wpCLIInstallDockerfile = `
### DDEV-injected WP-CLI install for wordpress/wp-bedrock projects
RUN curl --fail -sSL -o /usr/local/bin/wp-cli https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && chmod +x /usr/local/bin/wp-cli && ln -sf /usr/local/bin/wp-cli /usr/local/bin/wp
`

const magerunInstallDockerfile = `
### DDEV-injected magerun install for magento projects
RUN <<EOF
    set -eu -o pipefail
    curl --fail -sSL https://files.magerun.net/n98-magerun-latest.phar -o /usr/local/bin/magerun
    chmod 755 /usr/local/bin/magerun
    curl --fail -sSL https://raw.githubusercontent.com/netz98/n98-magerun/master/res/autocompletion/bash/n98-magerun.phar.bash -o /etc/bash_completion.d/n98-magerun.phar
EOF
`

const magerun2InstallDockerfile = `
### DDEV-injected magerun2 install for magento2 projects
RUN <<EOF
    set -eu -o pipefail
    curl --fail -sSL https://files.magerun.net/n98-magerun2-latest.phar -o /usr/local/bin/magerun2
    chmod 755 /usr/local/bin/magerun2
    curl --fail -sSL https://raw.githubusercontent.com/netz98/n98-magerun2/master/res/autocompletion/bash/n98-magerun2.phar.bash -o /etc/bash_completion.d/n98-magerun2.phar
EOF
`

// Use the bundled PHP for Composer so dependency installation is independent
// of the project PHP version, including versions that Composer cannot run on.
const drush8InstallDockerfile = `
### DDEV-injected Drush 8 install for drupal6/drupal7/backdrop projects
RUN <<ENDDRUSH
    set -eu -o pipefail
    mkdir -p /usr/local/src/drush
    curl -sSfL -o /tmp/drush.tgz https://github.com/drush-ops/drush/archive/refs/tags/8.5.0.tar.gz
    tar -C /usr/local/src/drush --strip-components=1 -zxf /tmp/drush.tgz
    pushd /usr/local/src/drush >/dev/null
    php8.4 /usr/local/bin/composer install --no-interaction
    ln -sf /usr/local/src/drush/drush /usr/local/bin/drush8
    popd >/dev/null
    rm -f /tmp/drush.tgz
ENDDRUSH
`

const backdropDrushInstallDockerfile = `
### DDEV-injected Backdrop Drush extension install for backdrop projects
RUN <<EOF
    set -eu -o pipefail
    tag=$(curl --fail -sSL https://api.github.com/repos/backdrop-contrib/drush/releases/latest | jq -er .tag_name)
    curl --fail -sSL "https://github.com/backdrop-contrib/drush/releases/download/${tag}/backdrop-drush-extension.zip" -o /tmp/backdrop-drush-extension.zip
    unzip -o /tmp/backdrop-drush-extension.zip -d /var/tmp/backdrop_drush_commands
    chmod -R ugo+w /var/tmp/backdrop_drush_commands
    rm -f /tmp/backdrop-drush-extension.zip
EOF
`

const symfonyCLIInstallDockerfile = `
### DDEV-injected Symfony CLI install for symfony projects
RUN <<EOF
    set -eu -o pipefail
    curl -1sLf 'https://dl.cloudsmith.io/public/symfony/stable/setup.deb.sh' | bash
    apt modernize-sources --assume-yes
    rm -f /etc/apt/sources.list.d/*.list.bak
    apt-get update
    apt-get install -y --no-install-recommends symfony-cli
    apt-get clean
    rm -rf /var/lib/apt/lists/*
EOF
`

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

var (
	FirefoxProfiles = []string{os.Getenv("HOME") + "/.mozilla/firefox/*",
		os.Getenv("HOME") + "/.mozilla/firefox-trunk/*",
		os.Getenv("HOME") + "/snap/firefox/common/.mozilla/firefox/*"}
	NSSBrowsers = "Firefox and/or Chrome/Chromium"

	CertutilInstallHelp string
)

func init() {
	switch {
	case binaryExists("apt"):
		CertutilInstallHelp = "apt install libnss3-tools"
	case binaryExists("yum"):
		CertutilInstallHelp = "yum install nss-tools"
	case binaryExists("zypper"):
		CertutilInstallHelp = "zypper install mozilla-nss-tools"
	case binaryExists("emerge"):
		CertutilInstallHelp = "echo 'dev-libs/nss utils' >> /etc/portage/package.use/nss && emerge -av dev-libs/nss"
	}
}

func getSystemTrust() (string, []string) {
	systemTrustFilenamePattern := ""
	var systemTrustCommand []string
	if pathExists("/etc/pki/ca-trust/source/anchors/") {
		systemTrustFilenamePattern = "/etc/pki/ca-trust/source/anchors/%s.pem"
		systemTrustCommand = []string{"update-ca-trust", "extract"}
	} else if pathExists("/usr/local/share/ca-certificates/") {
		systemTrustFilenamePattern = "/usr/local/share/ca-certificates/%s.crt"
		systemTrustCommand = []string{"update-ca-certificates"}
	} else if pathExists("/etc/ca-certificates/trust-source/anchors/") {
		systemTrustFilenamePattern = "/etc/ca-certificates/trust-source/anchors/%s.crt"
		systemTrustCommand = []string{"trust", "extract-compat"}
	} else if pathExists("/usr/share/pki/trust/anchors") {
		systemTrustFilenamePattern = "/usr/share/pki/trust/anchors/%s.pem"
		systemTrustCommand = []string{"update-ca-certificates"}
	}
	return systemTrustFilenamePattern, systemTrustCommand
}

func (ca *CA) systemTrustFilename(systemTrustFilenamePattern string) string {
	return fmt.Sprintf(systemTrustFilenamePattern, strings.Replace(ca.caUniqueName(), " ", "_", -1))
}

func (ca *CA) installPlatform() error {
	systemTrustFilenamePattern, systemTrustCommand := getSystemTrust()
	if systemTrustCommand == nil {
		terminal.Printf("Installing to the system store is not yet supported on this Linux but %s will still work.", NSSBrowsers)
		terminal.Printf("You can also manually install the root certificate at %q.", ca.rootpath)
		return nil
	}

	cert, err := ioutil.ReadFile(ca.rootpath)
	if err != nil {
		return errors.Wrap(err, "failed to read root certificate")
	}

	cmd := commandWithSudo("tee", ca.systemTrustFilename(systemTrustFilenamePattern))
	cmd.Stdin = bytes.NewReader(cert)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, "tee", out)
	}

	cmd = commandWithSudo(systemTrustCommand...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, strings.Join(systemTrustCommand, " "), out)
	}

	return nil
}

func (ca *CA) uninstallPlatform() error {
	systemTrustFilenamePattern, systemTrustCommand := getSystemTrust()
	if systemTrustCommand == nil {
		return nil
	}

	cmd := commandWithSudo("rm", "-f", ca.systemTrustFilename(systemTrustFilenamePattern))
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, "rm", out)
	}

	cmd = commandWithSudo(systemTrustCommand...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, strings.Join(systemTrustCommand, " "), out)
	}

	return nil
}

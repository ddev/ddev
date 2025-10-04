// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"bytes"
	"encoding/asn1"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"howett.net/plist"
)

var (
	FirefoxProfiles     = []string{os.Getenv("HOME") + "/Library/Application Support/Firefox/Profiles/*"}
	CertutilInstallHelp = "brew install nss"
	NSSBrowsers         = "Firefox"
)

// https://github.com/golang/go/issues/24652#issuecomment-399826583
var trustSettings []interface{}
var _, _ = plist.Unmarshal(trustSettingsData, &trustSettings)
var trustSettingsData = []byte(`
<array>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAED
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>sslServer</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAEC
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>basicX509</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
</array>
`)

func (ca *CA) installPlatform() error {
	cmd := commandWithSudo("security", "add-trusted-cert", "-d", "-k", "/Library/Keychains/System.keychain", ca.rootpath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, "security add-trusted-cert", out)
	}

	// Make trustSettings explicit, as older Go does not know the defaults.
	// https://github.com/golang/go/issues/24652

	plistFile, err := ioutil.TempFile("", "trust-settings")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}

	defer os.Remove(plistFile.Name())

	cmd = commandWithSudo("security", "trust-settings-export", "-d", plistFile.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, "security trust-settings-export", out)
	}

	plistData, err := ioutil.ReadFile(plistFile.Name())
	if err != nil {
		return errors.Wrap(err, "failed to read trust settings")
	}
	var plistRoot map[string]interface{}
	_, err = plist.Unmarshal(plistData, &plistRoot)
	if err != nil {
		return errors.Wrap(err, "failed to parse trust settings")
	}

	rootSubjectASN1, _ := asn1.Marshal(ca.cert.Subject.ToRDNSequence())

	if plistRoot["trustVersion"].(uint64) != 1 {
		return errors.Errorf("unsupported trust settings version: %s", plistRoot["trustVersion"])
	}
	trustList := plistRoot["trustList"].(map[string]interface{})
	for key := range trustList {
		entry := trustList[key].(map[string]interface{})
		if _, ok := entry["issuerName"]; !ok {
			continue
		}
		issuerName := entry["issuerName"].([]byte)
		if !bytes.Equal(rootSubjectASN1, issuerName) {
			continue
		}
		entry["trustSettings"] = trustSettings
		break
	}

	plistData, err = plist.MarshalIndent(plistRoot, plist.XMLFormat, "\t")
	if err != nil {
		return errors.Wrap(err, "failed to serialize trust settings")
	}
	err = ioutil.WriteFile(plistFile.Name(), plistData, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to write trust settings")
	}

	cmd = commandWithSudo("security", "trust-settings-import", "-d", plistFile.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, "security trust-settings-import", out)
	}
	return nil
}

func (ca *CA) uninstallPlatform() error {
	cmd := commandWithSudo("security", "remove-trusted-cert", "-d", ca.rootpath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapCmdErr(err, "security remove-trusted-cert", out)
	}
	return nil
}

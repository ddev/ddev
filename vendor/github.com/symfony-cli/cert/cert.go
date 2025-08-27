// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
	"golang.org/x/net/idna"
	"software.sslmate.com/src/go-pkcs12"
)

type CA struct {
	cert           *x509.Certificate
	Key            crypto.PrivateKey
	rootpath       string
	keypath        string
	trustedpath    string
	generatedCerts map[string]bool
}

func NewCA(dir string) (*CA, error) {
	return &CA{
		rootpath:       filepath.Join(dir, "rootCA.pem"),
		keypath:        filepath.Join(dir, "rootCA-key.pem"),
		trustedpath:    filepath.Join(dir, "trusted"),
		generatedCerts: make(map[string]bool),
	}, nil
}

var userAndHostnameCache string
var userAndHostnameCacheOnce sync.Once
var sudoWarningOnce sync.Once

func userAndHostname() string {
	userAndHostnameCacheOnce.Do(func() {
		u, err := user.Current()
		if err == nil {
			userAndHostnameCache = u.Username + "@"
		}
		if h, err := os.Hostname(); err == nil {
			userAndHostnameCache += h
		}
		if err == nil && u.Name != "" && u.Name != u.Username {
			userAndHostnameCache += " (" + u.Name + ")"
		}
	})
	return userAndHostnameCache
}

func (ca *CA) HasCA() bool {
	if _, err := os.Stat(ca.rootpath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (ca *CA) AsTLS() *tls.Certificate {
	return &tls.Certificate{
		Certificate: [][]byte{ca.cert.Raw},
		PrivateKey:  ca.Key,
		Leaf:        ca.cert,
	}
}

func (ca *CA) LoadCA() error {
	certPEMBlock, err := ioutil.ReadFile(ca.rootpath)
	if err != nil {
		return errors.Wrap(err, "failed to read the CA certificate")
	}
	certDERBlock, _ := pem.Decode(certPEMBlock)
	if certDERBlock == nil || certDERBlock.Type != "CERTIFICATE" {
		return errors.New("failed to read the CA certificate: unexpected content")
	}
	ca.cert, err = x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse the CA certificate")
	}
	keyPEMBlock, err := ioutil.ReadFile(ca.keypath)
	if err != nil {
		return errors.Wrap(err, "failed to read the CA key")
	}
	keyDERBlock, _ := pem.Decode(keyPEMBlock)
	if keyDERBlock == nil || keyDERBlock.Type != "PRIVATE KEY" {
		return errors.New("failed to read the CA key: unexpected content")
	}
	ca.Key, err = x509.ParsePKCS8PrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse the CA key")
	}
	return nil
}

func (ca *CA) IsTrusted() bool {
	_, err := ioutil.ReadFile(ca.trustedpath)
	return err == nil
}

func (ca *CA) IsExpired() bool {
	return ca.cert.NotAfter.Before(time.Now())
}

func (ca *CA) MustBeRegenerated() bool {
	if !isMacOS() {
		return false
	}
	if ca.cert.NotBefore.Before(time.Date(2019, 7, 1, 0, 0, 0, 0, time.UTC)) {
		return false
	}
	return int(ca.cert.NotAfter.Sub(ca.cert.NotBefore).Hours()/24) > 825
}

var isMacOS = func() bool {
	return runtime.GOOS == "darwin"
}

func (ca *CA) Install(force bool) error {
	if force || !ca.IsTrusted() {
		terminal.Println("You might have to enter your root password to install the local Certificate Authority certificate")
		if err := ca.installPlatform(); err != nil {
			return err
		}
		f, _ := os.OpenFile(ca.trustedpath, os.O_RDONLY|os.O_CREATE, 0644)
		f.Close()
		terminal.Println("The local CA is now installed in the system trust store!")
	}
	if hasNSS() && (force || !ca.checkNSS()) {
		hasCertutil := certutilPath() != ""
		if hasCertutil {
			if err := ca.installNSS(); err != nil {
				return err
			}
			terminal.Printf("The local CA is now installed in the %s trust store (requires browser restart)!\n", NSSBrowsers)
		} else if CertutilInstallHelp == "" {
			terminal.Printf("<comment>Note</>: %s support is not available on your platform.\n", NSSBrowsers)
		} else if !hasCertutil {
			terminal.Printf("<warning>WARNING</> \"certutil\" is not available, so the CA can't be automatically installed in %s!\n", NSSBrowsers)
			terminal.Printf("Install \"certutil\" with \"%s\" and re-run the command\n", CertutilInstallHelp)
		}
	}
	return nil
}

func (ca *CA) Uninstall() error {
	hasCertutil := certutilPath() != ""
	if hasNSS() {
		if hasCertutil {
			ca.uninstallNSS()
		} else if CertutilInstallHelp != "" {
			terminal.Printf("<warning>WARNING</> \"certutil\" is not available, so the CA can't be automatically uninstalled from %s (if it was ever installed)!\n", NSSBrowsers)
			terminal.Printf("You can install \"certutil\" with \"%s\" and re-run the command\n", CertutilInstallHelp)
		}
	}
	err := ca.uninstallPlatform()
	if err != nil {
		terminal.Println("The local CA is now uninstalled from the system trust store(s)!")
	}
	if hasCertutil {
		terminal.Printf("The local CA is now uninstalled from the %s trust store(s)!\n", NSSBrowsers)
	}
	return nil
}

func Cert(filename string) (tls.Certificate, error) {
	p12, err := ioutil.ReadFile(filename)
	if err != nil {
		return tls.Certificate{}, errors.WithStack(err)
	}
	priv, domainCert, caCerts, err := pkcs12.DecodeChain(p12, "")
	if err == pkcs12.ErrIncorrectPassword {
		priv, domainCert, caCerts, err := pkcs12.DecodeChain(p12, "symfony")
		if err != nil {
			return tls.Certificate{}, errors.WithStack(err)
		}

		terminal.Printfln("<warning>Removing passphrase on certificate %s</>", filename)

		// In case the previous certificate has a passphrase, we re-encode it
		// on the fly without passphrase
		pfxData, err := pkcs12.Modern.Encode(priv, domainCert, caCerts, "")
		if err != nil {
			return tls.Certificate{}, errors.WithStack(err)
		}
		defer ioutil.WriteFile(filename, pfxData, 0644)

		certs := [][]byte{domainCert.Raw}
		for _, c := range caCerts {
			certs = append(certs, c.Raw)
		}

		return tls.Certificate{
			Certificate: certs,
			PrivateKey:  priv,
		}, nil
	}
	if err != nil {
		return tls.Certificate{}, errors.WithStack(err)
	}
	certs := [][]byte{domainCert.Raw}
	for _, c := range caCerts {
		certs = append(certs, c.Raw)
	}

	return tls.Certificate{
		Certificate: certs,
		PrivateKey:  priv,
	}, nil
}

func (ca *CA) CreateCert(hosts []string) (tls.Certificate, error) {
	if ca.Key == nil {
		return tls.Certificate{}, errors.New("failed to create new certificates as the CA key is missing")
	}

	hostnameRegexp := regexp.MustCompile(`(?i)^(\*\.)?[0-9a-z_-]([0-9a-z._-]*[0-9a-z_-])?$`)
	for i, name := range hosts {
		if ip := net.ParseIP(name); ip != nil {
			continue
		}
		punycode, err := idna.ToASCII(name)
		if err != nil {
			return tls.Certificate{}, errors.Wrapf(err, "%q is not a valid hostname or IP", name)
		}
		hosts[i] = punycode
		if !hostnameRegexp.MatchString(punycode) {
			return tls.Certificate{}, errors.Errorf("%q is not a valid hostname or IP", name)
		}
	}

	/*
		FIXME: this checks should prints some warning but they are not errors
		secondLvlWildcardRegexp := regexp.MustCompile(`(?i)^\*\.[0-9a-z_-]+$`)
		for _, h := range hosts {
			if secondLvlWildcardRegexp.MatchString(h) {
				return tls.Certificate{}, errors.Errorf("many browsers don't support second-level wildcards like %q", h)
			}
		}

		for _, h := range hosts {
			if strings.HasPrefix(h, "*.") {
				return tls.Certificate{}, errors.Errorf("X.509 wildcards only go one level deep, so this won't match a.b.%s", h[2:])
			}
		}
	*/

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to generate certificate key")
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to generate serial number")
	}

	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Symfony dev cert"},
			OrganizationalUnit: []string{userAndHostname()},
		},

		// iOS13 and macOS 10.15 require a validity period of 825 days or fewer
		// see https://support.apple.com/en-us/HT210176
		NotAfter:  time.Now().AddDate(0, 0, 825),
		NotBefore: time.Now(),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, h)
		}
	}

	// IIS (the main target of PKCS #12 files), only shows the deprecated
	// Common Name in the UI. See issue #115.
	tpl.Subject.CommonName = hosts[0]

	pub := priv.PublicKey
	cert, err := x509.CreateCertificate(rand.Reader, tpl, ca.cert, &pub, ca.Key)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to generate certificate")
	}

	return tls.Certificate{
		Certificate: [][]byte{cert, ca.cert.Raw},
		PrivateKey:  priv,
	}, nil
}

func (ca *CA) MakeCert(filename string, hosts []string) error {
	c, err := ca.CreateCert(hosts)
	if err != nil {
		return err
	}
	cert := c.Certificate[0]
	priv := c.PrivateKey

	domainCert, _ := x509.ParseCertificate(cert)
	pfxData, err := pkcs12.Modern.Encode(priv, domainCert, []*x509.Certificate{ca.cert}, "")
	if err != nil {
		return errors.Wrap(err, "failed to generate PKCS#12")
	}
	if err := ioutil.WriteFile(filename, pfxData, 0644); err != nil {
		return errors.Wrap(err, "failed to save PKCS#12")
	}
	return nil
}

func (ca *CA) CreateCA() error {
	dir := filepath.Dir(ca.keypath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrap(err, "failed to create the CA directory")
		}
	}
	priv, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return errors.Wrap(err, "failed to generate the CA key")
	}
	pub := priv.PublicKey

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return errors.Wrap(err, "failed to generate serial number")
	}

	spkiASN1, err := x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		return errors.Wrap(err, "failed to encode public key")
	}

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	if err != nil {
		return errors.Wrap(err, "failed to decode public key")
	}

	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)

	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Symfony dev CA"},
			OrganizationalUnit: []string{userAndHostname()},
			CommonName:         "Symfony " + userAndHostname(),
		},
		SubjectKeyId: skid[:],

		// iOS13 and macOS 10.15 require a validity period of 825 days or fewer
		// see https://support.apple.com/en-us/HT210176
		NotAfter:  time.Now().AddDate(0, 0, 825),
		NotBefore: time.Now(),

		KeyUsage: x509.KeyUsageCertSign,

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &pub, priv)
	if err != nil {
		return errors.Wrap(err, "failed to generate CA certificate")
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return errors.Wrap(err, "failed to encode CA key")
	}

	return ca.save(privDER, cert)
}

func (ca *CA) save(privDER, cert []byte) error {
	if err := ioutil.WriteFile(ca.keypath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}), 0400); err != nil {
		return errors.Wrap(err, "failed to save CA key")
	}
	if err := ioutil.WriteFile(ca.rootpath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0644); err != nil {
		return errors.Wrap(err, "failed to save CA key")
	}
	return nil
}

func (ca *CA) caUniqueName() string {
	return "Symfony dev CA " + ca.cert.SerialNumber.String()
}

func wrapCmdErr(err error, cmd string, out []byte) error {
	return errors.Wrapf(err, `failed to execute "%s": %s`, cmd, out)
}

func commandWithSudo(cmd ...string) *exec.Cmd {
	if u, err := user.Current(); err == nil && u.Uid == "0" {
		return exec.Command(cmd[0], cmd[1:]...)
	}
	if !binaryExists("sudo") {
		sudoWarningOnce.Do(func() {
			log.Println(`Warning: "sudo" is not available, and the command is not running as root; the operation might fail.`)
		})
		return exec.Command(cmd[0], cmd[1:]...)
	}
	return exec.Command("sudo", append([]string{"--prompt=Sudo password:", "--"}, cmd...)...)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

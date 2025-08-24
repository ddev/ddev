package cert

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

var nssDBs = []string{
	filepath.Join(os.Getenv("HOME"), ".pki/nssdb"),
	filepath.Join(os.Getenv("HOME"), "snap/chromium/current/.pki/nssdb"), // Snapcraft
	"/etc/pki/nssdb", // CentOS 7
}

var firefoxPaths = []string{
	"/usr/bin/firefox",
	"/usr/bin/firefox-nightly",
	"/usr/bin/firefox-developer-edition",
	"/snap/firefox",
	"/Applications/Firefox.app",
	"/Applications/FirefoxDeveloperEdition.app",
	"/Applications/Firefox Developer Edition.app",
	"/Applications/Firefox Nightly.app",
	"C:\\Program Files\\Mozilla Firefox",
}

func hasNSS() bool {
	allPaths := append(append([]string{}, nssDBs...), firefoxPaths...)
	for _, path := range allPaths {
		if pathExists(path) {
			return true
		}
	}
	return false
}

func certutilPath() string {
	// certutil does not exist on Windows
	if runtime.GOOS == "windows" {
		return ""
	}
	if path, err := exec.LookPath("certutil"); err == nil {
		return path
	}
	if runtime.GOOS == "darwin" {
		// Check the default Homebrew path, to save executing Ruby
		if path, err := exec.LookPath("/usr/local/opt/nss/bin/certutil"); err == nil {
			return path
		}
		if out, err := exec.Command("brew", "--prefix", "nss").Output(); err == nil {
			path := filepath.Join(strings.TrimSpace(string(out)), "bin", "certutil")
			if _, err = os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

func (ca *CA) checkNSS() bool {
	certutilPath := certutilPath()
	if certutilPath == "" {
		return false
	}
	success := true
	found, err := ca.forEachNSSProfile(func(profile string) error {
		err := exec.Command(certutilPath, "-V", "-d", profile, "-u", "L", "-n", ca.caUniqueName()).Run()
		if err != nil {
			success = false
		}
		return nil
	})
	if err != nil || found == 0 {
		success = false
	}
	return success
}

func (ca *CA) installNSS() error {
	certutilPath := certutilPath()
	found, err := ca.forEachNSSProfile(func(profile string) error {
		cmd := exec.Command(certutilPath, "-A", "-d", profile, "-t", "C,,", "-n", ca.caUniqueName(), "-i", ca.rootpath)
		if out, err := execCertutil(cmd); err != nil {
			return wrapCmdErr(err, "certutil -A -d"+profile, out)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if found == 0 {
		return errors.Errorf("no %s security databases found", NSSBrowsers)
	}
	if !ca.checkNSS() {
		return errors.Errorf("Installing in %s failed (note that if you never started %s, you need to do that at least once)", NSSBrowsers, NSSBrowsers)
	}
	return nil
}

func (ca *CA) uninstallNSS() {
	certutilPath := certutilPath()
	ca.forEachNSSProfile(func(profile string) error {
		err := exec.Command(certutilPath, "-V", "-d", profile, "-u", "L", "-n", ca.caUniqueName()).Run()
		if err != nil {
			return nil
		}
		cmd := exec.Command(certutilPath, "-D", "-d", profile, "-n", ca.caUniqueName())
		if out, err := execCertutil(cmd); err != nil {
			return wrapCmdErr(err, "certutil -D -d"+profile, out)
		}
		return nil
	})
}

func (ca *CA) forEachNSSProfile(f func(profile string) error) (int, error) {
	found := 0
	var profiles []string
	profiles = append(profiles, nssDBs...)
	for _, ff := range FirefoxProfiles {
		pp, _ := filepath.Glob(ff)
		profiles = append(profiles, pp...)
	}
	for _, profile := range profiles {
		if stat, err := os.Stat(profile); err != nil || !stat.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(profile, "cert9.db")); err == nil {
			if err := f("sql:" + profile); err != nil {
				return 0, err
			}
			found++
			continue
		}
		if _, err := os.Stat(filepath.Join(profile, "cert8.db")); err == nil {
			if err := f("dbm:" + profile); err != nil {
				return 0, err
			}
			found++
		}
	}
	return found, nil
}

// execCertutil will execute a "certutil" command and if needed re-execute
// the command with commandWithSudo to work around file permissions.
func execCertutil(cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.CombinedOutput()
	if err != nil && bytes.Contains(out, []byte("SEC_ERROR_READ_ONLY")) && runtime.GOOS != "windows" {
		origArgs := cmd.Args[1:]
		cmd = commandWithSudo(cmd.Path)
		cmd.Args = append(cmd.Args, origArgs...)
		out, err = cmd.CombinedOutput()
	}
	return out, err
}

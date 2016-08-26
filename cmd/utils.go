package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"

	"github.com/drud/drud-go/secrets"
)

// RandomString returns a random string of len strlen
func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// CheckDependencies verifies dependensies of DRUD Cli.
func CheckDependencies() bool {
	// @todo verify versions of the dependencies
	_, dmachineerr := exec.LookPath("docker-machine")
	_, dockererr := exec.LookPath("docker")
	_, vboxerr := exec.LookPath("vboxmanage")

	deperr := false
	if dmachineerr != nil || dockererr != nil {
		fmt.Println("Please install docker toolbox to get access to `docker-machine` and `docker` commands. https://www.docker.com/docker-toolbox")
		deperr = true
	}
	if vboxerr != nil {
		fmt.Println("Please install virtualbox in order to get access to the `vboxmanage` command. https://www.virtualbox.org/wiki/Downloads")
		deperr = true
	}

	// @todo ensure the default docker-machine is running with increased specs
	// @todo ensure they have run `docker login` to generate the dockercfg.

	if deperr {
		log.Fatal("Unmet dependencies. See errors above.")
	}
	return !deperr
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

func empty(p string) bool {
	return p == ""
}

func byteify(s string) []byte {
	return []byte(s)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// NormalizePath prefixes secret paths with secret when necessary
func NormalizePath(sPath string) (newPath string) {
	newPath = sPath
	if !strings.HasPrefix(sPath, "secret/") || !strings.HasPrefix(sPath, "cubbyhole/") {
		if strings.HasPrefix(sPath, "/") {
			newPath = filepath.Join("secret", sPath[1:])
		} else {
			newPath = filepath.Join("secret", sPath)
		}
	}
	return
}

func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "yes"}
	nokayResponses := []string{"n", "no"}
	responseLower := strings.ToLower(response)

	if containsString(okayResponses, responseLower) {
		return true
	} else if containsString(nokayResponses, responseLower) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

// SetHomedir gets homedir and sets it to global homedir
func SetHomedir() {
	var err error
	pwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	// use the usr lib to get homedir then try $HOME if it does not work
	usr, err := user.Current()
	if err != nil {
		homedir = os.Getenv("HOME")
		if homedir == "" {
			log.Fatal("Could not write token to user's homedir")
		}
	} else {
		homedir = usr.HomeDir
	}
}

// SetConf sets global drudconf with abs path to conf file
func SetConf() {
	// figure out where the config is for reals
	var err error
	if cfgFile == "" {
		drudconf = filepath.Join(homedir, "drud.yaml")
	} else {
		if strings.HasPrefix(cfgFile, "/") {
			drudconf = cfgFile
		} else {
			drudconf, err = filepath.Abs(cfgFile)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

// EnableAvailablePackages gets token info and determines what functionality to enable based on policies
func EnableAvailablePackages() error {
	data, err := secrets.GetTokenDetails()
	if err != nil {
		return err
	}

	policies := data["policies"].([]interface{})
	// @todo: make policies like [files, hosting, etc] instead of using client name
	for _, p := range policies {
		switch p.(string) {
		case "root", "admin":
			drudAccess = true
			filesAccess = true
			clientCreateAccess = true
		case "1fee":
			filesAccess = true
		case "newmedia":
			filesAccess = true
		}
	}
	if drudAccess {
		RootCmd.AddCommand(HostingCmd)
		RootCmd.AddCommand(LocalCmd)
	}
	if filesAccess {
		RootCmd.AddCommand(FileCmd)
	}
	if clientCreateAccess {
		CreateCmd.AddCommand(clientCreateCmd)
		DeleteCmd.AddCommand(clientDeleteCmd)
		ListCmd.AddCommand(clientListCmd)
	}

	return nil

}

func getClientJWT(client string) ([]byte, error) {
	sobj := secrets.Secret{
		Path: fmt.Sprintf("secret/gce/%s-jwt", client),
	}

	err := sobj.Read()
	if err != nil {
		return []byte(""), err
	}

	b, err := json.Marshal(sobj.Data)
	if err != nil {
		return []byte(""), err
	}

	return b, nil
}

func getStorageClient(jwt []byte, ctx context.Context) (*storage.Client, error) {
	conf, err := google.JWTConfigFromJSON(
		jwt,
		storage.ScopeReadOnly,
	)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(ctx, cloud.WithTokenSource(conf.TokenSource(ctx)))
	if err != nil {
		return nil, err
	}

	return client, nil
}

// PrepLocalSiteDirs creates a site's directories for local dev in ~/.drud/client/site
func PrepLocalSiteDirs(base string) error {
	err := os.MkdirAll(base, os.FileMode(int(0774)))
	if err != nil {
		return err
	}

	dirs := []string{
		"src",
		"files",
		"data",
	}
	for _, d := range dirs {
		dirPath := path.Join(base, d)
		err := os.Mkdir(dirPath, os.FileMode(int(0774)))
		if err != nil {
			if !strings.Contains(err.Error(), "file exists") {
				return err
			}
		}
	}

	return nil
}

// getMAC returns the mac address for interface en0 or the first in the list otherwise
func getMAC() (string, error) {
	var macADDR string
	ifs, _ := net.Interfaces()
	for _, v := range ifs {
		h := v.HardwareAddr.String()
		if len(h) == 0 {
			continue
		}
		if v.Name == "en0" {
			macADDR = h
		}
	}
	if macADDR == "" {
		macADDR = ifs[0].HardwareAddr.String()
	}
	if macADDR == "" {
		return macADDR, fmt.Errorf("No MAC Addr found!")
	}
	return macADDR, nil
}

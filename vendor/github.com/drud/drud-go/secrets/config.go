package secrets

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
)

// SetVaultVars sets globals for vault access
func ConfigVault(tokenFile string, vaultHost string) string {
	// ensure there is a token for use with vault unless this is the auth command
	configEditor()
	var err error
	if _, err = os.Stat(tokenFile); os.IsNotExist(err) {
		log.Fatal("No sanctuary token found. Run `drud auth --help`")
	}

	vaultCFG := *api.DefaultConfig()
	vaultCFG.Address = vaultHost

	vClient, err = api.NewClient(&vaultCFG)
	if err != nil {
		fmt.Println(err)
	}

	var cTok string
	cTok, err = getSanctuaryToken(tokenFile)
	if err != nil {
		log.Fatalln("Error reading token file", err)
	}

	vClient.SetToken(cTok)

	vault = *vClient.Logical()
	
	return cTok
}

// GetTokenDetails returns a map of the user's token info
func GetTokenDetails() (map[string]interface{}, error) {
	sobj := Secret{
		Path: "/auth/token/lookup-self",
	}

	err := sobj.Read()
	if err != nil {
		return nil, err
	}

	return sobj.Data, nil
}

func getSanctuaryToken(tokenFile string) (string, error) {
	fileBytes, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(fileBytes)), nil
}

func configEditor() {
	// allow user to have different editor for secrets
	// fall back to default editor
	editor = os.Getenv("SECRET_EDITOR")
	if editor == "" {
		editor = os.Getenv("EDITOR")
		if editor == "" {
			editor = "atom -w"
		}
	}
}

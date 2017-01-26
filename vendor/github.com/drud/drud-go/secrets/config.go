package secrets

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
)

// ConfigVault sets globals for vault access
func ConfigVault(token string, vaultHost string) {
	// ensure there is a token for use with vault unless this is the auth command
	configEditor()
	var err error
	if token == "" {
		log.Fatal("Not authenticated. Run `drud auth --help`")
	}

	vaultCFG := *api.DefaultConfig()
	vaultCFG.Address = vaultHost

	vClient, err = api.NewClient(&vaultCFG)
	if err != nil {
		fmt.Println(err)
	}

	vClient.SetToken(token)

	vault = *vClient.Logical()
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

func configEditor() {
	// allow user to have different editor for secrets
	// fall back to default editor
	editor = os.Getenv("DRUD_SECRET_EDITOR")
	if editor == "" {
		editor = os.Getenv("EDITOR")
		if editor == "" {
			editor = "atom -w"
		}
	}
}

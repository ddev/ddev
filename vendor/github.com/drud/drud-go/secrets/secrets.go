package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/vault/api"
	vaultAPI "github.com/hashicorp/vault/api"
	vaultGithub "github.com/hashicorp/vault/builtin/credential/github"
	"github.com/xeipuuv/gojsonschema"
)

var (
	editor       string      // editor used for opening secrets
	vaultAddress string      // stores the vault host address
	vClient      *api.Client // vault api client instance
	vault        api.Logical // part of the vault go api
)

// Secret is an object used for workign with vault secrets
type Secret struct {
	Path    string
	Data    map[string]interface{}
	TmpFile string
}

func GetVault() api.Logical {
	return vault
}

// Init preps secret obj and handles existing secret issues
func (s *Secret) Init(args []string) error {

	secret, err := vault.Read(s.Path)
	if err != nil {
		return err
	}
	if secret != nil {
		s.Data = secret.Data
		s.PromptEdit("Secret already exists!")
		return nil
	}
	s.Data = make(map[string]interface{})

	// handle second (or more) args if they exist
	// otherwise open up a new text file for writing a yaml secret
	if len(args) >= 2 {
		// secret from file
		if strings.HasPrefix(args[1], "@") {
			file, err := filepath.Abs(args[1][1:])
			if err != nil {
				log.Fatal(err)
			}
			if _, err = os.Stat(file); os.IsNotExist(err) {
				return errors.New("file does not exist")
			}

			fileBytes, err := ioutil.ReadFile(file)
			if err != nil {
				return err
			}

			err = yaml.Unmarshal(fileBytes, &s.Data)
			if err != nil {
				s.Data["value"] = string(fileBytes)
			}

		} else if strings.Contains(args[1], "=") {
			//fmt.Println("key=value pairs passed")
			for _, pair := range args[1:] {
				pairs := strings.Split(pair, "=")
				s.Data[pairs[0]] = pairs[1]
			}
		} else {
			//fmt.Println("string passed")
			s.Data["vault"] = args[1]
		}

		if len(s.Data) == 0 {
			fmt.Println("Not able to create secret from input.")
			os.Exit(1)
		}

	} else {

		// open a temp file for working with the secret
		tmpfile, err := ioutil.TempFile("", "secret")
		if err != nil {
			log.Fatal(err)
		}
		tmpfile.Close()

		err = os.Rename(tmpfile.Name(), tmpfile.Name()+".yaml")
		if err != nil {
			log.Fatal(err)
		}
		s.TmpFile = tmpfile.Name() + ".yaml"

		s.Edit()
	}

	return nil
}

// PromptEdit checks if user wants to edit and calls edit if they do
func (s *Secret) PromptEdit(message string) {
	var answer string
	fmt.Println(message)
	fmt.Println("Edit secret? (Y/n): ")
	fmt.Scanf("%s", &answer)
	if strings.ToLower(answer) == "n" {
		os.Exit(1)
	} else {
		err := s.Edit()
		if err != nil {
			log.Fatal(err)
		}
		return
	}
}

// Edit handles opening of secrets into gui editors
func (s *Secret) Edit() error {

	tmpFileLoc := s.TmpFile
	if s.TmpFile == "" {
		tmpfile, err := ioutil.TempFile("", "secret")
		if err != nil {
			return err
		}
		if err = tmpfile.Close(); err != nil {
			return err
		}

		// rename file to have yaml extension the reopen
		newfile := tmpfile.Name() + ".yaml"
		err = os.Rename(tmpfile.Name(), newfile)
		if err != nil {
			return err
		}
		tmpfile, err = os.OpenFile(newfile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}

		content, err := s.ToYAML()
		if err != nil {
			return err
		}

		if _, err = tmpfile.Write(content); err != nil {
			return err
		}
		if err = tmpfile.Close(); err != nil {
			return err
		}
		tmpFileLoc = tmpfile.Name()
	}

	// editor var may have flags included so we need to split them off
	// for the LookPath command to work
	edparts := strings.Split(editor, " ")
	edpath, err := exec.LookPath(edparts[0])
	if err != nil {
		return errors.New("Couldnt find editor")
	}

	command := exec.Command(edpath, edparts[1], tmpFileLoc)
	err = command.Start()
	if err != nil {
		return err
	}
	log.Printf("Opening secret via '%s %s'", edpath, edparts[1])
	err = command.Wait()
	if err != nil {
		return err
	}
	err = s.UnMarshallEdit(tmpFileLoc)
	if err != nil {
		return err
	}
	return nil
}

// UnMarshallEdit gets secret data into its proper place in the Secret struct
func (s *Secret) UnMarshallEdit(filename string) error {

	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	log.Println("Updating secret at", s.Path)

	// convert yaml to json before converting back to go map
	// go-yaml wasnt properly parsing nested items so this workaround was
	jsonData, err := yaml.YAMLToJSON(fileBytes)
	// checks for trouble going from yaml to json
	if err != nil {
		s.TmpFile = filename
		s.PromptEdit("Error parsing yaml.")
		return nil
	}

	s.Data = make(map[string]interface{})
	err = json.Unmarshal(jsonData, &s.Data)
	// checks for trouble goign from json to go
	if err != nil {
		s.TmpFile = filename
		s.PromptEdit("Yaml invalid JSON.")
		return nil
	}

	if len(s.Data) == 0 {
		return errors.New("Not able to create secret from input")
	}

	return nil

}

//Read loads the secret from vault
func (s *Secret) Read() error {
	secret, err := vault.Read(s.Path)
	if err != nil {
		return err
	}
	if secret == nil {
		return errors.New("No secret.")
	}
	s.Data = secret.Data

	return nil
}

// List gets the sub-objects for a path
func (s *Secret) List() error {
	secret, err := vault.List(s.Path)
	if err != nil {
		return err
	}
	s.Data = secret.Data
	return nil
}

// Delete removes secret from vault
func (s *Secret) Delete() error {

	_, err := vault.Delete(s.Path)
	if err != nil {
		return err
	}
	return nil
}

// Write saves teh secret to vault
func (s *Secret) Write() error {
	fmt.Println("Creating secret at", s.Path)

	_, err := vault.Write(s.Path, s.Data)
	if err != nil {
		return err
	}
	return nil
}

// ToYAML converts secret to yaml string
func (s *Secret) ToYAML() ([]byte, error) {
	content, err := yaml.Marshal(s.Data)
	if err != nil {
		return []byte{}, err
	}
	return content, nil

}

// Validate checks for and applies schemas
func (s *Secret) Validate() (bool, error) {

	// get path to where a validation map would exist for secret's parent dir

	var sPath string
	if strings.HasPrefix(filepath.Dir(s.Path), "secret/") {
		sPath = filepath.Dir(s.Path)[7:]
	} else {
		sPath = filepath.Dir(s.Path)
	}

	mapPath := filepath.Join("secret/validation/maps", sPath)
	sMap, err := vault.Read(mapPath)
	if err != nil {
		return false, err
	}

	// no validation
	if sMap == nil {
		return true, nil
	}

	fmt.Println("Running validation...")

	// loop through schemas listed for this directory and test them against secret
	for _, v := range sMap.Data["schemas"].([]interface{}) {
		schema, err := vault.Read(filepath.Join("secret/validation/schemas", v.(string)))
		if err != nil {
			return false, err
		}

		schemaLoader := gojsonschema.NewGoLoader(schema.Data)
		documentLoader := gojsonschema.NewGoLoader(s.Data)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			return false, err
		}

		if !result.Valid() {
			fmt.Printf("The document is not valid. see errors :\n")
			for _, desc := range result.Errors() {
				fmt.Printf("- %s\n", desc)
			}
			return false, err
		}

	}
	return true, nil
}

// MustValidate loops Validate nutil the secret validates or user declines
func (s *Secret) MustValidate() error {
	valid := false
	var err error

	for {
		if valid {
			return nil
		}

		valid, err = s.Validate()
		if err != nil {
			return err
		}
		if !valid {
			s.PromptEdit("Secret does not validate.")
		}
	}
}

// GetJWT returns the jwt
func GetJWT(token string, vaultHost string, projectID string) ([]byte, error) {
	err := authVault(token, vaultHost)

	if err != nil {
		return []byte(""), err
	}
	// Get the JWT out vault and use the gcloud client to activate a service account
	project := strings.Split(projectID, "-")[0]
	sobj := Secret{
		Path: fmt.Sprintf("secret/gce/%s-jwt", project),
	}
	err = sobj.Read()
	if err != nil {
		return []byte(""), err
	}

	jwt, err := json.Marshal(sobj.Data)

	if err != nil {
		log.Println("Could not decode JWT file")
		return []byte(""), err
	}

	return jwt, nil
}

// authVault uses providedf git token and vault address to create an authenticated vautl client
func authVault(token string, vaultHost string) error {

	// Write our token to a tempfile and configure vault to use it.
	if token == "" || vaultHost == "" {
		return errors.New("Vault token or address not found")
	}

	vaultCFG := *vaultAPI.DefaultConfig()
	vaultCFG.Address = vaultHost
	vClient, err := vaultAPI.NewClient(&vaultCFG)
	if err != nil {
		log.Fatalln(err)
	}

	mountInput := map[string]string{
		"mount": "github",
		"token": token,
	}

	// create vault client instance
	cliHandler := vaultGithub.CLIHandler{}
	var cTok string

	cTok, err = cliHandler.Auth(vClient, mountInput)
	if err != nil {
		log.Fatalln("Error while authenticating with DRUD.", err)
	}

	tmpfile, err := ioutil.TempFile("", "vault")
	if err != nil {
		log.Fatal(err)
	}
	if _, err = tmpfile.Write([]byte(cTok)); err != nil {
		log.Fatal(err)
	}
	if err = tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	tmpFileLoc := tmpfile.Name()
	ConfigVault(tmpFileLoc, vaultHost)
	return nil
}

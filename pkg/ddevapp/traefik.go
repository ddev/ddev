package ddevapp

import (
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type TraeficRouting struct {
	ExternalHostname    string
	ExternalPort        string
	InternalServiceName string
	InternalServicePort string
	HTTPS               bool
}

func detectAppRouting(app *DdevApp) ([]TraeficRouting, error) {
	// app.ComposeYaml["services"];
	table := []TraeficRouting{}
	if services, ok := app.ComposeYaml["services"]; ok {

		for serviceName, s := range services.(map[interface{}]interface{}) {
			service := s.(map[interface{}]interface{})
			if env, ok := service["environment"].(map[interface{}]interface{}); ok {
				if httpExpose, ok := env["HTTP_EXPOSE"].(string); ok {
					fmt.Printf("HTTP_EXPOSE=%v for %s\n", httpExpose, serviceName)
					portPairs := strings.Split(httpExpose, ",")
					for _, portPair := range portPairs {
						// TODO: Implement VIRTUAL_HOST
						ports := strings.Split(portPair, ":")
						if len(ports) != 2 {
							util.Warning("Skipping bad HTTP_EXPOSE port pair spec %s for service %s", portPair, serviceName)
							continue
						}
						table = append(table, TraeficRouting{ExternalPort: ports[0], InternalServiceName: serviceName.(string), InternalServicePort: ports[1], HTTPS: false})
					}
				}
				//TODO: Consolidate these two usages
				if httpsExpose, ok := env["HTTPS_EXPOSE"].(string); ok {
					fmt.Printf("HTTPS_EXPOSE=%v for %s\n", httpsExpose, serviceName)
					portPairs := strings.Split(httpsExpose, ",")
					for _, portPair := range portPairs {
						// TODO: Implement VIRTUAL_HOST
						ports := strings.Split(portPair, ":")
						if len(ports) != 2 {
							util.Warning("Skipping bad HTTPS_EXPOSE port pair spec %s for service %s", portPair, serviceName)
							continue
						}
						table = append(table, TraeficRouting{ExternalPort: ports[0], InternalServiceName: serviceName.(string), InternalServicePort: ports[1], HTTPS: true})
					}
				}
			}
		}
	}
	return table, nil
}

func pushGlobalTraefikConfig() error {
	globalTraefikDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
	err := os.MkdirAll(globalTraefikDir, 0755)
	if err != nil {
		return fmt.Errorf("Failed to create global .ddev/traefik directory: %v", err)
	}
	sourceCertsPath := filepath.Join(globalTraefikDir, "certs")
	// SourceConfigDir for dynamic config
	sourceConfigDir := filepath.Join(globalTraefikDir, "config")
	targetCertsPath := path.Join("/mnt/ddev-global-cache/traefik/certs")

	err = os.MkdirAll(sourceCertsPath, 0755)
	if err != nil {
		return fmt.Errorf("Failed to create global traefik certs dir: %v", err)
	}
	err = os.MkdirAll(sourceConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("Failed to create global traefik config dir: %v", err)
	}

	// Assume that the #ddev-generated exists in file unless it doesn't
	sigExists := true
	for _, pemFile := range []string{"default_cert.crt", "default_key.key"} {
		origFile := filepath.Join(sourceCertsPath, pemFile)
		if fileutil.FileExists(origFile) {
			// Check to see if file has #ddev-generated in it, meaning we can recreate it.
			sigExists, err = fileutil.FgrepStringInFile(origFile, nodeps.DdevFileSignature)
			if err != nil {
				return err
			}
			// If either of the files has #ddev-generated, we will respect both
			if !sigExists {
				break
			}
		}
	}
	if sigExists {
		c := []string{"--cert-file", filepath.Join(sourceCertsPath, "default_cert.crt"), "--key-file", filepath.Join(sourceCertsPath, "default_key.key"), "*.ddev.site", "127.0.0.1", "localhost", "*.ddev.local", "ddev-router", "ddev-router.ddev", "ddev-router.ddev_default"}
		out, err := exec.RunHostCommand("mkcert", c...)
		if err != nil {
			util.Failed("failed to create certificates for app, check mkcert operation: %v", out)
		}

		// Prepend #ddev-generated in generated crt and key files
		for _, pemFile := range []string{"default_cert.crt", "default_key.key"} {
			origFile := filepath.Join(sourceCertsPath, pemFile)

			contents, err := fileutil.ReadFileIntoString(origFile)
			if err != nil {
				return fmt.Errorf("Failed to read file %v: %v", origFile, err)
			}
			contents = nodeps.DdevFileSignature + "\n" + contents
			err = fileutil.TemplateStringToFile(contents, nil, origFile)
			if err != nil {
				return err
			}
		}
	}

	type traefikData struct {
		App             *DdevApp
		Hostnames       []string
		PrimaryHostname string
		TargetCertsPath string
		RouterPorts     []string
	}
	templateData := traefikData{
		TargetCertsPath: targetCertsPath,
		RouterPorts:     determineRouterPorts(),
	}

	traefikYamlFile := filepath.Join(sourceConfigDir, "default_config.yaml")
	sigExists = true
	//TODO: Systematize this checking-for-signature, allow an arg to skip if empty
	fi, err := os.Stat(traefikYamlFile)
	// Don't use simple fileutil.FileExists() because of the danger of an empty file
	if err == nil && fi.Size() > 0 {
		// Check to see if file has #ddev-generated in it, meaning we can recreate it.
		sigExists, err = fileutil.FgrepStringInFile(traefikYamlFile, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}
	}
	if !sigExists {
		util.Debug("Not creating %s because it exists and is managed by user", traefikYamlFile)
	} else {
		f, err := os.Create(traefikYamlFile)
		if err != nil {
			util.Failed("failed to create traefik config file: %v", err)
		}
		t, err := template.New("traefik_global_config_template.yaml").Funcs(sprig.TxtFuncMap()).ParseFS(bundledAssets, "traefik_global_config_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_global_config_template.yaml: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_global_config_template.yaml with templatedate='%v':: %v", templateData, err)
		}
	}

	// sourceConfigDir for static config
	sourceConfigDir = globalTraefikDir
	traefikYamlFile = filepath.Join(sourceConfigDir, "static_config.yaml")
	sigExists = true
	//TODO: Systematize this checking-for-signature, allow an arg to skip if empty
	fi, err = os.Stat(traefikYamlFile)
	// Don't use simple fileutil.FileExists() because of the danger of an empty file
	if err == nil && fi.Size() > 0 {
		// Check to see if file has #ddev-generated in it, meaning we can recreate it.
		sigExists, err = fileutil.FgrepStringInFile(traefikYamlFile, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}
	}
	if !sigExists {
		util.Debug("Not creating %s because it exists and is managed by user", traefikYamlFile)
	} else {
		f, err := os.Create(traefikYamlFile)
		if err != nil {
			util.Failed("failed to create traefik config file: %v", err)
		}
		t, err := template.New("traefik_static_config_template.yaml").Funcs(sprig.TxtFuncMap()).ParseFS(bundledAssets, "traefik_static_config_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_static_config_template.yaml: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_global_config_template.yaml with templatedate='%v':: %v", templateData, err)
		}
	}
	uid, _, _ := util.GetContainerUIDGid()

	err = dockerutil.CopyIntoVolume(globalTraefikDir, "ddev-global-cache", "traefik", uid, "", false)
	if err != nil {
		util.Warning("failed to copy global traefik config into docker volume ddev-global-cache/traefik: %v", err)
	} else {
		util.Debug("Copied global traefik config in %s to ddev-global-cache/traefik", sourceCertsPath)
	}

	return nil
}

// configureTraefikForApp configures the dynamic configuration and creates cert+key
// in .ddev/traefik
func configureTraefikForApp(app *DdevApp) error {
	routingTable, err := detectAppRouting(app)
	if err != nil {
		return err
	}
	hostnames := app.GetHostnames()
	projectTraefikDir := app.GetConfigPath("traefik")
	err = os.MkdirAll(projectTraefikDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create .ddev/traefik directory: %v", err)
	}
	sourceCertsPath := filepath.Join(projectTraefikDir, "certs")
	sourceConfigDir := filepath.Join(projectTraefikDir, "config")
	targetCertsPath := path.Join("/mnt/ddev-global-cache/traefik/certs")

	err = os.MkdirAll(sourceCertsPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create traefik certs dir: %v", err)
	}
	err = os.MkdirAll(sourceConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create traefik config dir: %v", err)
	}

	baseName := filepath.Join(sourceCertsPath, app.Name)
	// Assume that the #ddev-generated exists in file unless it doesn't
	sigExists := true
	for _, pemFile := range []string{app.Name + ".crt", app.Name + ".key"} {
		origFile := filepath.Join(sourceCertsPath, pemFile)
		if fileutil.FileExists(origFile) {
			// Check to see if file has #ddev-generated in it, meaning we can recreate it.
			sigExists, err = fileutil.FgrepStringInFile(origFile, nodeps.DdevFileSignature)
			if err != nil {
				return err
			}
			// If either of the files has #ddev-generated, we will respect both
			if !sigExists {
				break
			}
		}
	}
	if sigExists {
		c := []string{"--cert-file", baseName + ".crt", "--key-file", baseName + ".key", "*.ddev.site", "127.0.0.1", "localhost", "*.ddev.local", "ddev-router", "ddev-router.ddev", "ddev-router.ddev_default"}
		c = append(c, hostnames...)
		out, err := exec.RunHostCommand("mkcert", c...)
		if err != nil {
			util.Failed("failed to create certificates for app, check mkcert operation: %v", out)
		}

		// Prepend #ddev-generated in generated crt and key files
		for _, pemFile := range []string{app.Name + ".crt", app.Name + ".key"} {
			origFile := filepath.Join(sourceCertsPath, pemFile)

			contents, err := fileutil.ReadFileIntoString(origFile)
			if err != nil {
				return fmt.Errorf("failed to read file %v: %v", origFile, err)
			}
			contents = nodeps.DdevFileSignature + "\n" + contents
			err = fileutil.TemplateStringToFile(contents, nil, origFile)
			if err != nil {
				return err
			}
		}
	}

	type traefikData struct {
		App             *DdevApp
		Hostnames       []string
		PrimaryHostname string
		TargetCertsPath string
		RoutingTable    []TraeficRouting
	}
	templateData := traefikData{
		App:             app,
		Hostnames:       app.GetHostnames(),
		PrimaryHostname: app.GetHostname(),
		TargetCertsPath: targetCertsPath,
		RoutingTable:    routingTable,
	}

	traefikYamlFile := filepath.Join(sourceConfigDir, app.Name+".yaml")
	sigExists = true
	fi, err := os.Stat(traefikYamlFile)
	// Don't use simple fileutil.FileExists() because of the danger of an empty file
	if err == nil && fi.Size() > 0 {
		// Check to see if file has #ddev-generated in it, meaning we can recreate it.
		sigExists, err = fileutil.FgrepStringInFile(traefikYamlFile, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}
	}
	if !sigExists {
		util.Debug("Not creating %s because it exists and is managed by user", traefikYamlFile)
	} else {
		f, err := os.Create(traefikYamlFile)
		if err != nil {
			util.Failed("failed to create traefik config file: %v", err)
		}
		t, err := template.New("traefik_config_template.yaml").Funcs(sprig.TxtFuncMap()).ParseFS(bundledAssets, "traefik_config_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_config_template.yaml: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_config_template.yaml with templatedate='%v':: %v", templateData, err)
		}
	}

	uid, _, _ := util.GetContainerUIDGid()

	err = dockerutil.CopyIntoVolume(projectTraefikDir, "ddev-global-cache", "traefik", uid, "", false)
	if err != nil {
		util.Warning("failed to copy traefik into docker volume ddev-global-cache/traefik: %v", err)
	} else {
		util.Debug("Copied traefik certs in %s to ddev-global-cache/traefik", sourceCertsPath)
	}
	return nil
}

package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

type TraefikRouting struct {
	ExternalHostnames []string
	ExternalPort      string
	Service           struct {
		ServiceName         string
		InternalServiceName string
		InternalServicePort string
	}
	HTTPS bool
}

// detectAppRouting reviews the configured services and uses their
// VIRTUAL_HOST and HTTP(S)_EXPOSE environment variables to set up routing
// for the project
func detectAppRouting(app *DdevApp) ([]TraefikRouting, error) {
	// app.ComposeYaml["services"];
	var table []TraefikRouting
	if services, ok := app.ComposeYaml["services"]; ok {
		for serviceName, s := range services.(map[string]interface{}) {
			service := s.(map[string]interface{})
			if env, ok := service["environment"].(map[string]interface{}); ok {
				var virtualHost string
				var ok bool
				if virtualHost, ok = env["VIRTUAL_HOST"].(string); ok {
					util.Debug("VIRTUAL_HOST=%v for %s", virtualHost, serviceName)
				}
				if virtualHost == "" {
					continue
				}
				hostnames := strings.Split(virtualHost, ",")
				if httpExpose, ok := env["HTTP_EXPOSE"].(string); ok && httpExpose != "" {
					util.Debug("HTTP_EXPOSE=%v for %s", httpExpose, serviceName)
					routeEntries, err := processHTTPExpose(serviceName, httpExpose, false, hostnames)
					if err != nil {
						return nil, err
					}
					table = append(table, routeEntries...)
				}

				if httpsExpose, ok := env["HTTPS_EXPOSE"].(string); ok && httpsExpose != "" {
					util.Debug("HTTPS_EXPOSE=%v for %s", httpsExpose, serviceName)
					routeEntries, err := processHTTPExpose(serviceName, httpsExpose, true, hostnames)
					if err != nil {
						return nil, err
					}
					table = append(table, routeEntries...)
				}
			}
		}
	}
	return table, nil
}

// processHTTPExpose creates routing table entry from VIRTUAL_HOST and HTTP(S)_EXPOSE
// environment variables
func processHTTPExpose(serviceName string, httpExpose string, isHTTPS bool, externalHostnames []string) ([]TraefikRouting, error) {
	var routingTable []TraefikRouting
	portPairs := strings.Split(httpExpose, ",")
	for _, portPair := range portPairs {
		ports := strings.Split(portPair, ":")
		if len(ports) == 0 || len(ports) > 2 {
			util.Warning("Skipping bad HTTP_EXPOSE port pair spec %s for service %s", portPair, serviceName)
			continue
		}
		if len(ports) == 1 {
			ports = append(ports, ports[0])
		}
		if ports[1] == "8025" && (globalconfig.DdevGlobalConfig.UseHardenedImages || globalconfig.DdevGlobalConfig.UseLetsEncrypt) {
			util.Debug("skipping port 8025 (mailpit) because not appropriate in casual hosting environment")
			continue
		}
		routingTable = append(routingTable, TraefikRouting{ExternalHostnames: externalHostnames, ExternalPort: ports[0],
			Service: struct {
				ServiceName         string
				InternalServiceName string
				InternalServicePort string
			}{
				ServiceName:         fmt.Sprintf("%s-%s", serviceName, ports[1]),
				InternalServiceName: serviceName,
				InternalServicePort: ports[1],
			}, HTTPS: isHTTPS})
	}
	return routingTable, nil
}

// PushGlobalTraefikConfig pushes the config into ddev-global-cache
func PushGlobalTraefikConfig() error {
	globalTraefikDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
	uid, _, _ := util.GetContainerUIDGid()
	err := os.MkdirAll(globalTraefikDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create global .ddev/traefik directory: %v", err)
	}
	sourceCertsPath := filepath.Join(globalTraefikDir, "certs")
	// SourceConfigDir for dynamic config
	sourceConfigDir := filepath.Join(globalTraefikDir, "config")
	targetCertsPath := path.Join("/mnt/ddev-global-cache/traefik/certs")

	err = os.MkdirAll(sourceCertsPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create global Traefik certs dir: %v", err)
	}
	err = os.MkdirAll(sourceConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create global Traefik config dir: %v", err)
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

	// If using Let's Encrypt, the default_cert.crt must not exist or
	// Traefik will use it.
	if globalconfig.DdevGlobalConfig.UseLetsEncrypt && sigExists {
		_ = os.RemoveAll(filepath.Join(sourceCertsPath, "default_cert.crt"))
		_ = os.RemoveAll(filepath.Join(sourceCertsPath, "default_key.key"))
		err = dockerutil.CopyIntoVolume(sourceCertsPath, "ddev-global-cache", "certs", uid, "", true)
		if err != nil {
			util.Warning("Failed to clear certs in ddev-global-cache volume certs directory: %v", err)
		}
	}
	// Install default certs, except when using Let's Encrypt (when they would
	// get used instead of Let's Encrypt certs)
	if !globalconfig.DdevGlobalConfig.UseLetsEncrypt && sigExists && globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		c := []string{"--cert-file", filepath.Join(sourceCertsPath, "default_cert.crt"), "--key-file", filepath.Join(sourceCertsPath, "default_key.key"), "127.0.0.1", "localhost", "*.ddev.local", "ddev-router", "ddev-router.ddev", "ddev-router.ddev_default", "*.ddev.site"}
		if globalconfig.DdevGlobalConfig.ProjectTldGlobal != "" {
			c = append(c, "*."+globalconfig.DdevGlobalConfig.ProjectTldGlobal)
		}

		out, err := exec.RunHostCommand("mkcert", c...)
		if err != nil {
			util.Failed("failed to create global mkcert certificate, check mkcert operation: %v", out)
		}

		// Prepend #ddev-generated in generated crt and key files
		for _, pemFile := range []string{"default_cert.crt", "default_key.key"} {
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
		App                *DdevApp
		Hostnames          []string
		PrimaryHostname    string
		TargetCertsPath    string
		RouterPorts        []string
		UseLetsEncrypt     bool
		LetsEncryptEmail   string
		TraefikMonitorPort string
	}
	templateData := traefikData{
		TargetCertsPath:    targetCertsPath,
		RouterPorts:        determineRouterPorts(),
		UseLetsEncrypt:     globalconfig.DdevGlobalConfig.UseLetsEncrypt,
		LetsEncryptEmail:   globalconfig.DdevGlobalConfig.LetsEncryptEmail,
		TraefikMonitorPort: globalconfig.DdevGlobalConfig.TraefikMonitorPort,
	}

	defaultConfigPath := filepath.Join(sourceConfigDir, "default_config.yaml")
	sigExists = true
	// TODO: Systematize this checking-for-signature, allow an arg to skip if empty
	fi, err := os.Stat(defaultConfigPath)
	// Don't use simple fileutil.FileExists() because of the danger of an empty file
	if err == nil && fi.Size() > 0 {
		// Check to see if file has #ddev-generated in it, meaning we can recreate it.
		sigExists, err = fileutil.FgrepStringInFile(defaultConfigPath, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}
	}
	if !sigExists {
		util.Debug("Not creating %s because it exists and is managed by user", defaultConfigPath)
	} else {
		f, err := os.Create(defaultConfigPath)
		if err != nil {
			util.Failed("Failed to create Traefik config file: %v", err)
		}
		defer f.Close()
		t, err := template.New("traefik_global_config_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "traefik_global_config_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_global_config_template.yaml: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_global_config_template.yaml with templatedate='%v':: %v", templateData, err)
		}
	}

	staticConfigFinalPath := filepath.Join(globalTraefikDir, ".static_config.yaml")

	staticConfigTemp, err := os.CreateTemp("", "static_config-")
	if err != nil {
		return err
	}

	t, err := template.New("traefik_static_config_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "traefik_static_config_template.yaml")
	if err != nil {
		return fmt.Errorf("could not create template from traefik_static_config_template.yaml: %v", err)
	}

	err = t.Execute(staticConfigTemp, templateData)
	if err != nil {
		return fmt.Errorf("could not parse traefik_static_config_template.yaml with templatedate='%v':: %v", templateData, err)
	}
	tmpFileName := staticConfigTemp.Name()
	err = staticConfigTemp.Close()
	if err != nil {
		return err
	}
	extraStaticConfigFiles, err := fileutil.GlobFilenames(globalTraefikDir, "static_config.*.yaml")
	if err != nil {
		return err
	}
	resultYaml, err := util.MergeYamlFiles(tmpFileName, extraStaticConfigFiles...)
	if err != nil {
		return err
	}

	finalYaml := "#ddev-generated\n" +
		"# Traefik \"static configuration\" file for DDEV\n" +
		"# DO NOT EDIT this file, your edits will be replaced.\n" +
		"# Although Traefik calls this \"static configuration\" it is changed\n" +
		"# regularly to update `entryPoints`\n\n" +
		resultYaml

	err = os.WriteFile(staticConfigFinalPath, []byte(finalYaml), 0755)
	if err != nil {
		return err
	}

	err = dockerutil.CopyIntoVolume(globalTraefikDir, "ddev-global-cache", "traefik", uid, "", false)
	if err != nil {
		return fmt.Errorf("failed to copy global Traefik config into Docker volume ddev-global-cache/traefik: %v", err)
	}
	util.Debug("Copied global Traefik config in %s to ddev-global-cache/traefik", sourceCertsPath)

	return nil
}

// ConfigureTraefikForApp configures the dynamic configuration and creates cert+key
// in .ddev/traefik
func ConfigureTraefikForApp(app *DdevApp) error {
	routingTable, err := detectAppRouting(app)
	if err != nil {
		return err
	}

	// hostnames here should be used only for creating the cert.
	hostnames := app.GetHostnames()
	// There can possibly be VIRTUAL_HOST entries which are not configured hostnames.
	for _, r := range routingTable {
		if r.ExternalHostnames != nil {
			hostnames = append(hostnames, r.ExternalHostnames...)
		}
	}
	hostnames = util.SliceToUniqueSlice(&hostnames)
	projectTraefikDir := app.GetConfigPath("traefik")
	err = os.MkdirAll(projectTraefikDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create .ddev/traefik directory: %v", err)
	}
	sourceCertsPath := filepath.Join(projectTraefikDir, "certs")
	sourceConfigDir := filepath.Join(projectTraefikDir, "config")
	targetCertsPath := path.Join("/mnt/ddev-global-cache/traefik/certs")
	customCertsPath := app.GetConfigPath("custom_certs")

	err = os.MkdirAll(sourceCertsPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create Traefik certs dir: %v", err)
	}
	err = os.MkdirAll(sourceConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create Traefik config dir: %v", err)
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
	// Assuming the certs don't exist, or they have #ddev-generated so can be replaced, create them
	// But not if we don't have mkcert already set up.
	if sigExists && globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		c := []string{"--cert-file", baseName + ".crt", "--key-file", baseName + ".key", "*.ddev.site", "127.0.0.1", "localhost", "*.ddev.local", "ddev-router", "ddev-router.ddev", "ddev-router.ddev_default"}
		c = append(c, hostnames...)
		if app.ProjectTLD != nodeps.DdevDefaultTLD {
			c = append(c, "*."+app.ProjectTLD)
		}
		out, err := exec.RunHostCommand("mkcert", c...)
		if err != nil {
			util.Failed("Failed to create certificates for project, check mkcert operation: %v; err=%v", out, err)
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
		RoutingTable    []TraefikRouting
		UseLetsEncrypt  bool
	}
	templateData := traefikData{
		App:             app,
		Hostnames:       []string{},
		PrimaryHostname: app.GetHostname(),
		TargetCertsPath: targetCertsPath,
		RoutingTable:    routingTable,
		UseLetsEncrypt:  globalconfig.DdevGlobalConfig.UseLetsEncrypt,
	}

	// Convert externalHostnames wildcards like `*.<anything>` to `[a-zA-Z0-9-]+.wild.ddev.site`
	for i, v := range routingTable {
		for j, h := range v.ExternalHostnames {
			if strings.HasPrefix(h, `*.`) {
				h = `[a-zA-Z0-9-]+` + strings.TrimPrefix(h, `*`)
				routingTable[i].ExternalHostnames[j] = h
			}
		}
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

		//create a temp file to hold the base traefik config while we merge and process it with the dynamic_config.*.yaml files
		dynamicConfigTemp, err := os.CreateTemp("", "dynamic_config-")
		if err != nil {
			return err
		}

		t, err := template.New("traefik_config_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "traefik_config_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_config_template.yaml: %v", err)
		}

		err = t.Execute(dynamicConfigTemp, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_config_template.yaml with templatedate='%v':: %v", templateData, err)
		}

		tmpFileName := dynamicConfigTemp.Name()
		err = dynamicConfigTemp.Close()
		if err != nil {
			return err
		}

		/* The following section reads the project/.ddev/traefik/dynamic_config.*.yaml files, fills any template placeholders in them with the
		App's templateData (for	targeting the appropriate routers (e.g. {projectname}-web-80-http) or for rewriting the App's URL in the
		response body), then merges their content into the base dynamic config YAML generated above, and is finally written to /project/.ddev/
		traefik/config/<project>.yaml. Allows for adding middlewares, overriding settings, etc...
		*/

		// find all dynamic_config.*.yaml files in the project's .ddev/traefik directory
		extraDynamicConfigFiles, err := fileutil.GlobFilenames(projectTraefikDir, "dynamic_config.*.yaml")
		if err != nil {
			return err
		}
		var finalYaml []byte
		// Check if extra config files were found. If not, no need to proceed.
		if len(extraDynamicConfigFiles) > 0 {

			// convert config files to maps and merge them, returning a yaml string
			resultYaml, err := util.MergeYamlFiles(tmpFileName, extraDynamicConfigFiles...)
			if err != nil {
				return err
			}

			// In the event that any of the extra configs contained go template {{ }} placeholders, create a new template and parse the YAML string into it.
			tmpl, err := template.New("dynamic_config_extras").Funcs(getTemplateFuncMap()).Parse(string(resultYaml))
			if err != nil {
				return fmt.Errorf("error parsing template: %s", err)
			}

			// Execute the template with the app's templateData
			var extraConfigProcessedYAML strings.Builder
			err = tmpl.Execute(&extraConfigProcessedYAML, templateData)
			if err != nil {
				return fmt.Errorf("error executing template: %s", err)
			}

			// convert the output to a string and prepend "#ddev-generated" to the string, and then convert all of it to []byte to be written to file
			finalYaml = []byte("#ddev-generated\n" +
				"# If you remove the ddev-generated line above you\n" +
				"# are responsible for maintaining this file. DDEV will not then\n" +
				"# update it, for example if you add `additional_hostnames`, etc.\n" +
				"# However, it is now unnecessary and imprudent to take control of\n" +
				"# this file, as there is now a dynamic merge mechanism. To use it,\n" +
				"# create <project>/.ddev/traefik/dynamic_config.*.yaml files which\n" +
				"# will then be merged in alphanumeric order. See the readme in the\n" +
				"# /traefik directory for more information\n\n" +
				extraConfigProcessedYAML.String())

		} else {

			// Since there weren't any dynamic_config.*.yaml files, we can just write the contents of the base temp file to .ddev/traefik/<project.yaml>
			finalYaml, err = os.ReadFile(tmpFileName)
			if err != nil {
				return fmt.Errorf("could not read tmpFileName: %v", err)
			}

		}

		// write the <project>.yaml file to the project's .ddev/traefik/config directory
		err = os.WriteFile(traefikYamlFile, finalYaml, 0755)
		if err != nil {
			return fmt.Errorf("could not write to traefikYamlFile: %v", err)
		}

		// Recreate the example config file on each ddev start, so as to both keep it up to date if we change things, as well as to always have
		// an example available there to be modified.
		// It is created from a template file, populating {{.App.Name}} where appropriate (e.g for router namespacing)
		// This is needed because using dotddev_assets/traefik/ to inject the example file does not really allow for filling out the templates,
		// which is desired so that users don't have to see or touch go templates

		dynamicExampleFile := filepath.Join(projectTraefikDir, "dynamic_config.middlewares.yaml.example")

		f, err := os.Create(dynamicExampleFile)
		if err != nil {
			return fmt.Errorf("failed to create Traefik config middlewares example file: %v", err)
		}

		t, err = template.New("traefik_dynamic_config_example_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "traefik_dynamic_config_example_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_dynamic_config_example_template.yaml: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_dynamic_config_example_template.yaml with templatedate='%v':: %v", templateData, err)
		}
	}

	uid, _, _ := util.GetContainerUIDGid()

	err = dockerutil.CopyIntoVolume(projectTraefikDir, "ddev-global-cache", "traefik", uid, "", false)
	if err != nil {
		util.Warning("Failed to copy Traefik into Docker volume ddev-global-cache/traefik: %v", err)
	} else {
		util.Debug("Copied Traefik certs in %s to ddev-global-cache/traefik", sourceCertsPath)
	}
	if fileutil.FileExists(filepath.Join(customCertsPath, fmt.Sprintf("%s.crt", app.Name))) {
		err = dockerutil.CopyIntoVolume(app.GetConfigPath("custom_certs"), "ddev-global-cache", "traefik/certs", uid, "", false)
		if err != nil {
			util.Warning("Failed copying custom certs into Docker volume ddev-global-cache/traefik/certs: %v", err)
		} else {
			util.Debug("Copied custom certs in %s to ddev-global-cache/traefik", sourceCertsPath)
		}
	}
	return nil
}

package ddevapp

import (
	"fmt"
	"os"
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
func detectAppRouting(app *DdevApp) ([]TraefikRouting, []string, error) {
	var table []TraefikRouting
	if app.ComposeYaml == nil || app.ComposeYaml.Services == nil {
		return table, nil, nil
	}
	for serviceName, service := range app.ComposeYaml.Services {
		var virtualHost string
		if virtualHostPointer, ok := service.Environment["VIRTUAL_HOST"]; ok && virtualHostPointer != nil && *virtualHostPointer != "" {
			virtualHost = *virtualHostPointer
			util.Debug("VIRTUAL_HOST=%v for %s", virtualHost, serviceName)
		}
		if virtualHost == "" {
			continue
		}
		hostnames := strings.Split(virtualHost, ",")
		if httpExposePointer, ok := service.Environment["HTTP_EXPOSE"]; ok && httpExposePointer != nil && *httpExposePointer != "" {
			httpExpose := *httpExposePointer
			util.Debug("HTTP_EXPOSE=%v for %s", httpExpose, serviceName)
			routeEntries, err := processHTTPExpose(serviceName, httpExpose, false, hostnames)
			if err != nil {
				return nil, nil, err
			}
			table = append(table, routeEntries...)
		}

		if httpsExposePointer, ok := service.Environment["HTTPS_EXPOSE"]; ok && httpsExposePointer != nil && *httpsExposePointer != "" {
			httpsExpose := *httpsExposePointer
			util.Debug("HTTPS_EXPOSE=%v for %s", httpsExpose, serviceName)
			routeEntries, err := processHTTPExpose(serviceName, httpsExpose, true, hostnames)
			if err != nil {
				return nil, nil, err
			}
			table = append(table, routeEntries...)
		}
	}

	hostnames := app.GetHostnames()
	// There can possibly be VIRTUAL_HOST entries which are not configured hostnames.
	for _, r := range table {
		if r.ExternalHostnames != nil {
			hostnames = append(hostnames, r.ExternalHostnames...)
		}
	}
	hostnames = util.SliceToUniqueSlice(&hostnames)

	return table, hostnames, nil
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
			util.Debug("skipping port 8025 (mailpit) because not appropriate in hosting environment")
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
func PushGlobalTraefikConfig(activeApps []*DdevApp) error {
	globalTraefikDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
	uid, _, _ := dockerutil.GetContainerUser()
	err := os.MkdirAll(globalTraefikDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create global .ddev/traefik directory: %v", err)
	}
	globalSourceCertsPath := filepath.Join(globalTraefikDir, "certs")
	// SourceConfigDir for dynamic config
	globalSourceConfigDir := filepath.Join(globalTraefikDir, "config")
	inContainerTargetCertsPath := "/mnt/ddev-global-cache/traefik/certs"

	// Set up directories in ~/.ddev/traefik
	err = os.MkdirAll(globalSourceCertsPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create global Traefik certs dir: %v", err)
	}
	err = os.MkdirAll(globalSourceConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create global Traefik config dir: %v", err)
	}

	// Assume that the #ddev-generated doesn't exist in files
	sigExists := false
	for _, pemFile := range []string{"default_cert.crt", "default_key.key"} {
		origFile := filepath.Join(globalSourceCertsPath, pemFile)
		// Check to see if file can be safely overwritten (has signature, is empty, or doesn't exist)
		err = fileutil.CheckSignatureOrNoFile(origFile, nodeps.DdevFileSignature)
		if err == nil {
			// File has a signature, or doesn't exists, or has no content - overwrite it
			sigExists = true
			break
		}
	}

	// If using Let's Encrypt, the default_cert.crt must not exist or
	// Traefik will use it.
	if globalconfig.DdevGlobalConfig.UseLetsEncrypt && sigExists {
		_ = os.RemoveAll(filepath.Join(globalSourceCertsPath, "default_cert.crt"))
		_ = os.RemoveAll(filepath.Join(globalSourceCertsPath, "default_key.key"))
	}
	// Install default certs, except when using Let's Encrypt (when they would
	// get used instead of Let's Encrypt certs)
	if !globalconfig.DdevGlobalConfig.UseLetsEncrypt && sigExists && globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		c := []string{"--cert-file", filepath.Join(globalSourceCertsPath, "default_cert.crt"), "--key-file", filepath.Join(globalSourceCertsPath, "default_key.key"), "127.0.0.1", "localhost", "*.ddev.local", "ddev-router", "ddev-router.ddev", "ddev-router.ddev_default", "*.ddev.site"}
		if globalconfig.DdevGlobalConfig.ProjectTldGlobal != "" {
			c = append(c, "*."+globalconfig.DdevGlobalConfig.ProjectTldGlobal)
		}

		out, err := exec.RunHostCommand("mkcert", c...)
		if err != nil {
			util.Failed("failed to create global mkcert certificate, check mkcert operation: %v", out)
		}

		// Prepend #ddev-generated in generated crt and key files
		for _, pemFile := range []string{"default_cert.crt", "default_key.key"} {
			origFile := filepath.Join(globalSourceCertsPath, pemFile)

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
		TargetCertsPath:    inContainerTargetCertsPath,
		RouterPorts:        determineRouterPorts(activeApps),
		UseLetsEncrypt:     globalconfig.DdevGlobalConfig.UseLetsEncrypt,
		LetsEncryptEmail:   globalconfig.DdevGlobalConfig.LetsEncryptEmail,
		TraefikMonitorPort: globalconfig.DdevGlobalConfig.TraefikMonitorPort,
	}

	defaultConfigPath := filepath.Join(globalSourceConfigDir, "default_config.yaml")
	// Check to see if file can be safely overwritten (has signature, is empty, or doesn't exist)
	err = fileutil.CheckSignatureOrNoFile(defaultConfigPath, nodeps.DdevFileSignature)
	sigExists = (err == nil)
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
	err = os.WriteFile(staticConfigFinalPath, []byte(resultYaml), 0755)
	if err != nil {
		return err
	}

	// Remove ~/.ddev/traefik/config and certs for clean start (regenerate from active projects)
	err = fileutil.PurgeDirectory(filepath.Join(globalTraefikDir, "config"))
	if err != nil {
		return fmt.Errorf("failed to purge global Traefik config dir: %v", err)
	}

	err = fileutil.PurgeDirectory(filepath.Join(globalTraefikDir, "certs"))
	if err != nil {
		return fmt.Errorf("failed to purge global Traefik config dir: %v", err)
	}

	// Track expected files in the volume for later sync
	expectedConfigs := map[string]bool{"default_config.yaml": true}
	expectedCerts := map[string]bool{}

	// Add default certs to expected list if not using Let's Encrypt
	if !globalconfig.DdevGlobalConfig.UseLetsEncrypt && globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		expectedCerts["default_cert.crt"] = true
		expectedCerts["default_key.key"] = true
	}

	// Copy active project configs and certs into the global traefik directory.
	// This ensures only running projects have their routing active in the router.
	for _, app := range activeApps {
		projectConfigDir := app.GetConfigPath("traefik/config")
		projectCertsDir := app.GetConfigPath("traefik/certs")

		// Mark this project's config as expected - even if we can't copy it now,
		// we don't want to remove an existing config from the volume
		expectedConfigs[app.Name+".yaml"] = true
		expectedCerts[app.Name+".crt"] = true
		expectedCerts[app.Name+".key"] = true

		// Copy project's config yaml to global config dir
		projectConfigFile := filepath.Join(projectConfigDir, app.Name+".yaml")
		if fileutil.FileExists(projectConfigFile) {
			destFile := filepath.Join(globalSourceConfigDir, app.Name+".yaml")
			err = fileutil.CopyFile(projectConfigFile, destFile)
			if err != nil {
				util.Warning("Failed to copy traefik config for project %s: %v", app.Name, err)
			}
		}

		// Copy project's certs to global certs dir
		for _, ext := range []string{".crt", ".key"} {
			projectCertFile := filepath.Join(projectCertsDir, app.Name+ext)
			if fileutil.FileExists(projectCertFile) {
				destFile := filepath.Join(globalSourceCertsPath, app.Name+ext)
				err = fileutil.CopyFile(projectCertFile, destFile)
				if err != nil {
					util.Warning("Failed to copy traefik cert for project %s: %v", app.Name, err)
				}
			}
		}

		// Copy custom certs from project's .ddev/custom_certs/ to global certs dir
		projectCustomCertsPath := app.GetConfigPath("custom_certs")
		customCertFile := filepath.Join(projectCustomCertsPath, app.Name+".crt")
		if fileutil.FileExists(customCertFile) {
			for _, ext := range []string{".crt", ".key"} {
				srcFile := filepath.Join(projectCustomCertsPath, app.Name+ext)
				if fileutil.FileExists(srcFile) {
					destFile := filepath.Join(globalSourceCertsPath, app.Name+ext)
					err = fileutil.CopyFile(srcFile, destFile)
					if err != nil {
						util.Warning("Failed to copy custom cert for project %s: %v", app.Name, err)
					} else {
						util.Debug("Copied custom cert %s to global traefik certs dir", srcFile)
					}
				}
			}
		}
	}

	// Copy user-managed custom global config files from ~/.ddev/traefik/custom-global-config/
	customGlobalConfigDir := filepath.Join(globalTraefikDir, "custom-global-config")
	if fileutil.IsDirectory(customGlobalConfigDir) {
		customFiles, err := fileutil.ListFilesInDir(customGlobalConfigDir)
		if err != nil {
			util.Warning("Failed to list custom global config files: %v", err)
		} else {
			for _, f := range customFiles {
				srcFile := filepath.Join(customGlobalConfigDir, f)
				destFile := filepath.Join(globalSourceConfigDir, f)
				err = fileutil.CopyFile(srcFile, destFile)
				if err != nil {
					util.Warning("Failed to copy custom global config file %s: %v", f, err)
				} else {
					util.Debug("Copied custom global config %s to traefik config dir", f)
					expectedConfigs[f] = true
				}
			}
		}
	}

	// Copy to volume (adds new files, overwrites existing)
	err = dockerutil.CopyIntoVolume(globalTraefikDir, "ddev-global-cache", "traefik", uid, "custom-global-config", false)
	if err != nil {
		return fmt.Errorf("failed to copy global Traefik config into Docker volume ddev-global-cache/traefik: %v", err)
	}
	util.Debug("Copied global Traefik config in %s to ddev-global-cache/traefik", globalTraefikDir)

	// Sync config directory - remove stale project configs from the volume
	actualConfigs, err := dockerutil.ListFilesInVolume("ddev-global-cache", "traefik/config")
	if err != nil {
		return fmt.Errorf("failed to list traefik config files in volume: %v", err)
	}
	var staleConfigs []string
	for _, f := range actualConfigs {
		if !expectedConfigs[f] {
			staleConfigs = append(staleConfigs, f)
		}
	}
	if len(staleConfigs) > 0 {
		err = dockerutil.RemoveFilesFromVolume("ddev-global-cache", "traefik/config", staleConfigs)
		if err != nil {
			return fmt.Errorf("failed to remove stale traefik configs from volume: %v", err)
		}
		util.Debug("Removed stale traefik configs from volume: %v", staleConfigs)
	}

	// Sync certs directory - remove stale project certs from the volume
	actualCerts, err := dockerutil.ListFilesInVolume("ddev-global-cache", "traefik/certs")
	if err != nil {
		return fmt.Errorf("failed to list traefik cert files in volume: %v", err)
	}
	var staleCerts []string
	for _, f := range actualCerts {
		if !expectedCerts[f] {
			staleCerts = append(staleCerts, f)
		}
	}
	if len(staleCerts) > 0 {
		err = dockerutil.RemoveFilesFromVolume("ddev-global-cache", "traefik/certs", staleCerts)
		if err != nil {
			return fmt.Errorf("failed to remove stale traefik certs from volume: %v", err)
		}
		util.Debug("Removed stale traefik certs from volume: %v", staleCerts)
	}

	return nil
}

// configureTraefikForApp configures the dynamic configuration and creates cert+key
// in .ddev/traefik/certs
func configureTraefikForApp(app *DdevApp) error {
	routingTable, hostnames, err := detectAppRouting(app)
	if err != nil {
		return err
	}
	projectTraefikDir := app.GetConfigPath("traefik")
	err = os.MkdirAll(projectTraefikDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create .ddev/traefik directory: %v", err)
	}
	projectSourceCertsPath := filepath.Join(projectTraefikDir, "certs")
	projectSourceConfigDir := filepath.Join(projectTraefikDir, "config")
	inContainerTargetCertsPath := "/mnt/ddev-global-cache/traefik/certs"

	err = os.MkdirAll(projectSourceCertsPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create project Traefik certs dir: %v", err)
	}
	err = os.MkdirAll(projectSourceConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create project Traefik config dir: %v", err)
	}

	baseName := filepath.Join(projectSourceCertsPath, app.Name)
	// Assume that the #ddev-generated doesn't exist in files
	sigExists := false
	for _, pemFile := range []string{app.Name + ".crt", app.Name + ".key"} {
		origFile := filepath.Join(projectSourceCertsPath, pemFile)
		// Check to see if file can be safely overwritten (has signature, is empty, or doesn't exist)
		err = fileutil.CheckSignatureOrNoFile(origFile, nodeps.DdevFileSignature)
		if err == nil {
			// File has a signature, or doesn't exists, or has no content - overwrite it
			sigExists = true
			break
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
			origFile := filepath.Join(projectSourceCertsPath, pemFile)

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
		TargetCertsPath: inContainerTargetCertsPath,
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

	projectTraefikYamlFile := filepath.Join(projectSourceConfigDir, app.Name+".yaml")
	// Check to see if file can be safely overwritten (has signature, is empty, or doesn't exist)
	err = fileutil.CheckSignatureOrNoFile(projectTraefikYamlFile, nodeps.DdevFileSignature)
	sigExists = (err == nil)
	if !sigExists {
		util.Debug("Not creating %s because it exists and is managed by user", projectTraefikYamlFile)
	} else {
		f, err := os.Create(projectTraefikYamlFile)
		if err != nil {
			return fmt.Errorf("failed to create Traefik config file: %v", err)
		}
		defer f.Close()
		t, err := template.New("traefik_config_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "traefik_config_template.yaml")
		if err != nil {
			return fmt.Errorf("could not create template from traefik_config_template.yaml: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse traefik_config_template.yaml with templatedate='%v':: %v", templateData, err)
		}
	}

	// Project config and certs are now collected and pushed by PushGlobalTraefikConfig
	// which handles all active projects in a single operation
	return nil
}

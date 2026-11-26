// Package dockerhint turns a failed connection to the Docker daemon into next
// steps, with commands a person or a coding agent can run without first
// reading the docs. It only inspects strings, so it is tested without Docker.
package dockerhint

import (
	"fmt"
	"regexp"
	"strings"
)

// Input describes the failed connection.
type Input struct {
	GOOS          string // runtime.GOOS of the host
	WSL2          bool   // the host is a WSL2 distro
	ContextName   string // current Docker context, empty if it couldn't be resolved
	Host          string // Docker endpoint, such as unix:///var/run/docker.sock
	DockerHostEnv string // value of DOCKER_HOST
	DockerCtxEnv  string // value of DOCKER_CONTEXT
	Err           error  // error from connecting
	DocsURL       string // troubleshooting page linked at the end
}

// Provider is a Docker provider recognized from the endpoint or context name.
type Provider string

// Providers recognized by DetectProvider.
const (
	ProviderUnknown       Provider = ""
	ProviderLima          Provider = "Lima"
	ProviderColima        Provider = "Colima"
	ProviderOrbStack      Provider = "OrbStack"
	ProviderRancher       Provider = "Rancher Desktop"
	ProviderDockerDesktop Provider = "Docker Desktop"
	ProviderPodman        Provider = "Podman"
	ProviderRootless      Provider = "rootless Docker"
	ProviderDockerCE      Provider = "Docker CE"
	ProviderRemote        Provider = "a remote Docker host"
)

var rootlessSocket = regexp.MustCompile(`^unix:///run/user/\d+/docker\.sock$`)

// DetectProvider guesses the provider from the endpoint and context name,
// and returns the Lima instance or Colima profile when the path names one.
func DetectProvider(goos, contextName, host string) (Provider, string) {
	lowerHost := strings.ToLower(host)
	switch {
	case strings.Contains(lowerHost, "/.lima/"):
		return ProviderLima, pathSegmentAfter(host, ".lima")
	case strings.Contains(lowerHost, "/.colima/") || strings.Contains(lowerHost, "/colima/"):
		profile := pathSegmentAfter(host, ".colima")
		if profile == "" {
			profile = pathSegmentAfter(host, "colima")
		}
		return ProviderColima, profile
	case contextName == "colima":
		return ProviderColima, "default"
	case strings.HasPrefix(contextName, "colima-"):
		return ProviderColima, strings.TrimPrefix(contextName, "colima-")
	case strings.HasPrefix(contextName, "lima-"):
		return ProviderLima, strings.TrimPrefix(contextName, "lima-")
	case strings.Contains(lowerHost, "orbstack") || contextName == "orbstack":
		return ProviderOrbStack, ""
	case strings.Contains(lowerHost, "/.rd/") || strings.Contains(contextName, "rancher-desktop"):
		return ProviderRancher, ""
	case strings.Contains(lowerHost, "podman"):
		return ProviderPodman, ""
	case contextName == "desktop-linux" || strings.Contains(lowerHost, "/.docker/run/docker.sock") ||
		strings.Contains(lowerHost, "/.docker/desktop/docker.sock") || strings.Contains(lowerHost, "dockerdesktop"):
		return ProviderDockerDesktop, ""
	case strings.HasPrefix(lowerHost, "tcp://") || strings.HasPrefix(lowerHost, "ssh://"):
		return ProviderRemote, ""
	case rootlessSocket.MatchString(host):
		return ProviderRootless, ""
	case goos == "windows" && strings.Contains(lowerHost, "docker_engine"):
		return ProviderDockerDesktop, ""
	case goos == "linux" && (host == "unix:///var/run/docker.sock" || host == "unix:///run/docker.sock"):
		return ProviderDockerCE, ""
	}
	return ProviderUnknown, ""
}

// pathSegmentAfter returns the path segment that follows segment in host,
// such as the instance name in unix:///Users/me/.lima/default/sock/docker.sock.
func pathSegmentAfter(host, segment string) string {
	parts := strings.Split(strings.ReplaceAll(host, `\`, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], segment) {
			next := parts[i+1]
			if next == "" || strings.HasSuffix(next, ".sock") {
				return ""
			}
			return next
		}
	}
	return ""
}

// ConnectionHelp returns what to try next after in.Err, ending with a link
// to in.DocsURL.
func ConnectionHelp(in Input) string {
	var b strings.Builder
	errText := ""
	if in.Err != nil {
		errText = strings.ToLower(in.Err.Error())
	}
	provider, instance := DetectProvider(in.GOOS, in.ContextName, in.Host)

	switch {
	case strings.Contains(errText, "context not found") || strings.Contains(errText, "context does not exist"):
		b.WriteString("The selected Docker context does not exist.")
		if in.DockerCtxEnv != "" {
			fmt.Fprintf(&b, " It comes from DOCKER_CONTEXT=%s; unset it to use the default context:\n\n", in.DockerCtxEnv)
			b.WriteString(indent(unsetEnvCommand(in.GOOS, "DOCKER_CONTEXT")))
			b.WriteString("\n\nOr")
		} else {
			b.WriteString(" ")
		}
		b.WriteString(" list the contexts and switch to the one for your Docker provider:\n\n")
		b.WriteString(indent("docker context ls", "docker context use <name>"))
		b.WriteString("\n")
		return finish(&b, in)
	case strings.Contains(errText, "permission denied") && provider == ProviderDockerCE:
		b.WriteString("Docker is running, but your user may not be allowed to use its socket. Add your user to the docker group, then log out and back in:\n\n")
		b.WriteString(indent("sudo usermod -aG docker $USER"))
		b.WriteString("\n\nOr apply the group to the current shell only:\n\n")
		b.WriteString(indent("newgrp docker"))
		b.WriteString("\n")
	default:
		writeStartHint(&b, in, provider, instance)
	}

	if in.DockerHostEnv != "" {
		fmt.Fprintf(&b, "\nDOCKER_HOST=%s is set, so DDEV uses it and ignores the Docker context. If it is out of date, unset it:\n\n", in.DockerHostEnv)
		b.WriteString(indent(unsetEnvCommand(in.GOOS, "DOCKER_HOST")))
		b.WriteString("\n")
	} else if provider != ProviderRemote {
		b.WriteString("\nIf you use a different Docker provider, list the Docker contexts and switch to its context:\n\n")
		b.WriteString(indent("docker context ls", "docker context use <name>"))
		b.WriteString("\n")
	}
	return finish(&b, in)
}

// writeStartHint says how to start the provider the endpoint belongs to.
func writeStartHint(b *strings.Builder, in Input, provider Provider, instance string) {
	location := ""
	if in.ContextName != "" && in.DockerHostEnv == "" {
		location = fmt.Sprintf(" (Docker context %q)", in.ContextName)
	}
	switch provider {
	case ProviderLima:
		if instance == "" {
			instance = "default"
		}
		fmt.Fprintf(b, "Docker is set up to use Lima instance %q%s, which does not seem to be running. Start it, then run the ddev command again:\n\n", instance, location)
		b.WriteString(indent("limactl start " + instance))
	case ProviderColima:
		fmt.Fprintf(b, "Docker is set up to use Colima%s, which does not seem to be running. Start it, then run the ddev command again:\n\n", location)
		if instance == "" || instance == "default" {
			b.WriteString(indent("colima start"))
		} else {
			b.WriteString(indent("colima start --profile " + instance))
		}
	case ProviderOrbStack:
		fmt.Fprintf(b, "Docker is set up to use OrbStack%s, which does not seem to be running. Start it, then run the ddev command again:\n\n", location)
		b.WriteString(indent("orbctl start"))
	case ProviderRancher:
		fmt.Fprintf(b, "Docker is set up to use Rancher Desktop%s, which does not seem to be running. Start the Rancher Desktop app, or run the following, then run the ddev command again:\n\n", location)
		b.WriteString(indent("rdctl start"))
	case ProviderPodman:
		fmt.Fprintf(b, "Docker is set up to use Podman%s, which does not seem to be running. Start it, then run the ddev command again:\n\n", location)
		if in.GOOS == "linux" {
			b.WriteString(indent("systemctl --user start podman.socket"))
		} else {
			b.WriteString(indent("podman machine start"))
		}
	case ProviderDockerDesktop:
		fmt.Fprintf(b, "Docker is set up to use Docker Desktop%s, which does not seem to be running. Start it, then run the ddev command again:\n\n", location)
		switch in.GOOS {
		case "darwin":
			b.WriteString(indent("open -a Docker"))
		case "windows":
			b.WriteString(indent(`Start-Process "$env:ProgramFiles\Docker\Docker\Docker Desktop.exe"`))
		default:
			b.WriteString(indent("systemctl --user start docker-desktop"))
		}
	case ProviderRootless:
		fmt.Fprintf(b, "Docker is set up to use rootless Docker at %s, which does not seem to be running. Start it, then run the ddev command again:\n\n", in.Host)
		b.WriteString(indent("systemctl --user start docker"))
	case ProviderRemote:
		fmt.Fprintf(b, "Docker is set up to use the remote Docker host %s%s. Check that the host is reachable and its Docker daemon is running, then run the ddev command again.\n", in.Host, location)
		return
	case ProviderDockerCE:
		if in.WSL2 {
			b.WriteString("Docker is not running in this WSL2 distro. If you use Docker Desktop, start it in Windows and enable this distro in Settings > Resources > WSL Integration. If you installed Docker inside WSL2, start it:\n\n")
		} else {
			fmt.Fprintf(b, "The Docker daemon at %s does not seem to be running. Start it, then run the ddev command again:\n\n", in.Host)
		}
		b.WriteString(indent("sudo systemctl start docker"))
	default:
		if in.Host != "" {
			fmt.Fprintf(b, "DDEV could not reach Docker at %s%s. Start your Docker provider (such as OrbStack, Docker Desktop, Colima, Lima, or Rancher Desktop), then run the ddev command again.\n", in.Host, location)
		} else {
			b.WriteString("DDEV could not reach Docker. Start your Docker provider, then run the ddev command again.\n")
		}
		return
	}
	b.WriteString("\n")
}

// unsetEnvCommand returns the shell command that removes name from the
// environment of the current shell.
func unsetEnvCommand(goos, name string) string {
	if goos == "windows" {
		return "Remove-Item Env:" + name
	}
	return "unset " + name
}

func indent(lines ...string) string {
	return "    " + strings.Join(lines, "\n    ")
}

func finish(b *strings.Builder, in Input) string {
	if in.DocsURL != "" {
		fmt.Fprintf(b, "\nIf that does not help, see %s", in.DocsURL)
	}
	return b.String()
}

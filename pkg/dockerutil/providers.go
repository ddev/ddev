package dockerutil

import "strings"

// IsDockerDesktop detects if running on Docker Desktop
func IsDockerDesktop() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.OperatingSystem, "Docker Desktop") {
		return true
	}
	if strings.Contains(info.Name, "docker-desktop") {
		return true
	}
	return false
}

// IsColima detects if running on Colima
func IsColima() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.Name, "colima") {
		return true
	}
	return false
}

// IsLima detects if running on lima
func IsLima() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	// Rancher Desktop uses "lima-rancher-desktop" as its name
	if strings.Contains(info.Name, "rancher-desktop") {
		return false
	}
	if strings.HasPrefix(info.Name, "lima") {
		return true
	}
	return false
}

// IsRancherDesktop detects if running on Rancher Desktop
func IsRancherDesktop() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.OperatingSystem, "Rancher Desktop") {
		return true
	}
	if strings.Contains(info.Name, "rancher-desktop") {
		return true
	}
	return false
}

// IsOrbStack detects if running on OrbStack
func IsOrbStack() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.OperatingSystem, "OrbStack") {
		return true
	}
	if strings.Contains(info.Name, "orbstack") {
		return true
	}
	return false
}

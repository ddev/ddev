package globalconfig

// Container types used with ddev (duplicated from ddevapp, avoiding cross-package cycles)
const (
	DdevSSHAgentContainer = "ddev-ssh-agent"
	DBAContainer          = "dba"
)

var ValidOmitContainers = map[string]bool{
	DdevSSHAgentContainer: true,
	DBAContainer:          true,
}

// IsValidOmitContainers is a helper function to determine if a the OmitContainers array is valid
func IsValidOmitContainers(containerList []string) bool {
	for _, containerName := range containerList {
		if _, ok := ValidOmitContainers[containerName]; !ok {
			return false
		}
	}
	return true
}

// GetValidOmitContainers is a helper function that returns a list of valid containers for OmitContainers.
func GetValidOmitContainers() []string {
	s := make([]string, 0, len(ValidOmitContainers))

	for p := range ValidOmitContainers {
		s = append(s, p)
	}

	return s
}

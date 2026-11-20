/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package loader

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/errdefs"
	"github.com/compose-spec/compose-go/v2/graph"
	"github.com/compose-spec/compose-go/v2/types"
)

// checkConsistency validate a compose model is consistent
func checkConsistency(project *types.Project) error {
	for name, s := range project.Services {
		if err := checkService(project, name, &s); err != nil {
			return err
		}
	}

	if err := checkJobNames(project); err != nil {
		return err
	}
	if err := checkJobsDependsOn(project); err != nil {
		return err
	}
	if err := checkSecrets(project); err != nil {
		return err
	}

	return graph.CheckCycle(project)
}

// checkService validate a service is consistent with the project it belongs to
func checkService(project *types.Project, name string, s *types.ServiceConfig) error {
	if s.Build == nil && s.Image == "" && s.Provider == nil {
		return fmt.Errorf("service %q has neither an image nor a build context specified: %w", s.Name, errdefs.ErrInvalid)
	}

	if s.Build != nil {
		if err := checkServiceBuild(project, name, s); err != nil {
			return err
		}
	}

	if err := checkServiceNetworks(project, s); err != nil {
		return err
	}
	if err := checkServiceHealthCheck(s); err != nil {
		return err
	}
	if err := checkServiceDependsOn(project, s); err != nil {
		return err
	}
	if err := checkServiceNetworkModeService(project, s); err != nil {
		return err
	}
	if err := checkServiceReferences(project, s); err != nil {
		return err
	}
	if err := checkServiceScale(s); err != nil {
		return err
	}
	if err := checkServiceResources(s); err != nil {
		return err
	}
	if err := checkServiceContainerName(s); err != nil {
		return err
	}
	if err := checkServiceDevelop(s); err != nil {
		return err
	}
	return checkServiceMounts(s)
}

// checkServiceBuild validate the build section of a service
func checkServiceBuild(project *types.Project, name string, s *types.ServiceConfig) error {
	if s.Build.DockerfileInline != "" && s.Build.Dockerfile != "" {
		return fmt.Errorf("service %q declares mutually exclusive dockerfile and dockerfile_inline: %w", s.Name, errdefs.ErrInvalid)
	}

	if err := checkBuildAdditionalContexts(project, name, s.Build); err != nil {
		return err
	}

	if len(s.Build.Platforms) > 0 && s.Platform != "" {
		var found bool
		for _, platform := range s.Build.Platforms {
			if platform == s.Platform {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("service.build.platforms MUST include service.platform %q: %w", s.Platform, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkBuildAdditionalContexts validate additional contexts pointing to a service refer to a buildable one
func checkBuildAdditionalContexts(project *types.Project, name string, build *types.BuildConfig) error {
	for add, c := range build.AdditionalContexts {
		if target, ok := strings.CutPrefix(c, types.ServicePrefix); ok {
			t, err := project.GetService(target)
			if errors.Is(err, errdefs.ErrDisabled) {
				t = project.DisabledServices[target]
				err = nil
			}
			if err != nil {
				return fmt.Errorf("service %q declares unknown service %q as additional contexts %s: %w", name, target, add, errdefs.ErrInvalid)
			}
			if t.Build == nil {
				return fmt.Errorf("service %q declares non-buildable service %q as additional contexts %s: %w", name, target, add, errdefs.ErrInvalid)
			}
		}
	}
	return nil
}

// checkServiceNetworks validate a service networks are declared, and not set along with network_mode
func checkServiceNetworks(project *types.Project, s *types.ServiceConfig) error {
	if s.NetworkMode != "" && len(s.Networks) > 0 {
		return fmt.Errorf("service %s declares mutually exclusive `network_mode` and `networks`: %w", s.Name, errdefs.ErrInvalid)
	}
	for network := range s.Networks {
		if _, ok := project.Networks[network]; !ok {
			return fmt.Errorf("service %q refers to undefined network %s: %w", s.Name, network, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkServiceHealthCheck validate the healthcheck test of a service starts with a supported keyword
func checkServiceHealthCheck(s *types.ServiceConfig) error {
	if s.HealthCheck != nil && len(s.HealthCheck.Test) > 0 {
		switch s.HealthCheck.Test[0] {
		case "CMD", "CMD-SHELL", "NONE":
		default:
			return fmt.Errorf(`healthcheck.test must start either by "CMD", "CMD-SHELL" or "NONE": %w`, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkServiceDependsOn validate a service only depends on services, which may be disabled when optional
func checkServiceDependsOn(project *types.Project, s *types.ServiceConfig) error {
	for dependedService, cfg := range s.DependsOn {
		if _, err := project.GetService(dependedService); err != nil {
			if errors.Is(err, errdefs.ErrDisabled) && !cfg.Required {
				continue
			}
			if _, isJob := project.Jobs[dependedService]; isJob {
				return fmt.Errorf("service %q cannot depend on job %q: services can only depend on other services: %w", s.Name, dependedService, errdefs.ErrInvalid)
			}
			return fmt.Errorf("service %q depends on undefined service %q: %w", s.Name, dependedService, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkServiceNetworkModeService validate a network_mode pointing to a service refers to an enabled one
func checkServiceNetworkModeService(project *types.Project, s *types.ServiceConfig) error {
	if strings.HasPrefix(s.NetworkMode, types.ServicePrefix) {
		serviceName := s.NetworkMode[len(types.ServicePrefix):]
		if _, err := project.GetServices(serviceName); err != nil {
			if errors.Is(err, errdefs.ErrDisabled) {
				return fmt.Errorf("service %q declares disabled service %q for network_mode: %w", s.Name, serviceName, errdefs.ErrInvalid)
			}
			return fmt.Errorf("service %q declares unknown service %q for network_mode: %w", s.Name, serviceName, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkServiceReferences validate the volumes, secrets, configs and models a service uses are declared
func checkServiceReferences(project *types.Project, s *types.ServiceConfig) error {
	for _, volume := range s.Volumes {
		if volume.Type == types.VolumeTypeVolume && volume.Source != "" { // non anonymous volumes
			if _, ok := project.Volumes[volume.Source]; !ok {
				return fmt.Errorf("service %q refers to undefined volume %s: %w", s.Name, volume.Source, errdefs.ErrInvalid)
			}
		}
	}
	if s.Build != nil {
		for _, secret := range s.Build.Secrets {
			if _, ok := project.Secrets[secret.Source]; !ok {
				return fmt.Errorf("service %q refers to undefined build secret %s: %w", s.Name, secret.Source, errdefs.ErrInvalid)
			}
		}
	}
	for _, config := range s.Configs {
		if _, ok := project.Configs[config.Source]; !ok {
			return fmt.Errorf("service %q refers to undefined config %s: %w", s.Name, config.Source, errdefs.ErrInvalid)
		}
	}

	for model := range s.Models {
		if _, ok := project.Models[model]; !ok {
			return fmt.Errorf("service %q refers to undefined model %s: %w", s.Name, model, errdefs.ErrInvalid)
		}
	}

	for _, secret := range s.Secrets {
		if _, ok := project.Secrets[secret.Source]; !ok {
			return fmt.Errorf("service %q refers to undefined secret %s: %w", s.Name, secret.Source, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkServiceScale validate scale and deploy.replicas agree and are not negative.
// When both are set, deploy.replicas is completed with the scale.
func checkServiceScale(s *types.ServiceConfig) error {
	if s.Scale != nil && s.Deploy != nil {
		if s.Deploy.Replicas != nil && *s.Scale != *s.Deploy.Replicas {
			return fmt.Errorf("services.%s: can't set distinct values on 'scale' and 'deploy.replicas': %w",
				s.Name, errdefs.ErrInvalid)
		}
		s.Deploy.Replicas = s.Scale
	}

	if s.Scale != nil && *s.Scale < 0 {
		return fmt.Errorf("services.%s.scale: must be greater than or equal to 0: %w", s.Name, errdefs.ErrInvalid)
	}
	if s.Deploy != nil && s.Deploy.Replicas != nil && *s.Deploy.Replicas < 0 {
		return fmt.Errorf("services.%s.deploy.replicas: must be greater than or equal to 0: %w", s.Name, errdefs.ErrInvalid)
	}
	return nil
}

// checkServiceResources validate the limits set on a service agree with the ones set by deploy.resources
func checkServiceResources(s *types.ServiceConfig) error {
	if s.CPUS != 0 && s.Deploy != nil {
		if s.Deploy.Resources.Limits != nil && s.Deploy.Resources.Limits.NanoCPUs.Value() != s.CPUS {
			return fmt.Errorf("services.%s: can't set distinct values on 'cpus' and 'deploy.resources.limits.cpus': %w",
				s.Name, errdefs.ErrInvalid)
		}
	}
	if s.MemLimit != 0 && s.Deploy != nil {
		if s.Deploy.Resources.Limits != nil && s.Deploy.Resources.Limits.MemoryBytes != s.MemLimit {
			return fmt.Errorf("services.%s: can't set distinct values on 'mem_limit' and 'deploy.resources.limits.memory': %w",
				s.Name, errdefs.ErrInvalid)
		}
	}
	if s.MemReservation != 0 && s.Deploy != nil {
		if s.Deploy.Resources.Reservations != nil && s.Deploy.Resources.Reservations.MemoryBytes != s.MemReservation {
			return fmt.Errorf("services.%s: can't set distinct values on 'mem_reservation' and 'deploy.resources.reservations.memory': %w",
				s.Name, errdefs.ErrInvalid)
		}
	}
	if s.PidsLimit != 0 && s.Deploy != nil {
		if s.Deploy.Resources.Limits != nil && s.Deploy.Resources.Limits.Pids != s.PidsLimit {
			return fmt.Errorf("services.%s: can't set distinct values on 'pids_limit' and 'deploy.resources.limits.pids': %w",
				s.Name, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkServiceContainerName validate a service with a container_name is not scaled, as the name must be unique
func checkServiceContainerName(s *types.ServiceConfig) error {
	if s.GetScale() > 1 && s.ContainerName != "" {
		attr := "scale"
		if s.Scale == nil {
			attr = "deploy.replicas"
		}
		return fmt.Errorf("services.%s: can't set container_name and %s as container name must be unique: %w",
			s.Name, attr, errdefs.ErrInvalid)
	}
	return nil
}

// checkServiceDevelop validate the develop.watch triggers of a service declare a target when required
func checkServiceDevelop(s *types.ServiceConfig) error {
	if s.Develop != nil && s.Develop.Watch != nil {
		for _, watch := range s.Develop.Watch {
			if watch.Target == "" && watch.Action != types.WatchActionRebuild && watch.Action != types.WatchActionRestart {
				return fmt.Errorf("services.%s.develop.watch: target is required for non-rebuild actions: %w", s.Name, errdefs.ErrInvalid)
			}
		}
	}
	return nil
}

// checkServiceMounts validate a service does not mount two tmpfs or volumes on the same target
func checkServiceMounts(s *types.ServiceConfig) error {
	mounts := map[string]string{}
	for i, tmpfs := range s.Tmpfs {
		loc := fmt.Sprintf("services.%s.tmpfs[%d]", s.Name, i)
		path, _, _ := strings.Cut(tmpfs, ":")
		if p, ok := mounts[path]; ok {
			return fmt.Errorf("%s: target %s already mounted as %s: %w", loc, path, p, errdefs.ErrInvalid)
		}
		mounts[path] = loc
	}
	for i, volume := range s.Volumes {
		loc := fmt.Sprintf("services.%s.volumes[%d]", s.Name, i)
		if p, ok := mounts[volume.Target]; ok {
			return fmt.Errorf("%s: target %s already mounted as %s: %w", loc, volume.Target, p, errdefs.ErrInvalid)
		}
		mounts[volume.Target] = loc
	}
	return nil
}

// checkJobNames validate names are unique across services and jobs so a name always
// resolves to exactly one of them (e.g. `docker compose run <name>`),
// including profile-disabled ones which may be enabled later
func checkJobNames(project *types.Project) error {
	for name := range project.AllJobs() {
		if _, ok := project.Services[name]; ok {
			return fmt.Errorf("%q is declared both as a service and a job: service and job names must be unique: %w", name, errdefs.ErrInvalid)
		}
		if _, ok := project.DisabledServices[name]; ok {
			return fmt.Errorf("%q is declared both as a service and a job: service and job names must be unique: %w", name, errdefs.ErrInvalid)
		}
	}
	return nil
}

// checkJobsDependsOn validate the jobs only depend on declared services and jobs
func checkJobsDependsOn(project *types.Project) error {
	for name, j := range project.Jobs {
		// a job can depend on services and on other jobs
		for depended, cfg := range j.DependsOn {
			if _, isJob := project.Jobs[depended]; isJob {
				continue
			}
			if _, err := project.GetService(depended); err != nil {
				if errors.Is(err, errdefs.ErrDisabled) && !cfg.Required {
					continue
				}
				return fmt.Errorf("job %q depends on undefined service or job %q: %w", name, depended, errdefs.ErrInvalid)
			}
		}
	}
	return nil
}

// checkSecrets validate the secrets of the project declare their source, unless external
func checkSecrets(project *types.Project) error {
	for name, secret := range project.Secrets {
		if secret.External {
			continue
		}
		if secret.File == "" && secret.Environment == "" {
			return fmt.Errorf("secret %q must declare either `file` or `environment`: %w", name, errdefs.ErrInvalid)
		}
	}
	return nil
}

package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
)

// describeImageForService returns the image name to show in `ddev describe`
// for a service, preferring its Dockerfile-derived base image(s) over the
// legacy "-<project>-built" tag-suffix guess.
func describeImageForService(service composeTypes.ServiceConfig, fallbackImage, appName string) string {
	if baseImages := resolveBuildBaseImages(service); len(baseImages) > 0 {
		return strings.Join(baseImages, ", ")
	}
	return strings.TrimSuffix(fallbackImage, fmt.Sprintf("-%s-built", appName))
}

// resolveBuildBaseImages returns the external, pullable base images a compose
// service's `build:` section resolves to, by reading the Dockerfile itself
// (or `dockerfile_inline`) instead of trying to derive it from the service's
// `image:` tag. That tag is only ever a local name chosen by the project or
// add-on author and isn't a reliable source for the upstream image DDEV needs
// to pre-pull.
//
// It returns nil when the service has no build section, or when the
// Dockerfile can't be read or parsed, so callers can fall back to their
// existing image-name-derived behavior.
func resolveBuildBaseImages(service composeTypes.ServiceConfig) []string {
	if service.Build == nil {
		return nil
	}

	var content []byte
	if service.Build.DockerfileInline != "" {
		content = []byte(service.Build.DockerfileInline)
	} else {
		dockerfile := service.Build.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}
		// service.Build.Context is resolved to an absolute path by the
		// compose loader by the time app.ComposeYaml is populated.
		b, err := os.ReadFile(filepath.Join(service.Build.Context, dockerfile))
		if err != nil {
			return nil
		}
		content = b
	}

	result, err := parser.Parse(strings.NewReader(string(content)))
	if err != nil {
		return nil
	}
	stages, metaArgs, err := instructions.Parse(result.AST, nil)
	if err != nil {
		return nil
	}

	// ARGs usable in a FROM line are only those declared at the top of the
	// Dockerfile (before the first stage); a compose `build.args` value
	// overrides the Dockerfile's own default, same as `docker build --build-arg`.
	// Each raw value is itself run through the shell lexer as it's added, so
	// quoting (`ARG X="scratch"`) and any reference to an earlier ARG resolve
	// the same way BuildKit resolves them.
	lex := shell.NewLex('\\')
	env := map[string]string{}
	setArg := func(key, rawValue string) {
		var envSlice []string
		for k, v := range env {
			envSlice = append(envSlice, k+"="+v)
		}
		expanded, _, err := lex.ProcessWord(rawValue, shell.EnvsFromSlice(envSlice))
		if err != nil {
			expanded = rawValue
		}
		env[key] = expanded
	}
	for _, a := range metaArgs {
		for _, kv := range a.Args {
			if kv.Value != nil {
				setArg(kv.Key, *kv.Value)
			}
		}
	}
	for k, v := range service.Build.Args {
		if v != nil {
			setArg(k, *v)
		}
	}
	var envSlice []string
	for k, v := range env {
		envSlice = append(envSlice, k+"="+v)
	}

	stageNames := map[string]bool{}
	for _, s := range stages {
		if s.Name != "" {
			stageNames[s.Name] = true
		}
	}

	seen := map[string]bool{}
	var images []string
	for _, s := range stages {
		base, _, err := lex.ProcessWord(s.BaseName, shell.EnvsFromSlice(envSlice))
		if err != nil || base == "" {
			continue
		}
		// A stage can be based on an earlier named stage instead of an
		// upstream image; that's not something to pull.
		if stageNames[base] {
			continue
		}
		if !seen[base] {
			seen[base] = true
			images = append(images, base)
		}
	}
	return images
}

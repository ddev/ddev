package ddevapp

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/stretchr/testify/require"
)

func TestBaseImageFromBuiltTag(t *testing.T) {
	tests := []struct {
		image, appName, service, want string
	}{
		{"ubuntu:24.04-acme-built", "acme", "pi", "ubuntu:24.04"},
		{"ubuntu:24.04-acme-pi-built", "acme", "pi", "ubuntu:24.04"},
		{"ubuntu:24.04-ddev.com-pi-built", "ddev.com", "pi", "ubuntu:24.04"},
		{"localhost:5000/img:1-acme-pi-built", "acme", "pi", "localhost:5000/img:1"},
		// Only the service's own name counts as a qualifier
		{"ubuntu:24.04-acme-other-built", "acme", "pi", ""},
		// Local names and bases without a tag aren't pulled
		{"ddev-pi-acme-built", "acme", "pi", ""},
		{"ddev-acme-custom-built", "acme", "custom", ""},
		{"ubuntu-acme-built", "acme", "pi", ""},
		{"localhost:5000/img-acme-pi-built", "acme", "pi", ""},
		{"Not A Reference:1-acme-built", "acme", "pi", ""},
		{"-acme-built", "acme", "pi", ""},
		{"ubuntu:24.04", "acme", "pi", ""},
		{"", "acme", "pi", ""},
		// DDEV's own web and db keep a custom untagged webimage or dbimage
		{"ddev/ddev-webserver:v1.25.4-acme-built", "acme", "web", "ddev/ddev-webserver:v1.25.4"},
		{"myorg/web-acme-built", "acme", "web", "myorg/web"},
		{"myorg/db-acme-built", "acme", "db", "myorg/db"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, baseImageFromBuiltTag(tc.image, tc.appName, tc.service), "image %q, service %q", tc.image, tc.service)
	}
}

func TestDescribeServiceImage(t *testing.T) {
	app := &DdevApp{Name: "acme"}
	project, err := dockerutil.CreateComposeProject(`
name: ddev-acme
services:
  web:
    image: ddev/ddev-webserver:v1-acme-built
    build:
      context: .
  qualified:
    image: busybox:1.36-acme-qualified-built
    build:
      context: .
  pulled:
    build:
      context: .
    x-ddev:
      pull-images:
        - busybox:1.36
        - alpine:3.20
  local:
    build:
      context: .
  plain:
    image: busybox:1.36
  worker:
    image: ddev/ddev-webserver:v1-acme-built
`)
	require.NoError(t, err)
	app.ComposeYaml = project

	require.Equal(t, "ddev/ddev-webserver:v1", app.describeServiceImage("web", "ddev/ddev-webserver:v1-acme-built"))
	require.Equal(t, "ddev/ddev-webserver:v1", app.describeServiceImage("web", ""))
	require.Equal(t, "busybox:1.36", app.describeServiceImage("qualified", ""))
	require.Equal(t, "ddev-acme-pulled", app.describeServiceImage("pulled", "ddev-acme-pulled"))
	require.Equal(t, "ddev-acme-local", app.describeServiceImage("local", ""))
	require.Equal(t, "ddev-acme-local", app.describeServiceImage("local", "ddev-acme-local"))
	require.Equal(t, "busybox:1.36", app.describeServiceImage("plain", "busybox:1.36"))
	require.Equal(t, "ddev/ddev-webserver:v1", app.describeServiceImage("worker", "ddev/ddev-webserver:v1-acme-built"))
	// A running container whose service is gone from the compose files
	require.Equal(t, "alpine:3.20", app.describeServiceImage("gone", "alpine:3.20-acme-built"))
	require.Equal(t, "ddev-acme-gone", app.describeServiceImage("gone", "ddev-acme-gone"))
}

func TestBuildTagCollisions(t *testing.T) {
	project, err := dockerutil.CreateComposeProject(`
name: ddev-acme
services:
  first:
    image: ubuntu:24.04-acme-built
    build:
      dockerfile_inline: FROM ubuntu:24.04
  second:
    image: ubuntu:24.04-acme-built
    profiles: [second]
    build:
      dockerfile_inline: FROM ubuntu:24.04
      args:
        SECOND: "1"
  no-image:
    build:
      dockerfile_inline: FROM ubuntu:24.04
  named-like-no-image:
    image: ddev-acme-no-image
    build:
      dockerfile_inline: FROM debian:12
  same-build-a:
    image: ubuntu:24.04-acme-same-built
    build:
      dockerfile_inline: FROM ubuntu:24.04
  same-build-b:
    image: ubuntu:24.04-acme-same-built
    build:
      dockerfile_inline: FROM ubuntu:24.04
  unique:
    image: ubuntu:24.04-acme-unique-built
    build:
      dockerfile_inline: FROM ubuntu:24.04
  reuses-first:
    image: ubuntu:24.04-acme-built
`)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		"ubuntu:24.04-acme-built": {"first", "second"},
		"ddev-acme-no-image":      {"named-like-no-image", "no-image"},
	}, buildTagCollisions(project))
}

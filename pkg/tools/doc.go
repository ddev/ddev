// Package tools exists because tools.go has a "//go:build tools" constraint.
// Without at least one unconstrained Go file in the package,
// "go test ./pkg/..." would fail with:
// "build constraints exclude all Go files"
package tools

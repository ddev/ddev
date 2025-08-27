package mkcert_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/mkcert"
)

func TestNewCA(t *testing.T) {
	ca := mkcert.NewCA()
	if ca == nil {
		t.Fatal("NewCA() returned nil")
	}

	caRoot := ca.GetCAROOT()
	if caRoot == "" {
		t.Fatal("GetCAROOT() returned empty string")
	}
}

func TestRunHostCommandCAROOT(t *testing.T) {
	output, err := mkcert.RunHostCommand("-CAROOT")
	if err != nil {
		t.Fatalf("RunHostCommand(-CAROOT) failed: %v", err)
	}

	if output == "" {
		t.Fatal("RunHostCommand(-CAROOT) returned empty string")
	}
}

func TestCreateCertificate(t *testing.T) {
	// Skip this test if we're in CI or similar environment where we can't create certs
	if os.Getenv("DDEV_TEST_NO_MKCERT") != "" {
		t.Skip("Skipping mkcert test in CI environment")
	}

	// Create a temporary directory for test certificates
	tmpDir, err := os.MkdirTemp("", "mkcert-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile := filepath.Join(tmpDir, "test.crt")
	keyFile := filepath.Join(tmpDir, "test.key")
	domains := []string{"test.example.com", "localhost"}

	output, err := mkcert.RunHostCommand("--cert-file", certFile, "--key-file", keyFile, domains[0], domains[1])
	if err != nil {
		t.Fatalf("RunHostCommand failed: %v", err)
	}

	if output == "" {
		t.Fatal("RunHostCommand returned empty output")
	}

	// Check if certificate and key files were created
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Fatalf("Certificate file was not created: %s", certFile)
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Fatalf("Key file was not created: %s", keyFile)
	}
}

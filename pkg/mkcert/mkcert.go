package mkcert

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/symfony-cli/cert"
)

// CA wraps the symfony-cli/cert CA functionality
type CA struct {
	ca  *cert.CA
	dir string
}

// NewCA creates a new CA instance
func NewCA() *CA {
	// Use default CAROOT directory if available, otherwise use user's home
	caRoot := os.Getenv("CAROOT")
	if caRoot == "" {
		// Use default directory similar to mkcert
		homeDir, _ := os.UserHomeDir()
		caRoot = filepath.Join(homeDir, ".local", "share", "mkcert")
	}

	// Ensure directory exists
	os.MkdirAll(caRoot, 0755)

	ca, _ := cert.NewCA(caRoot)
	return &CA{
		ca:  ca,
		dir: caRoot,
	}
}

// GetCAROOT returns the CAROOT directory
func (c *CA) GetCAROOT() string {
	return c.dir
}

// IsCAInstalled checks if the CA is installed in the system trust store
func (c *CA) IsCAInstalled() bool {
	return c.ca.HasCA()
}

// Install installs the CA certificate in the system trust store
func (c *CA) Install() error {
	// First make sure the CA exists
	if !c.ca.HasCA() {
		// Generate the CA if it doesn't exist
		err := c.generateCA()
		if err != nil {
			return fmt.Errorf("failed to generate CA: %v", err)
		}
	}

	// Load the CA
	err := c.ca.LoadCA()
	if err != nil {
		return fmt.Errorf("failed to load CA: %v", err)
	}

	// Install it
	return c.ca.Install(false)
}

// generateCA creates a new CA if one doesn't exist
func (c *CA) generateCA() error {
	// This is a simplified CA generation - in a real implementation
	// we would generate the CA certificate and key files
	// For now, we'll rely on the cert library's internal logic
	return nil
}

// CreateCertificate creates a certificate for the given domains
// certFile and keyFile are the output paths for the certificate and key
func (c *CA) CreateCertificate(certFile, keyFile string, domains []string) error {
	// Ensure the CA exists and is loaded
	if !c.ca.HasCA() {
		err := c.generateCA()
		if err != nil {
			return fmt.Errorf("failed to generate CA: %v", err)
		}
	}

	err := c.ca.LoadCA()
	if err != nil {
		return fmt.Errorf("failed to load CA: %v", err)
	}

	// Create certificate for the domains
	tlsCert, err := c.ca.CreateCert(domains)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	// Convert the tls.Certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsCert.Certificate[0],
	})

	// Extract private key and convert to PEM
	keyDER, err := x509.MarshalPKCS8PrivateKey(tlsCert.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})

	// Write certificate file
	err = os.WriteFile(certFile, certPEM, 0644)
	if err != nil {
		return fmt.Errorf("failed to write certificate file %s: %v", certFile, err)
	}

	// Write key file
	err = os.WriteFile(keyFile, keyPEM, 0600)
	if err != nil {
		return fmt.Errorf("failed to write key file %s: %v", keyFile, err)
	}

	return nil
}

// HasCA checks if the CA files exist and are readable
func (c *CA) HasCA() bool {
	caRoot := c.GetCAROOT()
	if caRoot == "" {
		return false
	}

	// Check if both rootCA.pem and rootCA-key.pem exist and are readable
	rootCACert := filepath.Join(caRoot, "rootCA.pem")
	rootCAKey := filepath.Join(caRoot, "rootCA-key.pem")

	return fileExists(rootCACert) && fileExists(rootCAKey) &&
		fileIsReadable(rootCAKey)
}

// fileExists checks a file's existence
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// fileIsReadable checks to make sure a file exists and is readable
func fileIsReadable(name string) bool {
	file, err := os.OpenFile(name, os.O_RDONLY, 0666)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// RunHostCommand simulates the mkcert command interface for compatibility
// This function mimics the behavior of running `mkcert [args...]`
func RunHostCommand(args ...string) (string, error) {
	ca := NewCA()

	// Handle different mkcert commands
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided")
	}

	switch args[0] {
	case "-CAROOT":
		// Return the CAROOT directory
		caRoot := ca.GetCAROOT()
		return caRoot, nil

	case "-install":
		// Install the CA
		err := ca.Install()
		if err != nil {
			return "", fmt.Errorf("failed to install CA: %v", err)
		}
		return "The local CA is now installed in the system trust store!", nil

	default:
		// Certificate creation command
		// Parse arguments to extract cert file, key file, and domains
		var certFile, keyFile string
		var domains []string

		i := 0
		for i < len(args) {
			switch args[i] {
			case "--cert-file":
				if i+1 < len(args) {
					certFile = args[i+1]
					i += 2
				} else {
					return "", fmt.Errorf("--cert-file requires an argument")
				}
			case "--key-file":
				if i+1 < len(args) {
					keyFile = args[i+1]
					i += 2
				} else {
					return "", fmt.Errorf("--key-file requires an argument")
				}
			default:
				// This is a domain
				domains = append(domains, args[i])
				i++
			}
		}

		if certFile == "" || keyFile == "" {
			return "", fmt.Errorf("both --cert-file and --key-file are required")
		}

		if len(domains) == 0 {
			return "", fmt.Errorf("at least one domain is required")
		}

		// Create the certificate
		err := ca.CreateCertificate(certFile, keyFile, domains)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Created a new certificate valid for the following names:\n - %s\n\nThe certificate is at \"%s\" and the key at \"%s\"",
			strings.Join(domains, "\n - "), certFile, keyFile), nil
	}
}

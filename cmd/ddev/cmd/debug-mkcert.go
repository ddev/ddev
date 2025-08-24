package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/mkcert"
	"github.com/spf13/cobra"
)

// DebugMkcertCmd implements the ddev debug mkcert command
var DebugMkcertCmd = &cobra.Command{
	Use:     "mkcert",
	Short:   "Show status of mkcert certificate authority and certificates",
	Long:    `Show status of DDEV's integrated mkcert certificate authority and certificates.`,
	Example: "ddev debug mkcert",
	Run: func(cmd *cobra.Command, args []string) {
		ca := mkcert.NewCA()

		fmt.Println("DDEV Integrated mkcert Status:")
		fmt.Println("==============================")

		// Show CA installation status
		if ca.IsCAInstalled() {
			fmt.Println("✓ Certificate Authority (CA) is installed in system trust store")
		} else {
			fmt.Println("✗ Certificate Authority (CA) is NOT installed in system trust store")
		}

		// Show CAROOT location
		caroot := ca.GetCAROOT()
		fmt.Printf("CAROOT location: %s\n", caroot)

		// Check if CA files exist
		rootCAPem := filepath.Join(caroot, "rootCA.pem")
		rootCAKey := filepath.Join(caroot, "rootCA-key.pem")

		if _, err := os.Stat(rootCAPem); err == nil {
			fmt.Println("✓ Root CA certificate file exists")
		} else {
			fmt.Println("✗ Root CA certificate file missing")
		}

		if _, err := os.Stat(rootCAKey); err == nil {
			fmt.Println("✓ Root CA private key file exists")
		} else {
			fmt.Println("✗ Root CA private key file missing")
		}

		// Show certificate files in CAROOT
		fmt.Println("\nCertificate files in CAROOT:")
		if entries, err := os.ReadDir(caroot); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					if strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".crt") || strings.HasSuffix(name, ".key") {
						fmt.Printf("  - %s\n", name)
					}
				}
			}
		} else {
			fmt.Printf("  Error reading CAROOT directory: %v\n", err)
		}

		// Show example usage
		fmt.Println("\nNote:")
		fmt.Println("  Certificate creation is handled automatically by DDEV during project startup.")
		fmt.Println("  This debug command only shows the status of the integrated mkcert CA.")
	},
}

func init() {
	DebugCmd.AddCommand(DebugMkcertCmd)
}

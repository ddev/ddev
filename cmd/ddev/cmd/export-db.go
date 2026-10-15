package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// exportCompressionTypes are the values accepted by `ddev export-db --compression`.
var exportCompressionTypes = []string{"gzip", "zstd", "xz", "bzip2", "none"}

// NewExportDBCmd initializes and returns the `ddev export-db` command.
func NewExportDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-db [project]",
		Short: "Dump a database to a file or to stdout",
		Long: heredoc.Doc(`
			Dump a database to a file or to stdout.

			When --file is given without --compression, the compression type is
			chosen from the file extension: .gz, .zst, .xz, .bz2, or .sql/.mysql
			for no compression. Otherwise gzip is used.
		`),
		Example: heredoc.DocI2S(`
			$ ddev export-db --file=/tmp/db.sql.gz
			$ ddev export-db -f /tmp/db.sql.zst
			$ ddev export-db --file /tmp/db.sql
			$ ddev export-db > /tmp/db.sql.gz
			$ ddev export-db --compression=none > /tmp/db.sql
			$ ddev export-db --compression=zstd > /tmp/db.sql.zst
			$ ddev export-db --database=additional_db --file=.tarballs/additional_db.sql.gz
			$ ddev export-db my-project --file=/tmp/my_project.sql
		`),
		Args: cobra.RangeArgs(0, 1),
		PreRun: func(_ *cobra.Command, _ []string) {
			dockerutil.EnsureDdevNetwork()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := getRequestedProjects(args, false)
			if err != nil {
				return fmt.Errorf("unable to get project: %v", err)
			}

			app := projects[0]

			dumpFile, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}

			database, err := cmd.Flags().GetString("database")
			if err != nil {
				return err
			}

			if !cmd.Flags().Lookup("database").Changed && cmd.Flags().Lookup("target-db").Changed {
				database, err = cmd.Flags().GetString("target-db")
				if err != nil {
					return err
				}
			}

			compressionType, err := resolveExportCompression(cmd.Flags(), dumpFile)
			if err != nil {
				return err
			}

			return exportDBRun(app, dumpFile, database, compressionType)
		},
	}

	cmd.Flags().StringP("file", "f", "", "Path to a SQL dump file to export to")
	cmd.Flags().StringP("database", "d", "db", "Target database to export from")
	cmd.Flags().String("compression", "", "Compression type: gzip, zstd, xz, bzip2, or none (default from --file extension, otherwise gzip)")
	_ = cmd.RegisterFlagCompletionFunc("compression", configCompletionFunc(exportCompressionTypes))

	// Backward compatibility
	cmd.Flags().String("target-db", "db", cmd.Flags().Lookup("database").Usage)
	_ = cmd.Flags().MarkDeprecated("target-db", "please use --database instead")

	cmd.Flags().BoolP("gzip", "z", true, "Use gzip compression")
	cmd.Flags().Bool("xz", false, "Use xz compression")
	cmd.Flags().Bool("bzip2", false, "Use bzip2 compression")
	for _, f := range []string{"gzip", "xz", "bzip2"} {
		_ = cmd.Flags().MarkHidden(f)
	}
	_ = cmd.Flags().MarkShorthandDeprecated("z", "please use --compression=gzip instead")

	return cmd
}

// resolveExportCompression picks the compression type from --compression,
// the legacy --gzip/--xz/--bzip2 flags, or the --file extension, in that order.
// --gzip=false alongside --xz or --bzip2 is an old idiom, so it only means
// "none" when no other legacy flag is given.
func resolveExportCompression(flags *pflag.FlagSet, dumpFile string) (string, error) {
	var legacy []string
	gzipFalse := false
	for _, name := range []string{"gzip", "xz", "bzip2"} {
		if !flags.Changed(name) {
			continue
		}
		v, err := flags.GetBool(name)
		if err != nil {
			return "", err
		}
		switch {
		case v:
			legacy = append(legacy, name)
		case name == "gzip":
			gzipFalse = true
		}
	}
	if len(legacy) > 1 {
		return "", fmt.Errorf("only one of --%s may be used, use --compression instead", strings.Join(legacy, ", --"))
	}

	if flags.Changed("compression") {
		compression, err := flags.GetString("compression")
		if err != nil {
			return "", err
		}
		if !slices.Contains(exportCompressionTypes, compression) {
			return "", fmt.Errorf("invalid --compression=%s, must be one of %s", compression, strings.Join(exportCompressionTypes, ", "))
		}
		if (len(legacy) == 1 && legacy[0] != compression) || (gzipFalse && compression == "gzip") {
			return "", fmt.Errorf("--compression=%s conflicts with --gzip, --xz, or --bzip2", compression)
		}
		return compression, nil
	}
	if len(legacy) == 1 {
		return legacy[0], nil
	}
	if gzipFalse {
		return "none", nil
	}

	if dumpFile != "" {
		lower := strings.ToLower(dumpFile)
		switch {
		case strings.HasSuffix(lower, ".gz"):
			return "gzip", nil
		case strings.HasSuffix(lower, ".zst"):
			return "zstd", nil
		case strings.HasSuffix(lower, ".xz"):
			return "xz", nil
		case strings.HasSuffix(lower, ".bz2"):
			return "bzip2", nil
		case strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".mysql"):
			return "none", nil
		}
	}
	return "gzip", nil
}

func registerExportDBCmd() {
	// TODO: Move to RootCmd
	RootCmd.AddCommand(NewExportDBCmd())
}

func exportDBRun(app *ddevapp.DdevApp, dumpFile, database, compressionType string) error {
	status, _ := app.SiteStatus()
	if status != ddevapp.SiteRunning {
		err := app.Start()
		if err != nil {
			return fmt.Errorf("failed to start app %s to import-db: %v", app.Name, err)
		}
	}

	err := app.ExportDB(dumpFile, compressionType, database)
	if err != nil {
		return fmt.Errorf("failed to export database for %s: %v", app.GetName(), err)
	}

	return nil
}

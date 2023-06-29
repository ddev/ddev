package cmd

import (
	"fmt"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/spf13/cobra"
)

// NewExportDBCmd initializes and returns the `ddev export-db` command.
func NewExportDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-db [project]",
		Short: "Dump a database to a file or to stdout",
		Long:  `Dump a database to a file or to stdout.`,
		Example: heredoc.DocI2S(`
			$ ddev export-db --file=/tmp/db.sql.gz
			$ ddev export-db -f /tmp/db.sql.gz
			$ ddev export-db --gzip=false --file /tmp/db.sql
			$ ddev export-db > /tmp/db.sql.gz
			$ ddev export-db --gzip=false > /tmp/db.sql
			$ ddev export-db --database=additional_db --file=.tarballs/additional_db.sql.gz
			$ ddev export-db my-project --gzip=false --file=/tmp/my_project.sql
		`),
		Args: cobra.RangeArgs(0, 1),
		PreRun: func(cmd *cobra.Command, args []string) {
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

			compXz, err := cmd.Flags().GetBool("xz")
			if err != nil {
				return err
			}

			compBzip2, err := cmd.Flags().GetBool("bzip2")
			if err != nil {
				return err
			}

			compGzip, err := cmd.Flags().GetBool("gzip")
			if err != nil {
				return err
			}

			compressionType := ""

			switch {
			case compXz:
				compressionType = "xz"
			case compBzip2:
				compressionType = "bzip2"
			case compGzip:
				compressionType = "gzip"
			}

			return exportDBRun(app, dumpFile, database, compressionType)
		},
	}

	cmd.Flags().StringP("file", "f", "", "Path to a SQL dump file to export to")
	cmd.Flags().StringP("database", "d", "db", "Target database to export from")
	cmd.Flags().BoolP("gzip", "z", true, "Use gzip compression")
	cmd.Flags().Bool("xz", false, "Use xz compression")
	cmd.Flags().Bool("bzip2", false, "Use bzip2 compression")

	// Backward compatibility
	cmd.Flags().String("target-db", "db", cmd.Flags().Lookup("database").Usage)
	_ = cmd.Flags().MarkDeprecated("target-db", "please use --database instead")

	_ = cmd.Flags().MarkShorthandDeprecated("z", "please use --gzip instead")

	return cmd
}

func init() {
	// TODO move to RootCmd
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

package cmd

import (
	"fmt"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// NewImportDBCmd initializes and returns the `ddev import-db` command.
func NewImportDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-db [project]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "Import a SQL dump file into the project",
		Long: heredoc.Doc(`
			Import a SQL dump file into the project.
			
			The database dump file can be provided as a SQL dump in a .sql, .sql.gz,
			sql.bz2, sql.xz, .mysql, .mysql.gz, .zip, .tgz, or .tar.gz format.
			
			For the zip and tar formats, the path to a .sql file within the archive
			can be provided if it is not located at the top level of the archive.
			
			An optional target database can also be provided; the default is the
			default database named "db".

			Also note the related "ddev mysql" command.
		`),
		Example: heredoc.DocI2S(`
			$ ddev import-db
			$ ddev import-db --file=.tarballs/junk.sql
			$ ddev import-db --file=.tarballs/junk.sql.gz
			$ ddev import-db --database=other_db --file=.tarballs/db.sql.gz
			$ ddev import-db --file=.tarballs/db.sql.bz2
			$ ddev import-db --file=.tarballs/db.sql.xz
			$ ddev import-db < db.sql
			$ ddev import-db my-project < db.sql
			$ gzip -dc db.sql.gz | ddev import-db
		`),
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

			if !cmd.Flags().Lookup("file").Changed && cmd.Flags().Lookup("src").Changed {
				dumpFile, err = cmd.Flags().GetString("src")
				if err != nil {
					return err
				}
			}

			extractPath, err := cmd.Flags().GetString("extract-path")
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

			noDrop, err := cmd.Flags().GetBool("no-drop")
			if err != nil {
				return err
			}

			noProgress, err := cmd.Flags().GetBool("no-progress")
			if err != nil {
				return err
			}

			if !cmd.Flags().Lookup("no-progress").Changed && cmd.Flags().Lookup("progress").Changed {
				progress, err := cmd.Flags().GetBool("progress")
				if err != nil {
					return err
				}

				noProgress = !progress
			}

			return importDBRun(app, dumpFile, extractPath, database, noDrop, noProgress)
		},
	}

	cmd.Flags().StringP("file", "f", "", "Path to a SQL dump in `.sql`, `.tar`, `.tar.gz`, `.tar.bz2`, `.tar.xz`, `.tgz`, or `.zip` format")
	cmd.Flags().String("extract-path", "", "Path to extract within the archive")
	cmd.Flags().StringP("database", "d", "db", "Target database to import into")
	cmd.Flags().Bool("no-drop", false, "Do not drop the database before importing")
	cmd.Flags().Bool("no-progress", false, "Do not output progress")

	// Backward compatibility
	cmd.Flags().String("src", "", cmd.Flags().Lookup("file").Usage)
	_ = cmd.Flags().MarkDeprecated("src", "please use --file instead")

	cmd.Flags().String("target-db", "db", cmd.Flags().Lookup("database").Usage)
	_ = cmd.Flags().MarkDeprecated("target-db", "please use --database instead")

	cmd.Flags().BoolP("progress", "p", true, "Display a progress bar during import")
	_ = cmd.Flags().MarkDeprecated("progress", "please use --no-progress instead")
	_ = cmd.Flags().MarkShorthandDeprecated("p", "please use --no-progress instead")

	return cmd
}

func init() {
	// TODO move to RootCmd
	RootCmd.AddCommand(NewImportDBCmd())
}

func importDBRun(app *ddevapp.DdevApp, dumpFile, extractPath, database string, noDrop, noProgress bool) error {
	status, _ := app.SiteStatus()

	if status != ddevapp.SiteRunning {
		err := app.Start()
		if err != nil {
			return fmt.Errorf("failed to start app %s to import-db: %v", app.Name, err)
		}
	}

	err := app.ImportDB(dumpFile, extractPath, !noProgress, noDrop, database)
	if err != nil {
		return fmt.Errorf("failed to import database '%s' for %s: %v", database, app.GetName(), err)
	}

	noDropInfo := ""
	if noDrop {
		noDropInfo = ", existing database was NOT dropped before importing"
	}

	util.Success("Successfully imported database '%s' for %s%s", database, app.GetName(), noDropInfo)

	return nil
}

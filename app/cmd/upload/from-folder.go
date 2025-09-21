package upload

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

func NewFromFolderCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-folder [flags] <path>...",
		Short: "Upload photos from a folder",
		Args: func(cmd *cobra.Command, args []string) error {
			// Check if only-update-metadata flag is set
			if cmd.Flags().Lookup("only-update-metadata").Changed {
				if len(args) == 0 {
					return errors.New("when using --only-update-metadata, you must specify the path(s) to your files for asset matching (e.g., '/path/to/folder')")
			}
			}
			return cobra.MinimumNArgs(1)(cmd, args)
		},
	}
	cmd.SetContext(ctx)
	options := &folder.ImportFolderOptions{}
	options.AddFromFolderFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()
		options.TZ = app.GetTZ()

		// parse arguments
		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		// create the adapter for folders
		options.SupportedMedia = client.Immich.SupportedMedia()
		upOptions.Filters = append(upOptions.Filters, options.ManageBurst.GroupFilter(), options.ManageRawJPG.GroupFilter(), options.ManageHEICJPG.GroupFilter())

		options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
		adapter, err := folder.NewLocalFiles(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}

		return newUpload(UpModeFolder, app, upOptions).run(ctx, adapter, app, fsyss)
	}

	return cmd
}

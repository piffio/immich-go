package upload

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

var _re3digits = regexp.MustCompile(`-\d{3}$`)

func NewFromGooglePhotosCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Upload photos either from a zipped Google Photos takeout or decompressed archive",
		Args: func(cmd *cobra.Command, args []string) error {
			// Check if only-update-metadata flag is set
		if cmd.Flags().Lookup("only-update-metadata").Changed {
				if len(args) == 0 {
					return errors.New("when using --only-update-metadata, you must specify the path(s) to your files for asset matching (e.g., '/path/to/takeout-*.zip' or '/path/to/folder')")
				}
			}
			return cobra.MinimumNArgs(1)(cmd, args)
		},
	}
	cmd.SetContext(ctx)
	options := &gp.ImportFlags{}
	options.AddFromGooglePhotosFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()

		options.TZ = app.GetTZ()

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		if options.TakeoutTag {
			for _, fsys := range fsyss {
				if fsys, ok := fsys.(fshelper.NameFS); ok {
					options.TakeoutName = fsys.Name()
					break
				}
			}

			if filepath.Ext(options.TakeoutName) == ".zip" {
				options.TakeoutName = strings.TrimSuffix(options.TakeoutName, filepath.Base(options.TakeoutName))
			}
			if options.TakeoutName == "" {
				options.TakeoutTag = false
			}
			options.TakeoutName = _re3digits.ReplaceAllString(options.TakeoutName, "")
		}

		upOptions.Filters = append(upOptions.Filters, options.ManageBurst.GroupFilter(), options.ManageRawJPG.GroupFilter(), options.ManageHEICJPG.GroupFilter())

		options.SupportedMedia = client.Immich.SupportedMedia()
		options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
		adapter, err := gp.NewTakeout(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}
		err = newUpload(UpModeGoogleTakeout, app, upOptions).setTakeoutOptions(options).run(ctx, adapter, app, fsyss)
		return err
	}

	return cmd
}

package upload

import (
	"context"
	"errors"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/spf13/cobra"
)

type UpLoadMode int

const (
	UpModeGoogleTakeout UpLoadMode = iota
	UpModeFolder
	UpModeICloud
	UpModePicasa
)

func (m UpLoadMode) String() string {
	switch m {
	case UpModeGoogleTakeout:
		return "Google Takeout"
	case UpModeFolder:
		return "Folder"
	case UpModeICloud:
		return "iCloud"
	case UpModePicasa:
		return "Picasa"
	default:
		return "Unknown"
	}
}

// UploadOptions represents a set of common flags used for filtering assets.
type UploadOptions struct {
	// TODO place this option at the top
	NoUI bool // Disable UI

	// Add Overwrite flag to UploadOptions
	Overwrite bool // Always overwrite files on the server with local versions

	// OnlyUpdateMetadata allows updating metadata without re-uploading files.
	OnlyUpdateMetadata bool

	Filters []filters.Filter
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, a *app.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload photos to an Immich server from various sources",
	}
	app.AddClientFlags(ctx, cmd, a, false)
	cmd.TraverseChildren = true
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentFlags().BoolVar(&options.Overwrite, "overwrite", false, "Always overwrite files on the server with local versions")
	cmd.PersistentFlags().BoolVar(&options.OnlyUpdateMetadata, "only-update-metadata", false, "Update metadata for existing assets without re-uploading files (Workaround for Immich #16747)")
	cmd.PersistentPreRunE = app.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, a)

	cmd.AddCommand(NewFromFolderCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromICloudCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromPicasaCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromGooglePhotosCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromImmichCommand(ctx, cmd, a, options))
	return cmd
}

func (options *UploadOptions) Open(ctx context.Context, cmd *cobra.Command, app *app.Application) error {
	// Initialize the Journal
	if app.Jnl() == nil {
		app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
	}

	if options.OnlyUpdateMetadata {
		if options.Overwrite {
			return errors.New("the --only-update-metadata flag cannot be used with --overwrite")
		}

		// Check if at least one metadata flag is set
		metadataFlags := []string{"tag", "session-tag", "into-album", "folder-as-album"}
		flagSet := false
		for _, flagName := range metadataFlags {
			if cmd.Flags().Changed(flagName) {
				flagSet = true
				break
			}
		}

		if !flagSet {
			return errors.New("the --only-update-metadata flag requires at least one metadata flag to be set (e.g., --tag, --into-album)")
		}
	}

	app.SetTZ(time.Local)
	if tz, err := cmd.Flags().GetString("time-zone"); err == nil && tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			app.SetTZ(loc)
		}
	}
	return nil
}

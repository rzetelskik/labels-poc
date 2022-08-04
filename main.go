package main

import (
	"context"
	"fmt"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/scylladb/go-log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var imageReference string

func main() {
	ctx := log.WithNewTraceID(context.Background())
	atom := zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	logger, _ := log.NewProduction(log.Config{
		Level: atom,
	})

	var rootCmd = &cobra.Command{}
	rootCmd.AddCommand(
		newInspectCmd(ctx, logger),
	)

	rootCmd.Execute()
}

func newInspectCmd(ctx context.Context, logger log.Logger) *cobra.Command {
	runFirstApproach := func(cmd *cobra.Command, args []string) (globalErr error) {
		transport := docker.Transport

		logger.Debug(ctx, "Parsing image reference")
		ref, err := transport.ParseReference(fmt.Sprintf("//%s", imageReference))
		if err != nil {
			return fmt.Errorf("can't parse image reference: %w", err)
		}

		sysCtx := &types.SystemContext{}

		logger.Debug(ctx, "Creating new ImageSource")
		src, err := ref.NewImageSource(ctx, sysCtx)
		if err != nil {
			return fmt.Errorf("can't get new image source: %w", err)
		}

		defer func() {
			err := src.Close()
			if err != nil {
				globalErr = errors.Wrap(err, "Could not close ImageSource")
			}
		}()

		logger.Debug(ctx, "Getting image from unparsed")
		img, err := image.FromUnparsedImage(ctx, sysCtx, image.UnparsedInstance(src, nil))
		if err != nil {
			return fmt.Errorf("can't read unparsed image: %w", err)
		}

		logger.Debug(ctx, "Inspecting image")
		inspect, err := img.Inspect(ctx)
		if err != nil {
			return fmt.Errorf("can't inspect image: %w", err)
		}

		logger.Debug(ctx, "Image inspected")
		version, ok := inspect.Labels["version"]
		if !ok {
			return errors.New("No version label found")
		}

		fmt.Printf("Version: %s\n", version)

		return nil
	}

	cmd := &cobra.Command{
		Use:           "inspect imageReference",
		Short:         "Inspect scylla image version.",
		SilenceErrors: false,
		SilenceUsage:  false,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("image reference missing")
			}

			imageReference = args[0]

			return nil
		},

		RunE: runFirstApproach,
	}

	return cmd
}

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aweris/cafs"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <ref> [tags...]",
	Short: "Push to remote registry",
	Long:  "Push local namespace data to an OCI registry. Optionally push to additional tags.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) (err error) {
	ref := args[0]
	tags := args[1:]

	fs, err := cafs.Open(ref, cafs.WithCacheDir(getCacheDir()))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fs.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	fmt.Fprintf(os.Stderr, "Pushing %s...\n", ref)

	if err := fs.Push(context.Background(), tags...); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Done. Root: %s\n", fs.Root())
	return nil
}

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aweris/cafs"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <ref>",
	Short: "Pull from remote registry",
	Long:  "Pull namespace data from an OCI registry to local cache.",
	Args:  cobra.ExactArgs(1),
	RunE:  runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) (err error) {
	ref := args[0]

	fs, err := cafs.Open(ref, cafs.WithCacheDir(getCacheDir()))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fs.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	fmt.Fprintf(os.Stderr, "Pulling %s...\n", ref)

	if err := fs.Pull(context.Background()); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Done. Root: %s\n", fs.Root())
	return nil
}

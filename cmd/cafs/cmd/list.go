package cmd

import (
	"fmt"

	"github.com/aweris/cafs"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list <ref> [prefix]",
	Short: "List entries in namespace",
	Long:  "List all entries in a namespace, optionally filtered by prefix.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) (err error) {
	ref := args[0]
	prefix := ""
	if len(args) > 1 {
		prefix = args[1]
	}

	fs, err := cafs.Open(ref, cafs.WithCacheDir(getCacheDir()))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fs.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	count := 0
	for key, digest := range fs.Index().List(prefix) {
		fmt.Printf("%s\t%s\n", key, digest)
		count++
	}

	if count == 0 {
		fmt.Println("(no entries)")
	}

	return nil
}

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(),
				"uploadcare-cli %s\ncommit: %s\nbuilt:  %s\ngo:     %s\nos/arch: %s/%s\n",
				version, commit, date,
				runtime.Version(),
				runtime.GOOS, runtime.GOARCH,
			)
		},
	}
}

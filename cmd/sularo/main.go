package main

import (
	"fmt"
	"os"

	"github.com/sawood14012/sularo/internal/test"
	"github.com/spf13/cobra"
)

func main() {
	var verbose bool

	root := &cobra.Command{
		Use:   "sularo",
		Short: "sularo is a tiny test harness for Crossplane compositions",
	}

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Run composition render tests under ./tests/",
		RunE: func(cmd *cobra.Command, args []string) error {
			return test.Run("./tests", verbose, os.Stdout)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	root.AddCommand(testCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

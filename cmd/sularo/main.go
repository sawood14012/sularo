package main

import (
	"fmt"
	"os"

	"github.com/sawood14012/sularo/internal/test"
	"github.com/sawood14012/sularo/internal/test/format"
	"github.com/spf13/cobra"
)

func main() {
	var verbose bool
	var outputFormat string
	var filter string

	root := &cobra.Command{
		Use:   "sularo",
		Short: "sularo is a tiny test harness for Crossplane compositions",
	}

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Run composition render tests under ./tests/",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := test.Run("./tests", filter)
			if err != nil {
				return err
			}

			var f format.Formatter
			switch outputFormat {
			case "junit":
				f = format.JUnit{}
			case "json":
				f = format.JSON{}
			default:
				f = format.TAP{Verbose: verbose}
			}
			f.Write(os.Stdout, results)

			for _, r := range results {
				if r.Status == test.StatusFail {
					return fmt.Errorf("tests failed")
				}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	testCmd.Flags().StringVar(&outputFormat, "format", "tap", "Output format: tap, junit, json")
	testCmd.Flags().StringVar(&filter, "filter", "", "Run only test cases whose name contains this substring")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Re-run render and overwrite expected.yaml for each test case",
		RunE: func(cmd *cobra.Command, args []string) error {
			return test.Update("./tests", filter, os.Stdout)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	updateCmd.Flags().StringVar(&filter, "filter", "", "Update only test cases whose name contains this substring")

	root.AddCommand(testCmd, updateCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

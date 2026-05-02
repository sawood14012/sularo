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

	var initXR, initComposition, initFunctions string
	initCmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Scaffold a new test case under ./tests/",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return test.Init(test.InitOptions{
				Name:        args[0],
				XRSource:    initXR,
				Composition: initComposition,
				Functions:   initFunctions,
			})
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	initCmd.Flags().StringVar(&initXR, "xr", "", "Path to source XR file (required)")
	initCmd.Flags().StringVar(&initComposition, "composition", "", "Repo-relative path to composition (required)")
	initCmd.Flags().StringVar(&initFunctions, "functions", "", "Repo-relative path to functions file (optional)")
	_ = initCmd.MarkFlagRequired("xr")
	_ = initCmd.MarkFlagRequired("composition")

	root.AddCommand(testCmd, updateCmd, initCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

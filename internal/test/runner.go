package test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sawood14012/sularo/internal/test/schema"
)

func Run(root, filter string) ([]Result, error) {
	reg, err := schema.LoadDir("./crds")
	if err != nil {
		return nil, fmt.Errorf("load crds: %w", err)
	}

	cases, err := Discover(root)
	if err != nil {
		return nil, err
	}
	cases = applyFilter(cases, filter)
	results := make([]Result, 0, len(cases))
	for _, c := range cases {
		results = append(results, runCase(c, reg))
	}
	return results, nil
}

func runCase(c Case, reg *schema.Registry) Result {
	if c.Skip {
		return Result{Name: c.Name, Status: StatusSkip}
	}

	start := time.Now()

	rendered, err := render(c)
	if err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start), Message: err.Error()}
	}

	// Schema validation — runs only when ./crds/ exists and has matching CRDs.
	if !reg.Empty() {
		docs, err := parseDocs(rendered)
		if err == nil {
			var resources []map[string]any
			for _, d := range docs {
				if m, ok := d.(map[string]any); ok {
					resources = append(resources, m)
				}
			}
			if errs := reg.Validate(resources); len(errs) > 0 {
				return Result{
					Name:     c.Name,
					Status:   StatusFail,
					Duration: time.Since(start),
					Message:  "schema validation failed:\n  " + strings.Join(errs, "\n  "),
				}
			}
		}
	}

	expected, err := os.ReadFile(c.Expected)
	if err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start),
			Message: fmt.Sprintf("read expected: %v", err)}
	}

	diff, err := Diff(rendered, expected)
	if err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start), Message: err.Error()}
	}
	if diff != "" {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start),
			Message: "--- expected\n+++ got\n" + diff}
	}
	return Result{Name: c.Name, Status: StatusPass, Duration: time.Since(start)}
}

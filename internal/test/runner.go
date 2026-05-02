package test

import (
	"fmt"
	"os"
	"time"
)

func Run(root, filter string) ([]Result, error) {
	cases, err := Discover(root)
	if err != nil {
		return nil, err
	}
	cases = applyFilter(cases, filter)
	results := make([]Result, 0, len(cases))
	for _, c := range cases {
		results = append(results, runCase(c))
	}
	return results, nil
}

func runCase(c Case) Result {
	if c.Skip {
		return Result{Name: c.Name, Status: StatusSkip}
	}

	start := time.Now()

	rendered, err := render(c)
	if err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start), Message: err.Error()}
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

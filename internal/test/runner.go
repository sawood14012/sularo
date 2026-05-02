package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func Run(root string) ([]Result, error) {
	cases, err := Discover(root)
	if err != nil {
		return nil, err
	}
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

	for _, p := range []string{c.Composition, c.XR, c.Expected} {
		if _, err := os.Stat(p); err != nil {
			return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start),
				Message: fmt.Sprintf("missing file: %s", p)}
		}
	}

	cmd := exec.Command("crossplane", "render", c.Composition, c.XR)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start),
			Message: fmt.Sprintf("crossplane render failed: %v\n%s", err, stderr.String())}
	}

	expected, err := os.ReadFile(c.Expected)
	if err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start),
			Message: fmt.Sprintf("read expected: %v", err)}
	}

	diff, err := Diff(stdout.Bytes(), expected)
	if err != nil {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start), Message: err.Error()}
	}
	if diff != "" {
		return Result{Name: c.Name, Status: StatusFail, Duration: time.Since(start),
			Message: "--- expected\n+++ got\n" + diff}
	}
	return Result{Name: c.Name, Status: StatusPass, Duration: time.Since(start)}
}

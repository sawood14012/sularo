package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func Run(root string, verbose bool, out io.Writer) error {
	cases, err := Discover(root)
	if err != nil {
		return err
	}

	if len(cases) == 0 {
		fmt.Fprintln(out, "1..0 # no test cases found")
		return nil
	}

	fmt.Fprintf(out, "1..%d\n", len(cases))

	failed := 0
	for i, c := range cases {
		n := i + 1
		ok, msg := runCase(c, verbose)
		if ok {
			fmt.Fprintf(out, "ok %d - %s\n", n, c.Name)
			if verbose && msg != "" {
				fmt.Fprintln(out, Indent(msg, "# "))
			}
		} else {
			failed++
			fmt.Fprintf(out, "not ok %d - %s\n", n, c.Name)
			if msg != "" {
				fmt.Fprintln(out, Indent(msg, "    "))
			}
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d/%d test(s) failed", failed, len(cases))
	}
	return nil
}

func runCase(c Case, verbose bool) (bool, string) {
	for _, p := range []string{c.Composition, c.XR, c.Expected} {
		if _, err := os.Stat(p); err != nil {
			return false, fmt.Sprintf("missing file: %s", p)
		}
	}

	cmd := exec.Command("crossplane", "render", c.Composition, c.XR)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Sprintf("crossplane render failed: %v\n%s", err, stderr.String())
	}

	expected, err := os.ReadFile(c.Expected)
	if err != nil {
		return false, fmt.Sprintf("read expected: %v", err)
	}

	diff, err := Diff(stdout.Bytes(), expected)
	if err != nil {
		return false, err.Error()
	}
	if diff != "" {
		return false, "--- expected\n+++ got\n" + diff
	}
	return true, ""
}

package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func Update(root, filter string, out io.Writer) error {
	cases, err := Discover(root)
	if err != nil {
		return err
	}
	cases = applyFilter(cases, filter)

	if len(cases) == 0 {
		fmt.Fprintln(out, "no test cases found")
		return nil
	}

	var updated, skipped int
	for _, c := range cases {
		if c.Skip {
			fmt.Fprintf(out, "skip  %s\n", c.Name)
			skipped++
			continue
		}

		rendered, err := render(c)
		if err != nil {
			return fmt.Errorf("%s: %w", c.Name, err)
		}

		if err := os.WriteFile(c.Expected, rendered, 0644); err != nil {
			return fmt.Errorf("%s: write expected: %w", c.Name, err)
		}

		fmt.Fprintf(out, "wrote %s\n", c.Expected)
		updated++
	}

	fmt.Fprintf(out, "\n%d updated", updated)
	if skipped > 0 {
		fmt.Fprintf(out, ", %d skipped", skipped)
	}
	fmt.Fprintln(out)
	return nil
}

func render(c Case) ([]byte, error) {
	for _, p := range []string{c.Composition, c.XR} {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("missing file: %s", p)
		}
	}

	cmd := exec.Command("crossplane", "render", c.Composition, c.XR)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("crossplane render: %v\n%s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

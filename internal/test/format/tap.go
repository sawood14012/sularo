package format

import (
	"fmt"
	"io"

	"github.com/sawood14012/sularo/internal/test"
)

type TAP struct {
	Verbose bool
}

func (t TAP) Write(w io.Writer, results []test.Result) {
	fmt.Fprintf(w, "1..%d\n", len(results))
	for i, r := range results {
		n := i + 1
		switch r.Status {
		case test.StatusSkip:
			fmt.Fprintf(w, "ok %d - %s # SKIP\n", n, r.Name)
		case test.StatusPass:
			fmt.Fprintf(w, "ok %d - %s\n", n, r.Name)
			if t.Verbose && r.Message != "" {
				fmt.Fprintln(w, test.Indent(r.Message, "# "))
			}
		case test.StatusFail:
			fmt.Fprintf(w, "not ok %d - %s\n", n, r.Name)
			if r.Message != "" {
				fmt.Fprintln(w, test.Indent(r.Message, "    "))
			}
		}
	}
}

package test

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func parseDocs(data []byte) ([]any, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []any
	for {
		var d any
		err := dec.Decode(&d)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if d == nil {
			continue
		}
		docs = append(docs, normalize(d))
	}
	return docs, nil
}

func normalize(v any) any {
	switch x := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(x))
		for k, vv := range x {
			m[fmt.Sprint(k)] = normalize(vv)
		}
		return m
	case map[string]any:
		for k, vv := range x {
			x[k] = normalize(vv)
		}
		return x
	case []any:
		for i, vv := range x {
			x[i] = normalize(vv)
		}
		return x
	}
	return v
}

// projectSubset returns a copy of actual containing only the keys present in
// template, recursively. Slices are kept verbatim (exact-match semantics).
// This enables subset assertion: extra fields in actual are silently ignored.
func projectSubset(actual, template any) any {
	t, tIsMap := template.(map[string]any)
	a, aIsMap := actual.(map[string]any)
	if !tIsMap || !aIsMap {
		return actual
	}
	result := make(map[string]any, len(t))
	for k, tv := range t {
		result[k] = projectSubset(a[k], tv)
	}
	return result
}

// Diff returns an empty string when every field declared in want exists in got
// with the same value (subset match). Extra fields in got are ignored.
func Diff(got, want []byte) (string, error) {
	gotDocs, err := parseDocs(got)
	if err != nil {
		return "", fmt.Errorf("parse actual: %w", err)
	}
	wantDocs, err := parseDocs(want)
	if err != nil {
		return "", fmt.Errorf("parse expected: %w", err)
	}

	projected := make([]any, len(wantDocs))
	for i, w := range wantDocs {
		if i < len(gotDocs) {
			projected[i] = projectSubset(gotDocs[i], w)
		}
		// projected[i] stays nil when got has fewer docs → diff shows missing doc
	}

	d := cmp.Diff(wantDocs, projected)
	return d, nil
}

func Indent(s, prefix string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

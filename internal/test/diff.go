package test

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// parseDocs parses a YAML stream into a slice of generic documents. Empty
// documents are dropped so that trailing `---` separators don't produce noise.
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

// normalize converts map[interface{}]interface{} values (which yaml.v3 should
// not produce, but be defensive) into map[string]interface{} so go-cmp output
// is readable and comparisons are stable.
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

// Diff returns an empty string when the two YAML byte slices are semantically
// equal, otherwise a human-readable diff.
func Diff(got, want []byte) (string, error) {
	gotDocs, err := parseDocs(got)
	if err != nil {
		return "", fmt.Errorf("parse actual: %w", err)
	}
	wantDocs, err := parseDocs(want)
	if err != nil {
		return "", fmt.Errorf("parse expected: %w", err)
	}
	d := cmp.Diff(wantDocs, gotDocs)
	return d, nil
}

// Indent prefixes every line with the given prefix. Used to format diffs as
// TAP "yaml-ish" subtext.
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

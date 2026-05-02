package test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	compositionAnnotation = "sularo.crossplane.io/composition"
	skipAnnotation        = "sularo.crossplane.io/skip"
)

type Case struct {
	Name        string
	Dir         string
	Composition string
	XR          string
	Expected    string
	Skip        bool
}

func Discover(root string) ([]Case, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", root, err)
	}

	var cases []Case
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		xrPath := filepath.Join(dir, "xr.yaml")

		annotations, err := xrAnnotations(xrPath)
		if err != nil {
			return nil, fmt.Errorf("test %s: %w", e.Name(), err)
		}

		skip := annotations[skipAnnotation] == "true"

		var compositionPath string
		if !skip {
			compositionPath, err = resolveComposition(dir, annotations)
			if err != nil {
				return nil, fmt.Errorf("test %s: %w", e.Name(), err)
			}
		}

		cases = append(cases, Case{
			Name:        e.Name(),
			Dir:         dir,
			Composition: compositionPath,
			XR:          xrPath,
			Expected:    filepath.Join(dir, "expected.yaml"),
			Skip:        skip,
		})
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	return cases, nil
}

func resolveComposition(dir string, annotations map[string]string) (string, error) {
	// Local composition.yaml takes precedence.
	local := filepath.Join(dir, "composition.yaml")
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}

	// Fall back to annotation pointing at a repo-relative path.
	if p := annotations[compositionAnnotation]; p != "" {
		if filepath.IsAbs(p) {
			return p, nil
		}
		return filepath.Clean(p), nil
	}

	return "", fmt.Errorf(
		"no composition found: add composition.yaml to the test dir or set annotation %q on the XR",
		compositionAnnotation,
	)
}

func xrAnnotations(xrPath string) (map[string]string, error) {
	f, err := os.Open(xrPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open xr: %w", err)
	}
	defer f.Close()

	var doc struct {
		Metadata struct {
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"metadata"`
	}
	if err := yaml.NewDecoder(f).Decode(&doc); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("parse xr: %w", err)
	}
	if doc.Metadata.Annotations == nil {
		return map[string]string{}, nil
	}
	return doc.Metadata.Annotations, nil
}

// applyFilter returns cases whose name contains filter as a substring.
// An empty filter returns all cases unchanged.
func applyFilter(cases []Case, filter string) []Case {
	if filter == "" {
		return cases
	}
	var out []Case
	for _, c := range cases {
		if strings.Contains(c.Name, filter) {
			out = append(out, c)
		}
	}
	return out
}

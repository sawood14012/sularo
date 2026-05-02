package test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const compositionAnnotation = "sularo.crossplane.io/composition"

type Case struct {
	Name        string
	Dir         string
	Composition string
	XR          string
	Expected    string
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

		compositionPath, err := resolveComposition(dir, xrPath)
		if err != nil {
			return nil, fmt.Errorf("test %s: %w", e.Name(), err)
		}

		cases = append(cases, Case{
			Name:        e.Name(),
			Dir:         dir,
			Composition: compositionPath,
			XR:          xrPath,
			Expected:    filepath.Join(dir, "expected.yaml"),
		})
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	return cases, nil
}

// resolveComposition returns the composition path for a test case.
// It first checks for a local composition.yaml in the test dir, then
// falls back to the sularo.crossplane.io/composition annotation on the XR.
func resolveComposition(dir, xrPath string) (string, error) {
	local := filepath.Join(dir, "composition.yaml")
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}

	annotated, err := compositionFromAnnotation(xrPath)
	if err != nil {
		return "", err
	}
	if annotated != "" {
		if filepath.IsAbs(annotated) {
			return annotated, nil
		}
		// Relative paths are resolved from the repo root (cwd), not the test dir.
		return filepath.Clean(annotated), nil
	}

	return "", fmt.Errorf(
		"no composition found: add a composition.yaml to the test dir or set annotation %q on the XR",
		compositionAnnotation,
	)
}

func compositionFromAnnotation(xrPath string) (string, error) {
	f, err := os.Open(xrPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("open xr: %w", err)
	}
	defer f.Close()

	return extractAnnotation(f)
}

func extractAnnotation(r io.Reader) (string, error) {
	var doc struct {
		Metadata struct {
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"metadata"`
	}
	if err := yaml.NewDecoder(r).Decode(&doc); err != nil {
		if err == io.EOF {
			return "", nil
		}
		return "", fmt.Errorf("parse xr: %w", err)
	}
	return doc.Metadata.Annotations[compositionAnnotation], nil
}

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
	functionsAnnotation   = "sularo.crossplane.io/functions"
	skipAnnotation        = "sularo.crossplane.io/skip"
)

type Case struct {
	Name        string
	Dir         string
	Composition string
	XR          string
	Functions   string // optional; empty means no functions file
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
		dirCases, err := discoverDir(e.Name(), dir)
		if err != nil {
			return nil, err
		}
		cases = append(cases, dirCases...)
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	return cases, nil
}

// discoverDir finds all xr*.yaml files in dir and produces one Case per file.
// xr.yaml → name is dirName (backwards-compat).
// xr-<suffix>.yaml → name is dirName/suffix.
// expected file mirrors the xr filename: xr-suffix.yaml → expected-suffix.yaml.
func discoverDir(dirName, dir string) ([]Case, error) {
	xrFiles, err := filepath.Glob(filepath.Join(dir, "xr*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(xrFiles) == 0 {
		return nil, nil
	}

	var cases []Case
	for _, xrPath := range xrFiles {
		base := filepath.Base(xrPath) // e.g. "xr.yaml" or "xr-with-subnets.yaml"

		var caseName, expectedFile string
		if base == "xr.yaml" {
			caseName = dirName
			expectedFile = "expected.yaml"
		} else {
			// xr-<suffix>.yaml → suffix
			suffix := strings.TrimSuffix(strings.TrimPrefix(base, "xr-"), ".yaml")
			caseName = dirName + "/" + suffix
			expectedFile = "expected-" + suffix + ".yaml"
		}

		annotations, err := xrAnnotations(xrPath)
		if err != nil {
			return nil, fmt.Errorf("test %s: %w", caseName, err)
		}

		skip := annotations[skipAnnotation] == "true"

		var compositionPath, functionsPath string
		if !skip {
			compositionPath, err = resolveComposition(dir, annotations)
			if err != nil {
				return nil, fmt.Errorf("test %s: %w", caseName, err)
			}
			functionsPath = resolveFunctions(dir, annotations)
		}

		cases = append(cases, Case{
			Name:        caseName,
			Dir:         dir,
			Composition: compositionPath,
			XR:          xrPath,
			Functions:   functionsPath,
			Expected:    filepath.Join(dir, expectedFile),
			Skip:        skip,
		})
	}
	return cases, nil
}

func resolveComposition(dir string, annotations map[string]string) (string, error) {
	local := filepath.Join(dir, "composition.yaml")
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}

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

// resolveFunctions returns the functions file path, or empty string if none is
// configured. Functions are optional — compositions that don't use a pipeline
// don't need them.
func resolveFunctions(dir string, annotations map[string]string) string {
	local := filepath.Join(dir, "functions.yaml")
	if _, err := os.Stat(local); err == nil {
		return local
	}

	if p := annotations[functionsAnnotation]; p != "" {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Clean(p)
	}

	return ""
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

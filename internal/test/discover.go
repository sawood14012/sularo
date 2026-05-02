package test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

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
		c := Case{
			Name:        e.Name(),
			Dir:         dir,
			Composition: filepath.Join(dir, "composition.yaml"),
			XR:          filepath.Join(dir, "xr.yaml"),
			Expected:    filepath.Join(dir, "expected.yaml"),
		}
		cases = append(cases, c)
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	return cases, nil
}

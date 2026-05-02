package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type InitOptions struct {
	Name        string
	XRSource    string // path to source XR file to copy from
	Composition string // repo-relative path to composition
	Functions   string // repo-relative path to functions (optional)
	TestsRoot   string // defaults to ./tests
}

func Init(opts InitOptions) error {
	if opts.TestsRoot == "" {
		opts.TestsRoot = "./tests"
	}
	dir := filepath.Join(opts.TestsRoot, opts.Name)

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("test case %q already exists at %s", opts.Name, dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	xrBytes, err := injectAnnotations(opts.XRSource, opts.Composition, opts.Functions)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "xr.yaml"), xrBytes, 0644); err != nil {
		return fmt.Errorf("write xr.yaml: %w", err)
	}

	stub := fmt.Sprintf("# Run: sularo update --filter %s\n", opts.Name)
	if err := os.WriteFile(filepath.Join(dir, "expected.yaml"), []byte(stub), 0644); err != nil {
		return fmt.Errorf("write expected.yaml: %w", err)
	}

	fmt.Printf("created %s/xr.yaml\n", dir)
	fmt.Printf("created %s/expected.yaml\n", dir)
	fmt.Printf("\nRun `sularo update --filter %s` to populate expected.yaml\n", opts.Name)
	return nil
}

// injectAnnotations reads a YAML XR file, sets the sularo annotations on
// metadata.annotations, and returns the re-encoded YAML bytes.
func injectAnnotations(xrPath, composition, functions string) ([]byte, error) {
	f, err := os.Open(xrPath)
	if err != nil {
		return nil, fmt.Errorf("open xr source %s: %w", xrPath, err)
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read xr source: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse xr: %w", err)
	}
	if doc.Kind == 0 {
		return nil, fmt.Errorf("xr source %s is empty", xrPath)
	}

	// yaml.Unmarshal wraps in a document node; unwrap to the mapping node.
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	annots := map[string]string{
		compositionAnnotation: composition,
	}
	if functions != "" {
		annots[functionsAnnotation] = functions
	}

	if err := setAnnotations(root, annots); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("encode xr: %w", err)
	}
	return buf.Bytes(), nil
}

// setAnnotations walks a YAML mapping node and merges keys into
// metadata.annotations, creating intermediate nodes as needed.
func setAnnotations(root *yaml.Node, annots map[string]string) error {
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("xr root is not a mapping")
	}

	metaNode := mappingValue(root, "metadata")
	if metaNode == nil {
		metaNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "metadata", Tag: "!!str"},
			metaNode,
		)
	}

	annotNode := mappingValue(metaNode, "annotations")
	if annotNode == nil {
		annotNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		metaNode.Content = append(metaNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "annotations", Tag: "!!str"},
			annotNode,
		)
	}

	for k, v := range annots {
		setMappingKey(annotNode, k, v)
	}
	return nil
}

// mappingValue returns the value node for key in a YAML mapping node, or nil.
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// setMappingKey sets or overwrites key=value in a YAML mapping node.
func setMappingKey(m *yaml.Node, key, value string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1].Value = value
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value, Tag: "!!str"},
	)
}

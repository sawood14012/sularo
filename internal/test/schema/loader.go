package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// Registry maps "group/version/Kind" to a compiled JSON schema.
type Registry struct {
	schemas map[string]*jsonschema.Schema
}

// Empty returns true when no CRDs were loaded — validation is a no-op.
func (r *Registry) Empty() bool {
	return len(r.schemas) == 0
}

// LoadDir walks dir for YAML files, parses any CRDs found, and returns a
// Registry. Files that are not CRDs are silently skipped.
func LoadDir(dir string) (*Registry, error) {
	r := &Registry{schemas: make(map[string]*jsonschema.Schema)}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return r, nil // no crds/ dir → empty registry, validation skipped
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	for _, path := range entries {
		if err := r.loadFile(path); err != nil {
			return nil, fmt.Errorf("load %s: %w", path, err)
		}
	}
	return r, nil
}

func (r *Registry) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var crd map[string]any
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return err
	}

	kind, _ := crd["kind"].(string)
	if kind != "CustomResourceDefinition" {
		return nil
	}

	spec, _ := crd["spec"].(map[string]any)
	if spec == nil {
		return nil
	}
	group, _ := spec["group"].(string)
	names, _ := spec["names"].(map[string]any)
	crdKind, _ := names["kind"].(string)

	// v1: spec.versions[].schema.openAPIV3Schema
	if versions, ok := spec["versions"].([]any); ok {
		for _, v := range versions {
			ver, _ := v.(map[string]any)
			version, _ := ver["name"].(string)
			schemaBlock, _ := ver["schema"].(map[string]any)
			rawSchema, _ := schemaBlock["openAPIV3Schema"].(map[string]any)
			if rawSchema == nil {
				continue
			}
			key := gvkKey(group, version, crdKind)
			if err := r.compile(key, rawSchema); err != nil {
				return fmt.Errorf("compile schema for %s: %w", key, err)
			}
		}
		return nil
	}

	// v1beta1: spec.validation.openAPIV3Schema
	validation, _ := spec["validation"].(map[string]any)
	rawSchema, _ := validation["openAPIV3Schema"].(map[string]any)
	if rawSchema == nil {
		return nil
	}
	// v1beta1 doesn't list versions in the same way; use spec.version
	version, _ := spec["version"].(string)
	if version == "" {
		version = "*"
	}
	key := gvkKey(group, version, crdKind)
	return r.compile(key, rawSchema)
}

func (r *Registry) compile(key string, raw map[string]any) error {
	cleaned := stripExtensions(raw)
	b, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7
	if err := compiler.AddResource(key, strings.NewReader(string(b))); err != nil {
		return err
	}
	s, err := compiler.Compile(key)
	if err != nil {
		return err
	}
	r.schemas[key] = s
	return nil
}

func gvkKey(group, version, kind string) string {
	return group + "/" + version + "/" + kind
}

// stripExtensions removes x-kubernetes-* keys and converts nullable:true to
// a ["type","null"] union so standard JSON Schema validators can process the schema.
func stripExtensions(v any) any {
	switch x := v.(type) {
	case map[string]any:
		clean := make(map[string]any, len(x))
		nullable := false
		for k, vv := range x {
			if strings.HasPrefix(k, "x-kubernetes-") {
				continue
			}
			if k == "nullable" {
				if b, ok := vv.(bool); ok && b {
					nullable = b
				}
				continue
			}
			clean[k] = stripExtensions(vv)
		}
		if nullable {
			if t, ok := clean["type"].(string); ok {
				clean["type"] = []any{t, "null"}
			}
		}
		return clean
	case []any:
		out := make([]any, len(x))
		for i, vv := range x {
			out[i] = stripExtensions(vv)
		}
		return out
	}
	return v
}

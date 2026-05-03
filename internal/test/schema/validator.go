package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Validate checks each resource in resources against the registry.
// Resources whose GVK has no CRD in the registry are skipped.
// Returns a list of human-readable validation errors.
func (r *Registry) Validate(resources []map[string]any) []string {
	if r.Empty() {
		return nil
	}

	var errs []string
	for _, res := range resources {
		apiVersion, _ := res["apiVersion"].(string)
		kind, _ := res["kind"].(string)
		name := resourceName(res)

		group, version := splitAPIVersion(apiVersion)
		key := group + "/" + version + "/" + kind

		s, ok := r.schemas[key]
		if !ok {
			continue // no CRD registered for this GVK — skip
		}

		b, err := json.Marshal(res)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s %s: marshal: %v", kind, name, err))
			continue
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			errs = append(errs, fmt.Sprintf("%s %s: unmarshal: %v", kind, name, err))
			continue
		}

		if err := s.Validate(v); err != nil {
			errs = append(errs, fmt.Sprintf("%s %s: %v", kind, name, err))
		}
	}
	return errs
}

func splitAPIVersion(apiVersion string) (group, version string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", apiVersion // core group (e.g. "v1")
}

func resourceName(res map[string]any) string {
	meta, _ := res["metadata"].(map[string]any)
	if meta == nil {
		return "<unknown>"
	}
	if name, ok := meta["name"].(string); ok && name != "" {
		return name
	}
	if gen, ok := meta["generateName"].(string); ok {
		return gen + "(generated)"
	}
	return "<unknown>"
}

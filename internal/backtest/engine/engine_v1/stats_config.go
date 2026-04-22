package engine

import (
	"fmt"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"
)

// configToYAMLNode marshals an arbitrary value (typically the engine config)
// into a *yaml.Node so it can be embedded inline as a YAML mapping in
// stats.yaml rather than as a quoted string.
func configToYAMLNode(v any) (*yamlv3.Node, error) {
	data, err := yamlv3.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	var node yamlv3.Node
	if err := yamlv3.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to decode config YAML node: %w", err)
	}

	return unwrapDocumentNode(&node), nil
}

// strategyConfigToYAMLNode parses a raw YAML string (the strategy config
// supplied to the engine) into a *yaml.Node. Returns nil for empty input so
// the field is omitted from the output entirely.
func strategyConfigToYAMLNode(content string) (*yamlv3.Node, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	var node yamlv3.Node
	if err := yamlv3.Unmarshal([]byte(content), &node); err != nil {
		return nil, fmt.Errorf("failed to parse strategy config YAML: %w", err)
	}

	return unwrapDocumentNode(&node), nil
}

// unwrapDocumentNode unwraps a top-level DocumentNode produced by
// yaml.Unmarshal so that the inner content is embedded inline (avoiding a
// stray document separator when re-marshaled inside another document).
func unwrapDocumentNode(node *yamlv3.Node) *yamlv3.Node {
	if node == nil {
		return nil
	}

	if node.Kind == yamlv3.DocumentNode && len(node.Content) == 1 {
		return node.Content[0]
	}

	return node
}

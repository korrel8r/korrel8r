// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package yaml wraps sigs.k8s.io/yaml, using sonic for the JSON marshal/unmarshal steps.
// YAML is converted to JSON by sigs.k8s.io/yaml, then unmarshaled by sonic.
// For marshaling, sonic produces JSON which is then converted to YAML.
package yaml

import (
	"github.com/bytedance/sonic"
	k8syaml "sigs.k8s.io/yaml"
)

var (
	std = sonic.ConfigStd
)

// Marshal marshals obj to YAML, using sonic for the intermediate JSON step.
func Marshal(obj any) ([]byte, error) {
	j, err := std.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return k8syaml.JSONToYAML(j)
}

// Unmarshal converts YAML to JSON, then uses sonic to unmarshal into obj.
func Unmarshal(yamlBytes []byte, obj any) error {
	j, err := k8syaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return err
	}
	return std.Unmarshal(j, obj)
}

func UnmarshalStrict(yamlBytes []byte, obj any) error {
	return k8syaml.UnmarshalStrict(yamlBytes, obj)
}

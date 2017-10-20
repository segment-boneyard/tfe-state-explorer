package tfstate

import (
	"encoding/json"
	"io"
)

type State struct {
	Version          int       `json:"version"`
	TerraformVersion string    `json:"terraform_version"`
	Serial           int64     `json:"serial"`
	Lineage          string    `json:"lineage"`
	Modules          []*Module `json:"modules"`
}

type Module struct {
	Path      []string             `json:"path"`
	Outputs   map[string]*Output   `json:"outputs"`
	Resources map[string]*Resource `json:"resources"`
	// Dependencies []string `json:"depends_on"`
}

type Output struct {
	Sensitive bool        `json:"sensitive"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
}

type Resource struct {
	Type         string    `json:"type"`
	Dependencies []string  `json:"depends_on"`
	Primary      *Instance `json:"primary"`
	Provider     string    `json:"provider"`
}

type Instance struct {
	ID         string                 `json:"id"`
	Attributes map[string]string      `json:"attributes"`
	Meta       map[string]interface{} `json:"meta"`
}

func ReadState(in io.Reader) (*State, error) {
	decoder := json.NewDecoder(in)
	var s State
	err := decoder.Decode(&s)
	return &s, err
}

type AttributeOrOutput struct {
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	ValueType string      `json:"valueType"`
}

func (s *State) FlattenAttributesAndOutputs() map[string]AttributeOrOutput {
	flat := map[string]AttributeOrOutput{}

	for _, mod := range s.Modules {
		modPath := getModulePath(mod)

		for key, output := range mod.Outputs {
			v := AttributeOrOutput{
				Type:      "output",
				ValueType: output.Type,
				Value:     output.Value,
			}
			flat[modPath+key] = v
		}

		for key, resource := range mod.Resources {
			path := modPath + key + "."
			for attrKey, attr := range resource.Primary.Attributes {
				attrPath := path + attrKey
				v := AttributeOrOutput{
					Type:      "attribute",
					ValueType: "string",
					Value:     attr,
				}
				flat[attrPath] = v
			}
		}
	}

	return flat
}

func getModulePath(mod *Module) string {
	path := ""
	for _, p := range mod.Path {
		if p == "root" {
			continue
		}
		path += "module." + p + "."
	}
	return path
}

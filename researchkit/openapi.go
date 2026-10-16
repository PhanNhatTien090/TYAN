package researchkit

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type APIOperation struct {
	Method          string   `json:"method"`
	Path            string   `json:"path"`
	PathParameters  []string `json:"path_parameters"`
	QueryParameters []string `json:"query_parameters"`
	BodyFields      []string `json:"body_fields"`
	ResponseFields  []string `json:"response_fields"`
	Authentication  string   `json:"authentication"`
}
type APISummary struct {
	Version    string         `json:"version"`
	Total      int            `json:"total"`
	ByMethod   map[string]int `json:"by_method"`
	Operations []APIOperation `json:"operations"`
	Truncated  bool           `json:"truncated"`
	Notice     string         `json:"notice"`
}

func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }
func str(v any) string         { s, _ := v.(string); return s }
func list(v any) []any         { a, _ := v.([]any); return a }
func unique(values []string) []string {
	set := map[string]bool{}
	for _, s := range values {
		if s != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// localRef resolves only in-document JSON pointers. It never requests a URL.
func localRef(root, node map[string]any, depth int) map[string]any {
	if depth > 6 {
		return nil
	}
	ref := str(node["$ref"])
	if ref == "" {
		return node
	}
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var value any = root
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		value = obj(value)[part]
	}
	return localRef(root, obj(value), depth+1)
}

func shape(root, node map[string]any, depth int) []string {
	if depth > 6 {
		return []string{}
	}
	node = localRef(root, node, 0)
	out := []string{}
	for _, key := range sortedKeys(obj(node["properties"])) {
		out = append(out, key)
		if len(out) >= 80 {
			break
		}
	}
	if len(out) == 0 && str(node["type"]) == "array" {
		for _, field := range shape(root, obj(node["items"]), depth+1) {
			out = append(out, "[]."+field)
		}
	}
	for _, union := range []string{"allOf", "oneOf", "anyOf"} {
		for _, part := range list(node[union]) {
			out = append(out, shape(root, obj(part), depth+1)...)
			if len(out) > 80 {
				out = out[:80]
				break
			}
		}
	}
	out = unique(out)
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}

func mediaShape(root, node map[string]any) []string {
	node = localRef(root, node, 0)
	if schema := obj(node["schema"]); schema != nil {
		return shape(root, schema, 0)
	}
	content := obj(node["content"])
	keys := sortedKeys(content)
	for _, k := range keys {
		if strings.Contains(k, "json") {
			return shape(root, obj(obj(content[k])["schema"]), 0)
		}
	}
	if len(keys) > 0 {
		return shape(root, obj(obj(content[keys[0]])["schema"]), 0)
	}
	return []string{}
}

func ParseOpenAPI(input string, limit int) (APISummary, error) {
	if err := CheckInput(input); err != nil {
		return APISummary{}, err
	}
	var root map[string]any
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		return APISummary{}, errors.New("invalid JSON/YAML specification")
	}
	version := str(root["openapi"])
	if version == "" {
		version = str(root["swagger"])
	}
	if !(strings.HasPrefix(version, "3.") || version == "2.0") {
		return APISummary{}, errors.New("expected OpenAPI 3.x or Swagger 2.0")
	}
	paths := obj(root["paths"])
	if paths == nil {
		return APISummary{}, errors.New("specification has no paths object")
	}
	if len(paths) > 3000 {
		return APISummary{}, errors.New("too many paths; split the specification")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	out := APISummary{Version: version, ByMethod: map[string]int{}, Operations: []APIOperation{}, Notice: "Schema names only; examples/defaults/credentials are omitted. Authentication is declared metadata, not a verified control. External references are not fetched."}
	methods := map[string]bool{"get": true, "post": true, "put": true, "delete": true, "patch": true, "head": true, "options": true, "trace": true}
	for _, path := range sortedKeys(paths) {
		item := obj(paths[path])
		if len(path) > 2048 || !strings.HasPrefix(path, "/") {
			return APISummary{}, fmt.Errorf("invalid or oversized operation path")
		}
		for _, method := range sortedKeys(item) {
			if !methods[method] {
				continue
			}
			op := obj(item[method])
			if op == nil {
				continue
			}
			out.Total++
			out.ByMethod[strings.ToUpper(method)]++
			if len(out.Operations) >= limit {
				out.Truncated = true
				continue
			}
			row := APIOperation{Method: strings.ToUpper(method), Path: path, PathParameters: []string{}, QueryParameters: []string{}, BodyFields: []string{}, ResponseFields: []string{}, Authentication: "unspecified"}
			// Operation security overrides root security, including an explicit [] override.
			sec, has := op["security"]
			if !has {
				sec, has = root["security"]
			}
			if has {
				requirements := list(sec)
				if len(requirements) == 0 {
					row.Authentication = "none_declared"
				} else {
					row.Authentication = "required_declared"
					for _, r := range requirements {
						if len(obj(r)) == 0 {
							row.Authentication = "optional_declared"
						}
					}
				}
			}
			params := append(append([]any{}, list(item["parameters"])...), list(op["parameters"])...)
			for _, p := range params {
				param := localRef(root, obj(p), 0)
				name := str(param["name"])
				switch str(param["in"]) {
				case "path":
					row.PathParameters = append(row.PathParameters, name)
				case "query":
					row.QueryParameters = append(row.QueryParameters, name)
				case "body":
					row.BodyFields = append(row.BodyFields, shape(root, obj(param["schema"]), 0)...)
				case "formData":
					row.BodyFields = append(row.BodyFields, name)
				}
			}
			row.PathParameters = unique(row.PathParameters)
			row.QueryParameters = unique(row.QueryParameters)
			row.BodyFields = unique(append(row.BodyFields, mediaShape(root, obj(op["requestBody"]))...))
			responses := obj(op["responses"])
			for _, code := range sortedKeys(responses) {
				if strings.HasPrefix(code, "2") {
					row.ResponseFields = unique(append(row.ResponseFields, mediaShape(root, obj(responses[code]))...))
				}
			}
			out.Operations = append(out.Operations, row)
		}
	}
	return out, nil
}

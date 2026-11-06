// SPDX-License-Identifier: Apache-2.0

package apicontract

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

type schemas struct{ components map[string]any }

func (s *schemas) schema(t reflect.Type, optional bool) map[string]any {
	if t.Kind() == reflect.Pointer {
		item := s.schema(t.Elem(), false)
		if optional {
			return item
		}
		return map[string]any{"anyOf": []any{item, map[string]any{"type": "null"}}}
	}
	switch t {
	case reflect.TypeFor[uuid.UUID]():
		return map[string]any{"type": "string", "format": "uuid"}
	case reflect.TypeFor[time.Time]():
		return map[string]any{"type": "string", "format": "date-time"}
	case reflect.TypeFor[rbac.Permission]():
		return map[string]any{"type": "string", "enum": rbac.Permissions()}
	case reflect.TypeFor[rbac.ScopeType]():
		return map[string]any{"type": "string", "enum": []string{"unit", "connection"}}
	case reflect.TypeFor[rbac.Source]():
		return map[string]any{"type": "string", "enum": []string{"direct", "inherited", "inherited_hidden"}}
	}
	switch t.Kind() {
	case reflect.Struct:
		name := t.Name()
		if _, ok := s.components[name]; !ok {
			s.components[name] = map[string]any{}
			properties := map[string]any{}
			required := []string{}
			s.fields(t, properties, &required)
			s.components[name] = map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
		}
		return map[string]any{"$ref": "#/components/schemas/" + name}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64, reflect.Int32:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": s.schema(t.Elem(), false)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": s.schema(t.Elem(), false)}
	default:
		return map[string]any{}
	}
}
func (s *schemas) fields(t reflect.Type, properties map[string]any, required *[]string) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		if f.Anonymous && tag == "" {
			s.fields(f.Type, properties, required)
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = f.Name
		}
		optional := strings.Contains(tag, ",omitempty")
		properties[name] = s.schema(f.Type, optional)
		if !optional {
			*required = append(*required, name)
		}
	}
}

var pathParameter = regexp.MustCompile(`\{([a-z_]+)\}`)

func Document() map[string]any {
	builder := &schemas{components: map[string]any{}}
	paths := map[string]any{}
	builder.schema(reflect.TypeFor[auth.User](), false)
	errorSchema := builder.schema(reflect.TypeFor[Error](), false)
	for _, op := range Operations() {
		path, ok := paths[op.Path].(map[string]any)
		if !ok {
			path = map[string]any{}
			paths[op.Path] = path
		}
		parameters := []any{}
		for _, match := range pathParameter.FindAllStringSubmatch(op.Path, -1) {
			schema := map[string]any{"type": "string", "format": "uuid"}
			if match[1] == "token" {
				schema = map[string]any{"type": "string", "pattern": "^[A-Za-z0-9_-]{43}$"}
			}
			parameters = append(parameters, map[string]any{"name": match[1], "in": "path", "required": true, "schema": schema})
		}
		for _, q := range op.Query {
			schema := map[string]any{"type": q.Type}
			if q.Type == "uuid" || q.Type == "date-time" {
				schema = map[string]any{"type": "string", "format": q.Type}
			}
			parameters = append(parameters, map[string]any{"name": q.Name, "in": "query", "required": false, "description": q.Description, "schema": schema})
		}
		success := map[string]any{"description": "Success"}
		if op.Response != nil {
			success["content"] = map[string]any{"application/json": map[string]any{"schema": builder.schema(reflect.TypeOf(op.Response), false)}}
		}
		responses := map[string]any{statusString(op.Status): success}
		for _, status := range []string{"400", "401", "403", "404", "409", "413", "429", "503"} {
			responses[status] = map[string]any{"description": "Request rejected; unreadable and missing units both return 403", "content": map[string]any{"application/json": map[string]any{"schema": errorSchema}}}
		}
		operation := map[string]any{"operationId": op.ID, "summary": op.Summary, "parameters": parameters, "responses": responses, "security": []any{map[string]any{"session": []string{}}}}
		if op.Public {
			operation["security"] = []any{}
		}
		if op.Method != "GET" {
			parameters = append(parameters, map[string]any{"name": "X-CSRF-Token", "in": "header", "required": true, "schema": map[string]any{"type": "string"}})
			operation["parameters"] = parameters
		}
		if op.Request != nil {
			operation["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": builder.schema(reflect.TypeOf(op.Request), false)}}}
		}
		path[strings.ToLower(op.Method)] = operation
	}
	return map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "Schewise administration API", "version": "0.1.0", "license": map[string]any{"name": "Apache-2.0", "identifier": "Apache-2.0"}}, "paths": paths, "components": map[string]any{"schemas": builder.components, "securitySchemes": map[string]any{"session": map[string]any{"type": "apiKey", "in": "cookie", "name": "__Host-schewise_session"}}}}
}
func statusString(status int) string {
	switch status {
	case 201:
		return "201"
	case 204:
		return "204"
	default:
		return "200"
	}
}
func JSON() ([]byte, error)        { return json.MarshalIndent(Document(), "", "  ") }
func FiberPath(path string) string { return pathParameter.ReplaceAllString(path, ":$1") }

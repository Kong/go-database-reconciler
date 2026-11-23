package main

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"sort"

	"github.com/alecthomas/jsonschema"
	"github.com/iancoleman/orderedmap"
	"github.com/kong/go-database-reconciler/pkg/file"
	"github.com/kong/go-kong/kong"
)

const nameField = "name"

func newProperties(properties map[string]*jsonschema.Type) *orderedmap.OrderedMap {
	result := orderedmap.New()
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result.Set(name, properties[name])
	}
	return result
}

func sortProperties(properties *orderedmap.OrderedMap) {
	if properties == nil {
		return
	}
	if len(properties.Keys()) == 0 {
		return
	}
	properties.SortKeys(sort.Strings)
	for _, name := range properties.Keys() {
		value, ok := properties.Get(name)
		if !ok {
			continue
		}
		property, ok := value.(*jsonschema.Type)
		if !ok {
			continue
		}
		sortProperties(property.Properties)
		if property.Items != nil {
			sortProperties(property.Items.Properties)
		}
	}
}

var (
	// routes and services
	anyOfNameOrID = []*jsonschema.Type{
		{
			Required: []string{nameField},
		},
		{
			Required: []string{"id"},
		},
	}

	// consumers
	anyOfUsernameOrCustomID = []*jsonschema.Type{
		{
			Description: "at least one of custom_id or username must be set",
			Required:    []string{"username"},
		},
		{
			Description: "at least one of custom_id or username must be set",
			Required:    []string{"custom_id"},
		},
	}
)

func main() {
	var reflector jsonschema.Reflector
	reflector.ExpandedStruct = true
	reflector.RequiredFromJSONSchemaTags = true
	reflector.TypeMapper = func(typ reflect.Type) *jsonschema.Type {
		// plugin configuration
		if typ == reflect.TypeFor[kong.Configuration]() {
			return &jsonschema.Type{
				Type: "object",

				AdditionalProperties: []byte("true"),
			}
		}
		return nil
	}
	schema := reflector.Reflect(file.Content{})

	schema.Definitions["FService"].AnyOf = anyOfNameOrID

	schema.Definitions["FRoute"].AnyOf = anyOfNameOrID

	schema.Definitions["FKey"].AnyOf = anyOfNameOrID

	schema.Definitions["FKeySet"].AnyOf = anyOfNameOrID

	schema.Definitions["FConsumer"].AnyOf = anyOfUsernameOrCustomID

	schema.Definitions["FUpstream"].Required = []string{nameField}

	schema.Definitions["FTarget"].Required = []string{"target"}
	schema.Definitions["FCACertificate"].Required = []string{"cert"}
	schema.Definitions["FPlugin"].Required = []string{nameField}

	schema.Definitions["FCertificate"].Required = []string{"id", "cert", "key"}
	schema.Definitions["FCertificate"].Properties.Set("snis", &jsonschema.Type{
		Type: "array",
		Items: &jsonschema.Type{
			Type: "object",
			Properties: newProperties(map[string]*jsonschema.Type{
				nameField: {
					Type: "string",
				},
			}),
		},
	})

	schema.Definitions["FFilterChain"].Required = []string{"filters"}

	schema.Definitions["FFilterChain"].Properties.Set("enabled", &jsonschema.Type{
		Type: "boolean",
	})

	schema.Definitions["FFilterChain"].Properties.Set("filters", &jsonschema.Type{
		Type: "array",
		Items: &jsonschema.Type{
			Ref: "#/definitions/FFilter",
		},
	})

	schema.Definitions["FFilter"] = &jsonschema.Type{
		Type:                 "object",
		Required:             []string{nameField},
		AdditionalProperties: json.RawMessage(`false`),
		Properties: newProperties(map[string]*jsonschema.Type{
			nameField: {
				Type: "string",
			},
			"config": {
				OneOf: []*jsonschema.Type{
					{Type: "array"},
					{Type: "boolean"},
					{Type: "integer"},
					{Type: "number"},
					{Type: "null"},
					{Type: "object"},
					{Type: "string"},
				},
			},
			"enabled": {
				Type: "boolean",
			},
		}),
	}

	// creds
	schema.Definitions["ACLGroup"].Required = []string{"group"}
	schema.Definitions["BasicAuth"].Required = []string{"username", "password"}
	schema.Definitions["HMACAuth"].Required = []string{"username", "secret"}
	schema.Definitions["JWTAuth"].Required = []string{
		"algorithm", "key",
		"secret",
	}
	schema.Definitions["KeyAuth"].Required = []string{"key"}
	schema.Definitions["Oauth2Credential"].Required = []string{
		nameField,
		"client_id", "client_secret",
	}
	schema.Definitions["MTLSAuth"].Required = []string{"id", "subject_name"}

	// custom entities
	schema.Definitions["FCustomEntity"].Required = []string{"type"}

	// RBAC resources
	schema.Definitions["FRBACRole"].Required = []string{nameField}
	schema.Definitions["FRBACEndpointPermission"].Required = []string{"workspace", "endpoint"}

	// partials
	schema.Definitions["FPartial"].Required = []string{"type"}

	// cloned plugin definitions
	schema.Definitions["FClonedPluginDefinition"].Required = []string{nameField, "ref"}

	// custom plugin definitions
	schema.Definitions["FCustomPluginDefinition"].Required = []string{nameField, "schema", "handler"}

	// AI model definitions
	schema.Definitions["FAIModel"].Required = []string{nameField}

	// Foreign references
	stringType := &jsonschema.Type{Type: "string"}
	schema.Definitions["FPlugin"].Properties.Set("consumer", stringType)
	schema.Definitions["FPlugin"].Properties.Set("service", stringType)
	schema.Definitions["FPlugin"].Properties.Set("route", stringType)
	schema.Definitions["FPlugin"].Properties.Set("consumer_group", stringType)
	schema.Definitions["FPlugin"].Properties.Set("model", stringType)

	schema.Definitions["FFilterChain"].Properties.Set("service", stringType)
	schema.Definitions["FFilterChain"].Properties.Set("route", stringType)

	schema.Definitions["FService"].Properties.Set("client_certificate", stringType)

	// Info.Defaults has a JSON tag without omitempty but a YAML tag with it.
	// The schema describes the optional YAML representation, not a required JSON field.
	schema.Definitions["Info"].Required = nil

	// Keep the existing schema for raw filter configuration.
	schema.Definitions["Filter"].Properties.Set("config", &jsonschema.Type{
		Type:  "array",
		Items: &jsonschema.Type{Type: "integer"},
	})

	// konnect resources
	schema.Definitions["FServicePackage"].Required = []string{nameField}
	schema.Definitions["FServiceVersion"].Required = []string{"version"}
	schema.Definitions["Implementation"].Required = []string{"type", "kong"}

	sortProperties(schema.Properties)
	for _, definition := range schema.Definitions {
		sortProperties(definition.Properties)
	}

	jsonSchema, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}

	err = os.WriteFile("kong_json_schema.json", jsonSchema, 0o644)
	if err != nil {
		log.Fatalln(err)
	}
}

package dump

import (
	"reflect"
	"testing"

	schema_pkg "github.com/kong/go-database-reconciler/pkg/schema"
	"github.com/kong/go-kong/kong"
	"github.com/tidwall/gjson"
)

// Test entities for testing
type TestEntity struct {
	Name        *string                `json:"name,omitempty"`
	Port        *int                   `json:"port,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	NestedField *NestedTestEntity      `json:"nested_field,omitempty"`
}

type NestedTestEntity struct {
	Value *string `json:"value,omitempty"`
}

type PluginEntity struct {
	Name   *string                `json:"name,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type VaultEntity struct {
	Name   *string                `json:"name,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type PartialEntity struct {
	Type   *string                `json:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

func TestRemoveDefaultsFromState_EmptyEntities(t *testing.T) {
	// Create a schema fetcher with a mock client that won't be used
	schemaFetcher := &SchemaFetcher{}
	entities := []*TestEntity{}
	err := processStateEntities(entities, schemaFetcher, "test")
	if err != nil {
		t.Errorf("Expected no error for empty entities, got %v", err)
	}
}

func TestRemoveDefaultsFromEntity_NonPointer(t *testing.T) {
	schemaFetcher := &SchemaFetcher{}
	entity := TestEntity{Name: kong.String("test")}
	err := removeDefaultsFromEntity(entity, "test", schemaFetcher)
	if err == nil {
		t.Error("Expected error for non-pointer entity, got nil")
	}
	if err.Error() != "entity is not a pointer" {
		t.Errorf("Expected 'entity is not a pointer', got %q", err.Error())
	}
}

func TestParseSchemaForDefaults(t *testing.T) {
	tests := []struct {
		name           string
		schemaJSON     string
		expectedFields map[string]interface{}
	}{
		{
			name: "simple schema with defaults",
			schemaJSON: `{
				"fields": [
					{
						"name": {
							"type": "string",
							"default": "default-name"
						}
					},
					{
						"port": {
							"type": "number",
							"default": 8080
						}
					},
					{
						"enabled": {
							"type": "bool",
							"default": true
						}
					},
					{
						"methods": {
							"type": "array",
							"elements": {
								"type": "string"
							},
							"default": ["GET", "POST"]
						}
					},
					{
						"nums": {
							"type": "set",
							"items": {
								"type": "number"
							},
							"default": [1, 2]
						}
					}

				]
			}`,
			expectedFields: map[string]interface{}{
				"name":    "default-name",
				"port":    float64(8080),
				"enabled": true,
				"methods": []interface{}{"GET", "POST"},
				"nums":    []interface{}{float64(1), float64(2)},
			},
		},
		{
			name: "schema with nested fields",
			schemaJSON: `{
				"fields": [
					{
						"name": {
							"type": "string",
							"default": "default-name"
						}
					},
					{
						"config": {
							"type": "record",
							"fields": [
								{
									"timeout": {
										"type": "number",
										"default": 5000
									}
								},
								{
									"retries": {
										"type": "number",
										"default": 3
									}
								}
							]
						}
					},
				]
			}`,
			expectedFields: map[string]interface{}{
				"name": "default-name",
				"config": map[string]interface{}{
					"timeout": float64(5000),
					"retries": float64(3),
				},
			},
		},
		{
			name: "schema with default record value",
			schemaJSON: `{
				"fields": [{
					"test": {
						"fields": [{
							"name": {
								"type": "string"
							}
						}]
					}
				}],
				"default": {
					"test": {
						"name": "record-default"
					}
				}
			}`,
			expectedFields: map[string]interface{}{
				"test": map[string]interface{}{
					"name": "record-default",
				},
			},
		},
		{
			name: "empty schema",
			schemaJSON: `{
				"fields": {}
			}`,
			expectedFields: map[string]interface{}{},
		},
		{
			name: "schema with deprecated shorthand_fields with backward translation",
			schemaJSON: `{
				"fields": [
					{
						"name": {
							"type": "string",
							"default": "default-name"
						}
					},
					{
						"config": {
							"type": "record",
							"fields": [
								{
									"redis": {
										"type": "record",
										"fields": [
											{
												"host": {
													"type": "string",
													"default": "localhost"
												}
											},
											{
												"port": {
													"type": "number",
													"default": 6379
												}
											}
										]
									}
								}
							],
							"shorthand_fields": [
								{	
									"redis_host": {
										"translate_backwards": [
											"redis",
											"host"
										],
										"type": "string"
									}
								},
								{
									"redis_port": {
										"translate_backwards": [
											"redis",
											"port"
										],
										"type": "integer"
									}	
								}
							]
						}
					}
				]
			}`,
			expectedFields: map[string]interface{}{
				"name": "default-name",
				"config": map[string]interface{}{
					"redis": map[string]interface{}{
						"host": "localhost",
						"port": float64(6379),
					},
					"redis_host": "localhost",
					"redis_port": float64(6379),
				},
			},
		},
		{
			name: "schema with deprecatedshorthand_fields with replaced_with paths",
			schemaJSON: `{
				"fields": [
					{
						"name": {
							"type": "string",
							"default": "default-name"
						}
					},
					{
						"config": {
							"type": "record",
							"fields": [
								{
									"redis": {
										"type": "record",
										"fields": [
											{
												"host": {
													"type": "string",
													"default": "localhost"
												}
											},
											{
												"port": {
													"type": "number",
													"default": 6379
												}
											},
											{
												"read_timeout": {
													"type": "number",
													"default": 10
												}
											},
											{
												"send_timeout": {
													"type": "number",
													"default": 10
												}
											}
										]
									}
								}
							],
							"shorthand_fields": [
								{	
									"redis_host": {
										"deprecation": {
											"replaced_with": [{
												"path": ["redis", "host"]
											}]
										},
										"type": "string"
									}
								},
								{
									"redis_port": {
										"deprecation": {
											"replaced_with": [
												{
													"path": ["redis", "port"]
												}
											]
										},
										"type": "integer"
									}	
								},
								{
									"timeout": {
										"deprecation": {
											"replaced_with": [
												{
													"path": ["redis", "read_timeout"]
												},
												{
													"path": ["redis", "send_timeout"]
												}
											]
										},
										"type": "integer"
									}	
								}
							]
						}
					}
				]
			}`,
			expectedFields: map[string]interface{}{
				"name": "default-name",
				"config": map[string]interface{}{
					"redis": map[string]interface{}{
						"host":         "localhost",
						"port":         float64(6379),
						"read_timeout": float64(10),
						"send_timeout": float64(10),
					},
					"redis_host": "localhost",
					"redis_port": float64(6379),
					"timeout":    float64(10),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := gjson.Parse(tt.schemaJSON)
			defaultFields := make(map[string]interface{})
			result := schema_pkg.ParseSchemaForDefaults(schema, defaultFields)

			if !reflect.DeepEqual(result, tt.expectedFields) {
				t.Errorf("parseSchemaForDefaults() = %v, expected %v", result, tt.expectedFields)
			}
		})
	}
}

func TestParseEntityWithDefaults(t *testing.T) {
	tests := []struct {
		name          string
		setupEntity   func() *TestEntity
		defaultFields map[string]interface{}
		validateFunc  func(*testing.T, *TestEntity)
		expectedError bool
	}{
		{
			name: "remove default string field",
			setupEntity: func() *TestEntity {
				return &TestEntity{
					Name: kong.String(""),
					Port: kong.Int(8080),
				}
			},
			defaultFields: map[string]interface{}{
				"name": "",
				"port": 3000,
			},
			validateFunc: func(t *testing.T, entity *TestEntity) {
				if entity.Name != nil {
					t.Errorf("Expected Name to be nil (zero value), got %v", *entity.Name)
				}
				if entity.Port == nil || *entity.Port != 8080 {
					t.Errorf("Expected Port to remain 8080, got %v", entity.Port)
				}
			},
			expectedError: false,
		},
		{
			name: "remove default numeric field",
			setupEntity: func() *TestEntity {
				return &TestEntity{
					Port: kong.Int(3000),
				}
			},
			defaultFields: map[string]interface{}{
				"port": 3000,
			},
			validateFunc: func(t *testing.T, entity *TestEntity) {
				if entity.Port != nil {
					t.Errorf("Expected Port to be nil (zero value), got %v", *entity.Port)
				}
			},
			expectedError: false,
		},
		{
			name: "handle map fields",
			setupEntity: func() *TestEntity {
				return &TestEntity{
					Config: map[string]interface{}{
						"timeout": 5000,
						"retries": 3,
						"name":    "custom-config",
					},
				}
			},
			defaultFields: map[string]interface{}{
				"config": map[string]interface{}{
					"timeout": 5000,
				},
			},
			validateFunc: func(t *testing.T, entity *TestEntity) {
				if entity.Config == nil {
					t.Error("Expected Config to not be nil")
					return
				}
				if len(entity.Config) != 2 {
					t.Errorf("Expected Config to have 2 keys, got %d", len(entity.Config))
				}
				if retries, exists := entity.Config["retries"]; !exists || retries != 3 {
					t.Errorf("Expected Config to contain retries=3, got %v", retries)
				}
				if name, exists := entity.Config["name"]; !exists || name != "custom-config" {
					t.Errorf("Expected Config to contain name=custom-config, got %v", name)
				}
				if _, exists := entity.Config["timeout"]; exists {
					t.Error("Expected timeout to be removed from Config as it matches default")
				}
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := tt.setupEntity()
			entityValue := reflect.ValueOf(entity).Elem()
			err := stripDefaultValuesFromEntity(entityValue, tt.defaultFields)

			if (err != nil) != tt.expectedError {
				t.Errorf("stripDefaultValuesFromEntity() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !tt.expectedError && tt.validateFunc != nil {
				tt.validateFunc(t, entity)
			}
		})
	}
}

func TestCompareSlices(t *testing.T) {
	tests := []struct {
		name         string
		fieldSlice   interface{}
		defaultSlice interface{}
		expected     bool
	}{
		{
			name:         "equal string slices",
			fieldSlice:   []string{"a", "b", "c"},
			defaultSlice: []string{"a", "b", "c"},
			expected:     true,
		},
		{
			name:         "different length slices",
			fieldSlice:   []string{"a", "b"},
			defaultSlice: []string{"a", "b", "c"},
			expected:     false,
		},
		{
			name:         "different content slices",
			fieldSlice:   []string{"a", "x", "c"},
			defaultSlice: []string{"a", "b", "c"},
			expected:     false,
		},
		{
			name:         "empty slices",
			fieldSlice:   []string{},
			defaultSlice: []string{},
			expected:     true,
		},
		{
			name:         "int slices",
			fieldSlice:   []int{1, 2, 3},
			defaultSlice: []int{1, 2, 3},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldVal := reflect.ValueOf(tt.fieldSlice)
			defaultVal := reflect.ValueOf(tt.defaultSlice)
			result := compareSlices(fieldVal, defaultVal)
			if result != tt.expected {
				t.Errorf("compareSlices(%v, %v) = %v, expected %v", tt.fieldSlice, tt.defaultSlice, result, tt.expected)
			}
		})
	}
}

func TestCompareMaps(t *testing.T) {
	tests := []struct {
		name       string
		fieldMap   map[string]interface{}
		defaultMap map[string]interface{}
		expected   map[string]interface{}
	}{
		{
			name: "remove default values",
			fieldMap: map[string]interface{}{
				"timeout": 5000,
				"retries": 3,
				"name":    "test",
			},
			defaultMap: map[string]interface{}{
				"timeout": 5000,
				"name":    "default",
			},
			expected: map[string]interface{}{
				"retries": 3,
				"name":    "test",
			},
		},
		{
			name: "nested maps",
			fieldMap: map[string]interface{}{
				"config": map[string]interface{}{
					"timeout":   5000,
					"retries":   3,
					"protocols": []string{"http", "https"},
					"throttling": map[string]interface{}{
						"enabled": true,
						"max":     100,
					},
				},
				"enabled": true,
				"version": "1.0",
			},
			defaultMap: map[string]interface{}{
				"config": map[string]interface{}{
					"timeout":   5000,
					"protocols": []string{"http", "https"},
					"throttling": map[string]interface{}{
						"enabled": true,
						"max":     10,
					},
				},
				"enabled": false,
				"version": "1.0",
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"retries": 3,
					"throttling": map[string]interface{}{
						"max": 100,
					},
				},
				"enabled": true,
			},
		},
		{
			name: "all values are defaults",
			fieldMap: map[string]interface{}{
				"enabled": false,
				"version": "1.0",
				"config": map[string]interface{}{
					"timeout":   5000,
					"protocols": []string{"http", "https"},
				},
			},
			defaultMap: map[string]interface{}{
				"enabled": false,
				"version": "1.0",
				"config": map[string]interface{}{
					"timeout":   5000,
					"protocols": []string{"http", "https"},
				},
			},
			expected: map[string]interface{}{},
		},
		{
			name: "no values are defaults",
			fieldMap: map[string]interface{}{
				"enabled": true,
				"version": "2.0",
				"config": map[string]interface{}{
					"timeout":   6000,
					"protocols": []string{"grpc", "https"},
				},
			},
			defaultMap: map[string]interface{}{
				"enabled": false,
				"version": "1.0",
				"config": map[string]interface{}{
					"timeout":   5000,
					"protocols": []string{"http", "https"},
				},
			},
			expected: map[string]interface{}{
				"enabled": true,
				"version": "2.0",
				"config": map[string]interface{}{
					"timeout":   6000,
					"protocols": []string{"grpc", "https"},
				},
			},
		},
		{
			name: "field has nil value",
			fieldMap: map[string]interface{}{
				"timeout": 5000,
				"name":    nil,
			},
			defaultMap: map[string]interface{}{
				"timeout": 5000,
			},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldVal := reflect.ValueOf(tt.fieldMap)
			defaultVal := reflect.ValueOf(tt.defaultMap)
			result := compareMaps(fieldVal, defaultVal)

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Errorf("compareMaps() returned non-map type")
				return
			}

			if !reflect.DeepEqual(resultMap, tt.expected) {
				t.Errorf("compareMaps() = %v, expected %v", resultMap, tt.expected)
			}
		})
	}
}

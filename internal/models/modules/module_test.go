package modules_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/models"
	"github.com/a-novel/service-narrative-engine/internal/models/modules"
)

func TestSystemModuleUnmarshalYAML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		expect modules.SystemModule
	}{
		{
			name: "BasicFields",
			input: `
id: test-module
namespace: test-namespace
description: A test module description
schema:
  type: object
  properties:
    name:
      type: string
`,
			expect: modules.SystemModule{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Description: "A test module description",
			},
		},
		{
			name: "SchemaWithRequiredFields",
			input: `
id: required-test
namespace: test
description: Test required fields
schema:
  type: object
  required:
    - field1
    - field2
  properties:
    field1:
      type: string
    field2:
      type: integer
`,
			expect: modules.SystemModule{
				ID:          "required-test",
				Namespace:   "test",
				Description: "Test required fields",
			},
		},
		{
			name: "SchemaWithNestedObjects",
			input: `
id: nested-test
namespace: test
description: Test nested objects
schema:
  type: object
  properties:
    outer:
      type: object
      properties:
        inner:
          type: string
          description: Inner field
`,
			expect: modules.SystemModule{
				ID:          "nested-test",
				Namespace:   "test",
				Description: "Test nested objects",
			},
		},
		{
			name: "SchemaWithArrays",
			input: `
id: array-test
namespace: test
description: Test array types
schema:
  type: object
  properties:
    items:
      type: array
      items:
        type: string
`,
			expect: modules.SystemModule{
				ID:          "array-test",
				Namespace:   "test",
				Description: "Test array types",
			},
		},
		{
			name: "SchemaWithEnums",
			input: `
id: enum-test
namespace: test
description: Test enum values
schema:
  type: object
  properties:
    status:
      type: string
      enum:
        - ACTIVE
        - INACTIVE
        - PENDING
`,
			expect: modules.SystemModule{
				ID:          "enum-test",
				Namespace:   "test",
				Description: "Test enum values",
			},
		},
		{
			name: "SchemaWithConstraints",
			input: `
id: constraint-test
namespace: test
description: Test schema constraints
schema:
  type: object
  properties:
    percentage:
      type: integer
      minimum: 0
      maximum: 100
    name:
      type: string
      minLength: 1
      maxLength: 255
`,
			expect: modules.SystemModule{
				ID:          "constraint-test",
				Namespace:   "test",
				Description: "Test schema constraints",
			},
		},
		{
			name: "WithUIComponent",
			input: `
id: ui-test
namespace: test
description: Test UI component
schema:
  type: object
  properties:
    content:
      type: string
ui:
  component: text-editor
  target: content
  params:
    maxLength: 1000
    placeholder: Enter text here
`,
			expect: modules.SystemModule{
				ID:          "ui-test",
				Namespace:   "test",
				Description: "Test UI component",
				UI: models.ModuleUi{
					Component: "text-editor",
					Target:    "content",
					Params: map[string]any{
						"maxLength":   float64(1000),
						"placeholder": "Enter text here",
					},
				},
			},
		},
		{
			name: "EmptyUI",
			input: `
id: empty-ui
namespace: test
description: Test empty UI
schema:
  type: object
  properties:
    field:
      type: string
ui:
`,
			expect: modules.SystemModule{
				ID:          "empty-ui",
				Namespace:   "test",
				Description: "Test empty UI",
				UI:          models.ModuleUi{},
			},
		},
		{
			name: "MultilineDescription",
			input: `
id: multiline-desc
namespace: test
description: |
  This is a multiline description.
  It spans multiple lines.
  And has proper formatting.
schema:
  type: object
  properties:
    field:
      type: string
`,
			expect: modules.SystemModule{
				ID:          "multiline-desc",
				Namespace:   "test",
				Description: "This is a multiline description.\nIt spans multiple lines.\nAnd has proper formatting.\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var module modules.SystemModule

			err := yaml.Unmarshal([]byte(tc.input), &module)
			require.NoError(t, err)

			require.Equal(t, tc.expect.ID, module.ID)
			require.Equal(t, tc.expect.Namespace, module.Namespace)
			require.Equal(t, tc.expect.Description, module.Description)
			require.Equal(t, tc.expect.UI, module.UI)
		})
	}
}

func TestSystemModuleUnmarshalYAML_Schema(t *testing.T) {
	t.Parallel()

	t.Run("SchemaType", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-type
namespace: test
description: Test schema type
schema:
  type: object
  properties:
    field:
      type: string
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		require.Equal(t, "object", module.Schema.Type)
	})

	t.Run("SchemaRequired", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-required
namespace: test
description: Test schema required
schema:
  type: object
  required:
    - field1
    - field2
  properties:
    field1:
      type: string
    field2:
      type: integer
    field3:
      type: boolean
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		require.ElementsMatch(t, []string{"field1", "field2"}, module.Schema.Required)
	})

	t.Run("SchemaProperties", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-props
namespace: test
description: Test schema properties
schema:
  type: object
  properties:
    stringField:
      type: string
      description: A string field
    intField:
      type: integer
    boolField:
      type: boolean
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		require.Len(t, module.Schema.Properties, 3)
		require.NotNil(t, module.Schema.Properties["stringField"])
		require.Equal(t, "string", module.Schema.Properties["stringField"].Type)
		require.Equal(t, "A string field", module.Schema.Properties["stringField"].Description)
		require.NotNil(t, module.Schema.Properties["intField"])
		require.Equal(t, "integer", module.Schema.Properties["intField"].Type)
		require.NotNil(t, module.Schema.Properties["boolField"])
		require.Equal(t, "boolean", module.Schema.Properties["boolField"].Type)
	})

	t.Run("SchemaNestedObject", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-nested
namespace: test
description: Test nested schema
schema:
  type: object
  properties:
    parent:
      type: object
      additionalProperties: false
      required:
        - child
      properties:
        child:
          type: string
        optional:
          type: integer
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		parent := module.Schema.Properties["parent"]
		require.NotNil(t, parent)
		require.Equal(t, "object", parent.Type)
		require.ElementsMatch(t, []string{"child"}, parent.Required)
		require.Len(t, parent.Properties, 2)
		require.Equal(t, "string", parent.Properties["child"].Type)
		require.Equal(t, "integer", parent.Properties["optional"].Type)
	})

	t.Run("SchemaArrayWithItems", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-array
namespace: test
description: Test array schema
schema:
  type: object
  properties:
    tags:
      type: array
      items:
        type: string
    numbers:
      type: array
      items:
        type: integer
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		tags := module.Schema.Properties["tags"]
		require.NotNil(t, tags)
		require.Equal(t, "array", tags.Type)
		require.NotNil(t, tags.Items)
		require.Equal(t, "string", tags.Items.Type)

		numbers := module.Schema.Properties["numbers"]
		require.NotNil(t, numbers)
		require.Equal(t, "array", numbers.Type)
		require.NotNil(t, numbers.Items)
		require.Equal(t, "integer", numbers.Items.Type)
	})

	t.Run("SchemaArrayWithObjectItems", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-array-objects
namespace: test
description: Test array with object items
schema:
  type: object
  properties:
    users:
      type: array
      items:
        type: object
        required:
          - name
        properties:
          name:
            type: string
          age:
            type: integer
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		users := module.Schema.Properties["users"]
		require.NotNil(t, users)
		require.Equal(t, "array", users.Type)
		require.NotNil(t, users.Items)
		require.Equal(t, "object", users.Items.Type)
		require.ElementsMatch(t, []string{"name"}, users.Items.Required)
		require.Len(t, users.Items.Properties, 2)
	})

	t.Run("SchemaEnumValues", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-enum
namespace: test
description: Test enum values
schema:
  type: object
  properties:
    status:
      type: string
      enum:
        - PENDING
        - ACTIVE
        - COMPLETED
        - CANCELLED
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		status := module.Schema.Properties["status"]
		require.NotNil(t, status)
		require.Equal(t, "string", status.Type)
		require.Len(t, status.Enum, 4)
		require.Contains(t, status.Enum, "PENDING")
		require.Contains(t, status.Enum, "ACTIVE")
		require.Contains(t, status.Enum, "COMPLETED")
		require.Contains(t, status.Enum, "CANCELLED")
	})

	t.Run("SchemaNumericConstraints", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-numeric
namespace: test
description: Test numeric constraints
schema:
  type: object
  properties:
    percentage:
      type: integer
      minimum: 0
      maximum: 100
    rating:
      type: number
      minimum: 0.0
      maximum: 5.0
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		percentage := module.Schema.Properties["percentage"]
		require.NotNil(t, percentage)
		require.NotNil(t, percentage.Minimum)
		require.InDelta(t, 0.0, *percentage.Minimum, 0.000001)
		require.NotNil(t, percentage.Maximum)
		require.InDelta(t, 100.0, *percentage.Maximum, 0.000001)

		rating := module.Schema.Properties["rating"]
		require.NotNil(t, rating)
		require.NotNil(t, rating.Minimum)
		require.InDelta(t, 0.0, *rating.Minimum, 0.000001)
		require.NotNil(t, rating.Maximum)
		require.InDelta(t, 5.0, *rating.Maximum, 0.000001)
	})

	t.Run("SchemaStringConstraints", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-string
namespace: test
description: Test string constraints
schema:
  type: object
  properties:
    username:
      type: string
      minLength: 3
      maxLength: 50
      pattern: "^[a-zA-Z0-9_]+$"
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		username := module.Schema.Properties["username"]
		require.NotNil(t, username)
		require.NotNil(t, username.MinLength)
		require.Equal(t, 3, *username.MinLength)
		require.NotNil(t, username.MaxLength)
		require.Equal(t, 50, *username.MaxLength)
		require.Equal(t, "^[a-zA-Z0-9_]+$", username.Pattern)
	})

	t.Run("SchemaAdditionalProperties", func(t *testing.T) {
		t.Parallel()

		input := `
id: schema-additional
namespace: test
description: Test additionalProperties
schema:
  type: object
  additionalProperties: false
  properties:
    field:
      type: string
`

		var module modules.SystemModule

		err := yaml.Unmarshal([]byte(input), &module)
		require.NoError(t, err)

		require.NotNil(t, module.Schema.AdditionalProperties)
	})
}

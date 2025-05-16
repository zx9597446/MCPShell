package common

import (
	"testing"
)

func TestConvertStringToType(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		paramType   string
		expected    interface{}
		expectError bool
	}{
		{"string value", "test", "string", "test", false},
		{"number value", "42.5", "number", 42.5, false},
		{"integer value", "42", "integer", int64(42), false},
		{"boolean true", "true", "boolean", true, false},
		{"boolean yes", "yes", "boolean", true, false},
		{"boolean false", "false", "boolean", false, false},
		{"boolean no", "no", "boolean", false, false},
		{"invalid number", "not-a-number", "number", nil, true},
		{"invalid integer", "not-an-integer", "integer", nil, true},
		{"invalid boolean", "not-a-boolean", "boolean", nil, true},
		{"empty type defaults to string", "test", "", "test", false},
		{"unsupported type", "test", "unknown", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringToType(tt.value, tt.paramType)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && result != tt.expected {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

// Mock CommandHandler to test default parameter values
type mockCommandHandler struct {
	params map[string]ParamConfig
	args   map[string]interface{}
}

func (m *mockCommandHandler) applyDefaults() {
	for paramName, paramConfig := range m.params {
		if _, exists := m.args[paramName]; !exists && paramConfig.Default != nil {
			m.args[paramName] = paramConfig.Default
		}
	}
}

func TestParamConfigDefault(t *testing.T) {
	tests := []struct {
		name         string
		paramConfig  map[string]ParamConfig
		args         map[string]interface{}
		expectedArgs map[string]interface{}
	}{
		{
			name: "default string value applied",
			paramConfig: map[string]ParamConfig{
				"name": {
					Type:        "string",
					Description: "A name",
					Default:     "default-name",
				},
			},
			args: map[string]interface{}{},
			expectedArgs: map[string]interface{}{
				"name": "default-name",
			},
		},
		{
			name: "default number value applied",
			paramConfig: map[string]ParamConfig{
				"count": {
					Type:        "number",
					Description: "A count",
					Default:     42.5,
				},
			},
			args: map[string]interface{}{},
			expectedArgs: map[string]interface{}{
				"count": 42.5,
			},
		},
		{
			name: "default boolean value applied",
			paramConfig: map[string]ParamConfig{
				"flag": {
					Type:        "boolean",
					Description: "A flag",
					Default:     true,
				},
			},
			args: map[string]interface{}{},
			expectedArgs: map[string]interface{}{
				"flag": true,
			},
		},
		{
			name: "existing value not overridden by default",
			paramConfig: map[string]ParamConfig{
				"name": {
					Type:        "string",
					Description: "A name",
					Default:     "default-name",
				},
			},
			args: map[string]interface{}{
				"name": "provided-name",
			},
			expectedArgs: map[string]interface{}{
				"name": "provided-name",
			},
		},
		{
			name: "multiple defaults applied",
			paramConfig: map[string]ParamConfig{
				"name": {
					Type:        "string",
					Description: "A name",
					Default:     "default-name",
				},
				"count": {
					Type:        "number",
					Description: "A count",
					Default:     42.5,
				},
			},
			args: map[string]interface{}{},
			expectedArgs: map[string]interface{}{
				"name":  "default-name",
				"count": 42.5,
			},
		},
		{
			name: "no default value for some parameters",
			paramConfig: map[string]ParamConfig{
				"required": {
					Type:        "string",
					Description: "Required parameter",
					Required:    true,
				},
				"optional": {
					Type:        "string",
					Description: "Optional parameter with default",
					Default:     "default-value",
				},
			},
			args: map[string]interface{}{
				"required": "provided-value",
			},
			expectedArgs: map[string]interface{}{
				"required": "provided-value",
				"optional": "default-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockCommandHandler{
				params: tt.paramConfig,
				args:   make(map[string]interface{}),
			}

			// Copy the original args to avoid modifying the test case
			for k, v := range tt.args {
				handler.args[k] = v
			}

			// Apply defaults
			handler.applyDefaults()

			// Check if all expected args are present with the correct values
			for paramName, expectedValue := range tt.expectedArgs {
				value, exists := handler.args[paramName]
				if !exists {
					t.Errorf("Expected parameter '%s' to be set, but it wasn't", paramName)
					continue
				}

				if value != expectedValue {
					t.Errorf("Parameter '%s': expected %v (%T), got %v (%T)",
						paramName, expectedValue, expectedValue, value, value)
				}
			}

			// Check that there are no unexpected args
			if len(handler.args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expectedArgs), len(handler.args))
			}
		})
	}
}

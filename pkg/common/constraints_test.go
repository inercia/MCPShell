package common

import (
	"testing"
)

// Create a test logger that discards output to keep test output clean
var testLogger, _ = NewLogger("", "", LogLevelNone, false)

// TestConstraints tests both compilation and evaluation of constraints
func TestConstraints(t *testing.T) {
	// Test cases for compilation and evaluation
	tests := []struct {
		name           string
		constraints    []string
		paramTypes     map[string]ParamConfig
		args           map[string]interface{}
		wantCompileErr bool
		wantEvalResult bool
		wantEvalErr    bool
		skipEvaluation bool
	}{
		// Compilation-only test cases
		{
			name:           "Empty constraints",
			constraints:    []string{},
			paramTypes:     map[string]ParamConfig{},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Single string constraint",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Multiple string constraints",
			constraints: []string{"text.size() < 10", "text.startsWith('hello')"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Number constraints",
			constraints: []string{"value > 0.0", "value < 100.0"},
			paramTypes: map[string]ParamConfig{
				"value": {Type: "number", Description: "Numeric value"},
			},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Boolean constraints",
			constraints: []string{"flag == true"},
			paramTypes: map[string]ParamConfig{
				"flag": {Type: "boolean", Description: "Boolean flag"},
			},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Mixed type constraints",
			constraints: []string{"name.size() > 0", "value > 0.0", "flag == true"},
			paramTypes: map[string]ParamConfig{
				"name":  {Type: "string", Description: "Name"},
				"value": {Type: "number", Description: "Value"},
				"flag":  {Type: "boolean", Description: "Flag"},
			},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Invalid constraint syntax",
			constraints: []string{"text.invalid()"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			skipEvaluation: true,
			wantCompileErr: true,
		},
		{
			name:        "Unknown parameter",
			constraints: []string{"unknown.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			skipEvaluation: true,
			wantCompileErr: true,
		},
		{
			name:        "Default string type",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Description: "Text input"}, // No type specified, should default to string
			},
			skipEvaluation: true,
			wantCompileErr: false,
		},
		{
			name:        "Unsupported parameter type",
			constraints: []string{"obj.field == 'value'"},
			paramTypes: map[string]ParamConfig{
				"obj": {Type: "object", Description: "Object"}, // Unsupported type
			},
			skipEvaluation: true,
			wantCompileErr: true,
		},
		{
			name:        "Nil logger",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			skipEvaluation: true,
			wantCompileErr: true,
		},

		// Compilation and evaluation test cases
		{
			name:           "Empty constraints - evaluation",
			constraints:    []string{},
			paramTypes:     map[string]ParamConfig{},
			args:           map[string]interface{}{},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "String length constraint - pass",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{"text": "short"},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "String length constraint - fail",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{"text": "this is a long text"},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    false,
		},
		{
			name:        "Numeric comparison - pass",
			constraints: []string{"value > 0.0", "value <= 100.0"},
			paramTypes: map[string]ParamConfig{
				"value": {Type: "number", Description: "Numeric value"},
			},
			args:           map[string]interface{}{"value": 50.0},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Numeric comparison - fail",
			constraints: []string{"value > 0.0", "value <= 100.0"},
			paramTypes: map[string]ParamConfig{
				"value": {Type: "number", Description: "Numeric value"},
			},
			args:           map[string]interface{}{"value": 150.0},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    false,
		},
		{
			name:        "Boolean constraint - pass",
			constraints: []string{"flag == true"},
			paramTypes: map[string]ParamConfig{
				"flag": {Type: "boolean", Description: "Boolean flag"},
			},
			args:           map[string]interface{}{"flag": true},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Boolean constraint - fail",
			constraints: []string{"flag == true"},
			paramTypes: map[string]ParamConfig{
				"flag": {Type: "boolean", Description: "Boolean flag"},
			},
			args:           map[string]interface{}{"flag": false},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    false,
		},
		{
			name:        "String pattern matching - pass",
			constraints: []string{"text.matches('^[a-z]+$')"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{"text": "lowercase"},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "String pattern matching - fail",
			constraints: []string{"text.matches('^[a-z]+$')"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{"text": "Mixed123"},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    false,
		},
		{
			name:        "Multiple constraints - all pass",
			constraints: []string{"name.size() > 0", "value > 0.0", "flag == true"},
			paramTypes: map[string]ParamConfig{
				"name":  {Type: "string", Description: "Name"},
				"value": {Type: "number", Description: "Value"},
				"flag":  {Type: "boolean", Description: "Flag"},
			},
			args:           map[string]interface{}{"name": "Alice", "value": 42.0, "flag": true},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Multiple constraints - one fails",
			constraints: []string{"name.size() > 0", "value > 0.0", "flag == true"},
			paramTypes: map[string]ParamConfig{
				"name":  {Type: "string", Description: "Name"},
				"value": {Type: "number", Description: "Value"},
				"flag":  {Type: "boolean", Description: "Flag"},
			},
			args:           map[string]interface{}{"name": "Alice", "value": -5.0, "flag": true},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    false,
		},
		{
			name:        "Existential quantifier - pass",
			constraints: []string{"['apple', 'banana', 'orange'].exists(f, f == fruit)"},
			paramTypes: map[string]ParamConfig{
				"fruit": {Type: "string", Description: "Fruit name"},
			},
			args:           map[string]interface{}{"fruit": "banana"},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Existential quantifier - fail",
			constraints: []string{"['apple', 'banana', 'orange'].exists(f, f == fruit)"},
			paramTypes: map[string]ParamConfig{
				"fruit": {Type: "string", Description: "Fruit name"},
			},
			args:           map[string]interface{}{"fruit": "grape"},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    false,
		},
		{
			name:        "Missing parameter",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Wrong parameter type",
			constraints: []string{"text.size() < 10"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{"text": 123},
			wantCompileErr: false,
			wantEvalResult: false,
			wantEvalErr:    true,
		},
		{
			name:        "Default string value",
			constraints: []string{"text.size() == 0"},
			paramTypes: map[string]ParamConfig{
				"text": {Type: "string", Description: "Text input"},
			},
			args:           map[string]interface{}{},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Default number value",
			constraints: []string{"value == 0.0"},
			paramTypes: map[string]ParamConfig{
				"value": {Type: "number", Description: "Numeric value"},
			},
			args:           map[string]interface{}{},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Default boolean value",
			constraints: []string{"flag == false"},
			paramTypes: map[string]ParamConfig{
				"flag": {Type: "boolean", Description: "Boolean flag"},
			},
			args:           map[string]interface{}{},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Multiple default values",
			constraints: []string{"name.size() == 0", "value == 0.0", "flag == false"},
			paramTypes: map[string]ParamConfig{
				"name":  {Type: "string", Description: "Name"},
				"value": {Type: "number", Description: "Value"},
				"flag":  {Type: "boolean", Description: "Flag"},
			},
			args:           map[string]interface{}{},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
		{
			name:        "Partial parameters provided",
			constraints: []string{"name.size() > 0", "value == 0.0", "flag == true"},
			paramTypes: map[string]ParamConfig{
				"name":  {Type: "string", Description: "Name"},
				"value": {Type: "number", Description: "Value"},
				"flag":  {Type: "boolean", Description: "Flag"},
			},
			args:           map[string]interface{}{"name": "Bob", "flag": true},
			wantCompileErr: false,
			wantEvalResult: true,
			wantEvalErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the compilation phase
			var compiled *CompiledConstraints
			var err error

			if tt.name == "Nil logger" {
				compiled, err = NewCompiledConstraints(tt.constraints, tt.paramTypes, nil)
			} else {
				compiled, err = NewCompiledConstraints(tt.constraints, tt.paramTypes, testLogger)
			}

			// Check compile error expectation
			if (err != nil) != tt.wantCompileErr {
				t.Errorf("NewCompiledConstraints() error = %v, wantCompileErr %v", err, tt.wantCompileErr)
				return
			}

			if err != nil {
				// If compilation failed as expected, no need to test evaluation
				return
			}

			// For empty constraints, compiled should be non-nil but with empty programs
			if len(tt.constraints) == 0 && compiled == nil {
				t.Errorf("NewCompiledConstraints() returned nil for empty constraints")
				return
			}

			// Skip evaluation for compilation-only test cases
			if tt.skipEvaluation {
				return
			}

			// Test the evaluation phase
			got, failedConstraints, err := compiled.Evaluate(tt.args, tt.paramTypes)

			// Check evaluation error expectation
			if (err != nil) != tt.wantEvalErr {
				t.Errorf("CompiledConstraints.EvaluateWithDetails() error = %v, wantEvalErr %v", err, tt.wantEvalErr)
				return
			}

			// Check evaluation result expectation
			if got != tt.wantEvalResult {
				t.Errorf("CompiledConstraints.EvaluateWithDetails() = %v, want %v", got, tt.wantEvalResult)
			}

			// Verify failed constraints array is consistent with the result
			if err == nil {
				if !got && len(failedConstraints) == 0 {
					t.Errorf("CompiledConstraints.EvaluateWithDetails() returned false but no failed constraints")
				}
				if got && failedConstraints != nil {
					t.Errorf("CompiledConstraints.EvaluateWithDetails() returned true but had failed constraints: %v", failedConstraints)
				}
			}
		})
	}

	// Test with nil CompiledConstraints (special case)
	t.Run("Nil constraints pass by default", func(t *testing.T) {
		emptyConstraints := &CompiledConstraints{logger: testLogger}
		got, failedConstraints, err := emptyConstraints.Evaluate(map[string]interface{}{"value": 42.0}, nil)
		if err != nil {
			t.Errorf("CompiledConstraints.EvaluateWithDetails() error = %v, wantErr false", err)
			return
		}
		if !got {
			t.Errorf("CompiledConstraints.EvaluateWithDetails() = %v, want true", got)
		}
		if failedConstraints != nil {
			t.Errorf("CompiledConstraints.EvaluateWithDetails() returned failed constraints when result is true: %v", failedConstraints)
		}
	})
}

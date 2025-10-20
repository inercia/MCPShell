package common

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

// CompiledConstraints holds the compiled CEL programs for a tool's constraints
type CompiledConstraints struct {
	programs    []cel.Program
	expressions []string // Original constraint expressions
	logger      *Logger
}

// NewCompiledConstraints compiles a list of CEL constraint expressions
// paramTypes is a map of parameter names to their types
// logger is required for logging constraint compilation and evaluation information
func NewCompiledConstraints(constraints []string, paramTypes map[string]ParamConfig, logger *Logger) (*CompiledConstraints, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required for constraint compilation")
	}

	if len(constraints) == 0 {
		logger.Debug("No constraints to compile")
		return &CompiledConstraints{logger: logger}, nil
	}

	// Create a new CEL environment with the parameter declarations
	var envOpts []cel.EnvOption

	// Add parameter declarations based on their types
	for name, param := range paramTypes {
		paramType := param.Type
		if paramType == "" {
			paramType = "string"
		}

		switch paramType {
		case "string":
			envOpts = append(envOpts, cel.Variable(name, cel.StringType))
		case "number", "integer":
			envOpts = append(envOpts, cel.Variable(name, cel.DoubleType))
		case "boolean":
			envOpts = append(envOpts, cel.Variable(name, cel.BoolType))
		default:
			return nil, fmt.Errorf("unsupported parameter type for CEL: %s", paramType)
		}
	}

	env, err := cel.NewEnv(envOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	// Compile each constraint expression
	var programs []cel.Program
	var expressions []string
	for _, expr := range constraints {
		ast, issues := env.Compile(expr)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("failed to compile constraint '%s': %w", expr, issues.Err())
		}

		// Create a program from the AST
		prg, err := env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("failed to create program for constraint '%s': %w", expr, err)
		}

		programs = append(programs, prg)
		expressions = append(expressions, expr)
	}

	return &CompiledConstraints{
		programs:    programs,
		expressions: expressions,
		logger:      logger,
	}, nil
}

// Evaluate evaluates all compiled constraints against the provided arguments
// and returns details about which constraints failed.
//
// Parameters:
//   - args: Map of argument names to their values
//   - paramTypes: Map of parameter names to their type configurations
//
// Returns:
//   - true if all constraints pass, false otherwise
//   - slice of strings containing the failed constraint expressions
//   - error if evaluation fails or if a required parameter is missing
func (cc *CompiledConstraints) Evaluate(args map[string]interface{}, params map[string]ParamConfig) (bool, []string, error) {
	if cc == nil {
		return true, nil, nil
	}

	if len(cc.programs) == 0 {
		// If there are no constraints, evaluation passes by default
		cc.logger.Debug("No constraints to evaluate, passing by default")
		return true, nil, nil
	}

	cc.logger.Debug("Evaluating %d constraints with details", len(cc.programs))

	// Create a copy of args to avoid modifying the original
	evalArgs := make(map[string]interface{})
	for k, v := range args {
		evalArgs[k] = v
		cc.logger.Debug("Argument provided: %s = %v", k, v)
	}

	// Ensure all parameters have at least empty values if not provided
	for name, param := range params {
		if _, exists := evalArgs[name]; !exists {
			// Parameter not provided, add default empty value based on type
			switch param.Type {
			case "string", "":
				evalArgs[name] = ""
				cc.logger.Debug("Adding default empty string for missing parameter: %s", name)
			case "number", "integer":
				evalArgs[name] = 0.0
				cc.logger.Debug("Adding default zero value for missing parameter: %s", name)
			case "boolean":
				evalArgs[name] = false
				cc.logger.Debug("Adding default false value for missing parameter: %s", name)
			}
		}
	}

	var failedConstraints []string

	// Evaluate each constraint program
	for i, prg := range cc.programs {
		// Execute the program
		cc.logger.Debug("Evaluating constraint #%d: %s", i+1, cc.expressions[i])
		val, _, err := prg.Eval(evalArgs)
		if err != nil {
			cc.logger.Debug("Constraint #%d evaluation error: %v", i+1, err)
			return false, nil, fmt.Errorf("constraint evaluation error: %w", err)
		}

		// Check if the result is a boolean and is true
		boolVal, ok := val.Value().(bool)
		if !ok {
			cc.logger.Debug("Constraint #%d did not evaluate to a boolean", i+1)
			return false, nil, fmt.Errorf("constraint did not evaluate to a boolean")
		}

		if !boolVal {
			// If any constraint fails, add it to the failed constraints list
			failureMsg := fmt.Sprintf("%s (with values: %s)", cc.expressions[i], formatArgValues(evalArgs))
			failedConstraints = append(failedConstraints, failureMsg)
			cc.logger.Debug("Constraint #%d failed evaluation: %s", i+1, failureMsg)
		} else {
			cc.logger.Debug("Constraint #%d passed evaluation", i+1)
		}
	}

	// Return failure if any constraints failed
	if len(failedConstraints) > 0 {
		cc.logger.Debug("%d constraints failed evaluation", len(failedConstraints))
		return false, failedConstraints, nil
	}

	// All constraints passed
	cc.logger.Debug("All constraints passed evaluation")
	return true, nil, nil
}

// formatArgValues returns a formatted string of the argument values for error reporting
func formatArgValues(args map[string]interface{}) string {
	result := ""
	for k, v := range args {
		if result != "" {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", k, v)
	}
	return result
}

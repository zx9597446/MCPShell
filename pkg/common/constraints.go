package common

import (
	"fmt"
	"log"

	"github.com/google/cel-go/cel"
)

// CompiledConstraints holds the compiled CEL programs for a tool's constraints
type CompiledConstraints struct {
	programs []cel.Program
	logger   *log.Logger
}

// NewCompiledConstraints compiles a list of CEL constraint expressions
// paramTypes is a map of parameter names to their types
// logger is required for logging constraint compilation and evaluation information
func NewCompiledConstraints(constraints []string, paramTypes map[string]ParamConfig, logger *log.Logger) (*CompiledConstraints, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required for constraint compilation")
	}

	if len(constraints) == 0 {
		logger.Println("No constraints to compile")
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
	}

	return &CompiledConstraints{
		programs: programs,
		logger:   logger,
	}, nil
}

// Evaluate evaluates all compiled constraints against the provided arguments
// Returns true if all constraints pass, false otherwise
//
// Parameters:
//   - args: Map of argument names to their values
//   - paramTypes: Map of parameter names to their type configurations
//
// Returns:
//   - true if all constraints pass, false otherwise
//   - error if evaluation fails or if a required parameter is missing
func (cc *CompiledConstraints) Evaluate(args map[string]interface{}, paramTypes map[string]ParamConfig) (bool, error) {
	if cc == nil {
		return true, nil
	}

	if len(cc.programs) == 0 {
		// If there are no constraints, evaluation passes by default
		cc.logger.Println("No constraints to evaluate, passing by default")
		return true, nil
	}

	cc.logger.Printf("Evaluating %d constraints", len(cc.programs))

	// Create a copy of args to avoid modifying the original
	evalArgs := make(map[string]interface{})
	for k, v := range args {
		evalArgs[k] = v
		cc.logger.Printf("Argument provided: %s = %v", k, v)
	}

	// Ensure all parameters have at least empty values if not provided
	for name, param := range paramTypes {
		if _, exists := evalArgs[name]; !exists {
			// Parameter not provided, add default empty value based on type
			switch param.Type {
			case "string", "":
				evalArgs[name] = ""
				cc.logger.Printf("Adding default empty string for missing parameter: %s", name)
			case "number", "integer":
				evalArgs[name] = 0.0
				cc.logger.Printf("Adding default zero value for missing parameter: %s", name)
			case "boolean":
				evalArgs[name] = false
				cc.logger.Printf("Adding default false value for missing parameter: %s", name)
			}
		}
	}

	// Evaluate each constraint program
	for i, prg := range cc.programs {
		// Execute the program
		cc.logger.Printf("Evaluating constraint #%d", i+1)
		val, _, err := prg.Eval(evalArgs)
		if err != nil {
			cc.logger.Printf("Constraint #%d evaluation error: %v", i+1, err)
			return false, fmt.Errorf("constraint evaluation error: %w", err)
		}

		// Check if the result is a boolean and is true
		boolVal, ok := val.Value().(bool)
		if !ok {
			cc.logger.Printf("Constraint #%d did not evaluate to a boolean", i+1)
			return false, fmt.Errorf("constraint did not evaluate to a boolean")
		}

		if !boolVal {
			// If any constraint fails, the evaluation fails
			cc.logger.Printf("Constraint #%d failed evaluation", i+1)
			return false, nil
		}

		cc.logger.Printf("Constraint #%d passed evaluation", i+1)
	}

	// All constraints passed
	cc.logger.Println("All constraints passed evaluation")
	return true, nil
}

package collection

import (
	"errors"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/expr-lang/expr/types"
	"github.com/expr-lang/expr/vm"

	"harns-platform/pkg/gateway/validation"
)

// Calculation is one compiled calculated variable.
type Calculation struct {
	Variable *Variable
	program  *vm.Program
	deps     []string
}

// CalculationError describes a calculated variable expression problem.
type CalculationError struct {
	Variable string
	Missing  string
	Cycle    bool
	Err      error
}

func (e *CalculationError) Error() string {
	switch {
	case e == nil:
		return ""
	case e.Missing != "":
		return fmt.Sprintf("calculated variable %s references unknown variable %s", e.Variable, e.Missing)
	case e.Cycle:
		return fmt.Sprintf("calculated variable cycle at %s", e.Variable)
	case e.Err != nil:
		return fmt.Sprintf("compile variable %s: %v", e.Variable, e.Err)
	default:
		return "invalid calculated variable"
	}
}

func (e *CalculationError) Unwrap() error { return e.Err }

// CompileCalculations compiles calculated variables and sorts them by dependency.
func CompileCalculations(variables []*Variable) ([]*Calculation, error) {
	known := make(map[string]struct{}, len(variables))
	for _, variable := range variables {
		if variable != nil {
			known[variable.Name] = struct{}{}
		}
	}
	// Declare collection variables in the compile environment so names that
	// collide with expr builtins (for example "one", "min", "len") resolve to
	// variables instead of functions.
	env := make(types.Map, len(known))
	for name := range known {
		env[name] = types.Any
	}
	compileOptions := []expr.Option{expr.Env(env), expr.AllowUndefinedVariables()}
	calculations := make([]*Calculation, 0)
	byName := make(map[string]*Calculation)
	for _, variable := range variables {
		if variable == nil || !variable.IsCalculated {
			continue
		}
		dependencies, err := expressionDependencies(variable.Expression)
		if err != nil {
			return nil, &CalculationError{Variable: variable.Name, Err: err}
		}
		knownDependencies := make([]string, 0, len(dependencies))
		for _, dependency := range dependencies {
			if _, ok := known[dependency]; !ok {
				return nil, &CalculationError{Variable: variable.Name, Missing: dependency}
			}
			knownDependencies = append(knownDependencies, dependency)
		}
		program, err := expr.Compile(variable.Expression, compileOptions...)
		if err != nil {
			return nil, &CalculationError{Variable: variable.Name, Err: err}
		}
		item := &Calculation{
			Variable: variable,
			program:  program,
			deps:     knownDependencies,
		}
		calculations = append(calculations, item)
		byName[variable.Name] = item
	}
	return sortCalculations(calculations, byName)
}

// ValidateCalculations validates calculated variable expressions and dependencies.
func ValidateCalculations(variables []*Variable) error {
	_, err := CompileCalculations(variables)
	if err == nil {
		return nil
	}
	indexes := make(map[string]int)
	for index, variable := range variables {
		if variable != nil && variable.IsCalculated {
			indexes[variable.Name] = index
		}
	}
	var calculationErr *CalculationError
	if errors.As(err, &calculationErr) {
		field := "variables"
		if index, ok := indexes[calculationErr.Variable]; ok {
			field = fmt.Sprintf("variables[%d].expression", index)
		}
		switch {
		case calculationErr.Missing != "":
			return validation.New([]validation.FieldIssue{{
				Field: field,
				MsgID: "validate.calculated_variable_reference_missing",
				Args:  []interface{}{calculationErr.Variable, calculationErr.Missing},
			}})
		case calculationErr.Cycle:
			return validation.New([]validation.FieldIssue{{
				Field: field,
				MsgID: "validate.calculated_variable_cycle",
				Args:  []interface{}{calculationErr.Variable},
			}})
		default:
			return validation.New([]validation.FieldIssue{{
				Field: field,
				MsgID: "validate.calculated_variable_expression_invalid",
				Args:  []interface{}{calculationErr.Variable, compactExpressionError(calculationErr.Err)},
			}})
		}
	}
	return validation.New([]validation.FieldIssue{{
		Field: "variables",
		MsgID: "validate.calculated_variable_expression_invalid",
		Args:  []interface{}{"", compactExpressionError(err)},
	}})
}

func compactExpressionError(err error) string {
	if err == nil {
		return "invalid expression"
	}
	text := err.Error()
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		text = text[:index]
	}
	text = strings.Join(strings.Fields(text), " ")
	const maxRunes = 160
	runes := []rune(text)
	if len(runes) > maxRunes {
		text = string(runes[:maxRunes]) + "..."
	}
	if text == "" {
		return "invalid expression"
	}
	return text
}

// Calculate evaluates compiled calculations against the collection's current values.
func Calculate(meta *CollectionMeta, calculations []*Calculation) error {
	if meta == nil || len(calculations) == 0 {
		return nil
	}
	environment := make(map[string]any)
	for _, variable := range meta.SnapshotVariables() {
		if variable != nil {
			environment[variable.Name] = variable.Value
		}
	}
	for _, item := range calculations {
		result, err := expr.Run(item.program, environment)
		if err != nil {
			return fmt.Errorf("calculate variable %s: %w", item.Variable.Name, err)
		}
		value := ApplyTransform(result, item.Variable)
		meta.SetVariableValue(item.Variable.Name, value)
		environment[item.Variable.Name] = value
	}
	return nil
}

type identifierVisitor struct {
	names []string
	seen  map[string]struct{}
}

func (v *identifierVisitor) Visit(node *ast.Node) {
	identifier, ok := (*node).(*ast.IdentifierNode)
	if !ok {
		return
	}
	if _, exists := v.seen[identifier.Value]; exists {
		return
	}
	v.seen[identifier.Value] = struct{}{}
	v.names = append(v.names, identifier.Value)
}

func expressionDependencies(expression string) ([]string, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, err
	}
	visitor := &identifierVisitor{seen: make(map[string]struct{})}
	ast.Walk(&tree.Node, visitor)
	return visitor.names, nil
}

func sortCalculations(calculations []*Calculation, byName map[string]*Calculation) ([]*Calculation, error) {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(calculations))
	order := make([]*Calculation, 0, len(calculations))
	var visit func(*Calculation) error
	visit = func(item *Calculation) error {
		switch state[item.Variable.Name] {
		case visiting:
			return &CalculationError{Variable: item.Variable.Name, Cycle: true}
		case visited:
			return nil
		}
		state[item.Variable.Name] = visiting
		for _, dependency := range item.deps {
			if dependencyItem, ok := byName[dependency]; ok {
				if err := visit(dependencyItem); err != nil {
					return err
				}
			}
		}
		state[item.Variable.Name] = visited
		order = append(order, item)
		return nil
	}
	for _, item := range calculations {
		if err := visit(item); err != nil {
			return nil, err
		}
	}
	return order, nil
}

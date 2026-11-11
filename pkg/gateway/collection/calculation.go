package collection

import (
	"fmt"
	"regexp"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var identifierPattern = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// Calculation is one compiled calculated variable.
type Calculation struct {
	Variable *Variable
	program  *vm.Program
	deps     []string
}

// CompileCalculations compiles calculated variables and sorts them by dependency.
func CompileCalculations(variables []*Variable) ([]*Calculation, error) {
	known := make(map[string]struct{}, len(variables))
	for _, variable := range variables {
		if variable != nil {
			known[variable.Name] = struct{}{}
		}
	}
	calculations := make([]*Calculation, 0)
	byName := make(map[string]*Calculation)
	for _, variable := range variables {
		if variable == nil || !variable.IsCalculated {
			continue
		}
		program, err := expr.Compile(variable.Expression, expr.AllowUndefinedVariables())
		if err != nil {
			return nil, fmt.Errorf("compile variable %s: %w", variable.Name, err)
		}
		item := &Calculation{
			Variable: variable,
			program:  program,
			deps:     expressionDependencies(variable.Expression, known),
		}
		calculations = append(calculations, item)
		byName[variable.Name] = item
	}
	return sortCalculations(calculations, byName)
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

func expressionDependencies(expression string, known map[string]struct{}) []string {
	seen := make(map[string]struct{})
	deps := make([]string, 0)
	for _, name := range identifierPattern.FindAllString(expression, -1) {
		if _, ok := known[name]; !ok {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		deps = append(deps, name)
	}
	return deps
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
			return fmt.Errorf("calculated variable cycle at %s", item.Variable.Name)
		case visited:
			return nil
		}
		state[item.Variable.Name] = visiting
		for _, dep := range item.deps {
			if dependency, ok := byName[dep]; ok {
				if err := visit(dependency); err != nil {
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

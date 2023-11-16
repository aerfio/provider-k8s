package celcheck

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"k8s.io/apiserver/pkg/cel/library"
)

var celEnvOptions = []cel.EnvOption{
	cel.EagerlyValidateDeclarations(true),
	cel.DefaultUTCTimeZone(true),
	cel.HomogeneousAggregateLiterals(),
	cel.ASTValidators(
		cel.ValidateDurationLiterals(),
		cel.ValidateTimestampLiterals(),
		cel.ValidateRegexLiterals(),
		cel.ValidateHomogeneousAggregateLiterals(),
	),
	ext.Bindings(),
	ext.Encoders(),
	ext.Sets(),
	ext.Strings(ext.StringsVersion(2)),
	cel.CrossTypeNumericComparisons(true),
	cel.OptionalTypes(),
	library.URLs(),
	library.Regex(),
	library.Lists(),
	library.Quantity(),
}

var celProgramOptions = []cel.ProgramOption{
	cel.EvalOptions(cel.OptOptimize, cel.OptTrackCost),
}

// slightly adapted https://github.com/undistro/cel-playground/blob/a015ab6d50145af7397bc9e382b23429b57d4c6c/eval/eval.go#L45
// Eval evaluates the cel expression against the given input. Expression must return bool value.
func Eval(exp string, input map[string]any) (bool, error) {
	inputVars := make([]cel.EnvOption, 0, len(input))
	for k := range input {
		inputVars = append(inputVars, cel.Variable(k, cel.DynType))
	}
	env, err := cel.NewEnv(append(celEnvOptions, inputVars...)...)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL env: %s", err)
	}
	ast, issues := env.Compile(exp)
	if issues != nil {
		return false, fmt.Errorf("failed to compile the CEL expression: %s", issues.String())
	}
	prog, err := env.Program(ast, celProgramOptions...)
	if err != nil {
		return false, fmt.Errorf("failed to instantiate CEL program: %s", err)
	}
	val, _, err := prog.Eval(input)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate: %s", err)
	}
	anyBool, err := val.ConvertToNative(reflect.TypeOf(true))
	if err != nil {
		return false, fmt.Errorf("failed to marshal the output to bool: %s", err)
	}

	return anyBool.(bool), nil //nolint:forcetypeassert // it's checked by ConvertToNative
}

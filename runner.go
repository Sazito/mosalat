package mosalat

import (
	"github.com/sazito/mosalat/eval"
	"github.com/sazito/mosalat/parse"
)

func Run(input []string, funcMap, inputMap, outputMap map[string]interface{}) (map[string]interface{}, error) {
	e, err := eval.New(
		funcMap, inputMap, outputMap,
	)
	if err != nil {
		return nil, err
	}
	ast, err := parse.Parse(input, funcMap, inputMap, outputMap)
	if err != nil {
		return nil, err
	}
	return e.Eval(ast)
}

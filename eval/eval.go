package eval

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sync"

	"github.com/sazito/mosalat/parse"
)

func isTrue(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return false, true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true // Struct values are always true.
	default:
		return
	}
	return truth, true
}

type stateMaps struct {
	inputMap  map[string]interface{}
	outputMap map[string]interface{}
	funcMap   map[string]interface{}
}

type Evaluator struct {
	mu    sync.Mutex
	state stateMaps
}

func New(funcMap, inputMap, outputMap map[string]interface{}) (e *Evaluator, err error) {
	e = &Evaluator{
		state: stateMaps{
			inputMap:  inputMap,
			outputMap: outputMap,
			funcMap:   funcMap,
		},
	}
	return
}

func (e *Evaluator) Eval(ast parse.AST) (map[string]interface{}, error) {
	return e.eval(ast.Node)
}

func (e *Evaluator) eval(node parse.Node) (res map[string]interface{}, err error) {
	defer func() {
		e := recover()
		if e != nil {
			switch er := e.(type) {
			case runtime.Error:
				err = er
			case *reflect.ValueError:
				err = er
			default:
				err = fmt.Errorf(fmt.Sprint(er))
			}
		}
	}()
	e.mu.Lock()
	defer e.mu.Unlock()
	res, err = e.evalEngine(node)
	return
}

func (e *Evaluator) evalEngine(node parse.Node) (map[string]interface{}, error) {
	switch n := node.(type) {
	case *parse.EngineNode:
		for _, nr := range n.Rules {
			if err := e.evalRuleNode(&nr); err != nil {
				return nil, err
			}
		}
	case parse.EngineNode:
		for _, nr := range n.Rules {
			if err := e.evalRuleNode(&nr); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("unknown command %T", node)
	}
	return e.state.outputMap, nil
}

func (e *Evaluator) evalRuleNode(node *parse.RuleNode) error {
	shouldRunAction := false
	if node.Condition == nil {
		shouldRunAction = true
	}
	if !shouldRunAction {
		var err error
		shouldRunAction, err = e.evalCondition(node.Condition)
		if err != nil {
			return err
		}
	}
	if shouldRunAction {
		for _, ar := range node.Actions {
			if err := e.evalAction(&ar); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Evaluator) evalCondition(node *parse.ExpressionNode) (bool, error) {
	res, err := e.evalExpression(node.Expression)
	if err != nil {
		return false, err
	}
	if r, ok := isTrue(reflect.ValueOf(res)); ok {
		return r, nil
	} else {
		return false, fmt.Errorf("condition not a bool")
	}
}

func (e *Evaluator) evalAction(node *parse.AssingmentNode) error {
	res, err := e.evalExpression(node.RightExpression)
	if err != nil {
		return err
	}
	if val, ok := e.state.outputMap[node.Variable.Identifier]; ok {
		if reflect.TypeOf(val) != reflect.TypeOf(res) {
			return fmt.Errorf("new variable type is not compatible with the old one")
		}
	}
	e.state.outputMap[node.Variable.Identifier] = res

	return nil
}

func (e *Evaluator) evalExpression(node parse.Node) (interface{}, error) {
	switch n := node.(type) {
	case *parse.ExpressionNode:
		return e.evalExpression(n.Expression)
	case parse.ExpressionNode:
		return e.evalExpression(n.Expression)
	case *parse.NumberNode:
		return e.evalNumber(n)
	case parse.NumberNode:
		return e.evalNumber(&n)
	case *parse.StringNode:
		return e.evalString(n)
	case parse.StringNode:
		return e.evalString(&n)
	case *parse.BoolNode:
		return e.evalBool(n)
	case parse.BoolNode:
		return e.evalBool(&n)
	case *parse.NotNode:
		return e.evalNot(n)
	case parse.NotNode:
		return e.evalNot(&n)
	case *parse.IdentifierNode:
		return e.evalIdentifier(n)
	case parse.IdentifierNode:
		return e.evalIdentifier(&n)
	case *parse.FunctionNode:
		return e.evalFunction(n)
	case parse.FunctionNode:
		return e.evalFunction(&n)
	case *parse.MathExpressionNode:
		return e.evalMathExpression(n)
	case parse.MathExpressionNode:
		return e.evalMathExpression(&n)
	case *parse.ConditionalExpressionNode:
		return e.evalConditionalExpression(n)
	case parse.ConditionalExpressionNode:
		return e.evalConditionalExpression(&n)
	default:
		return nil, fmt.Errorf("unknown command %T", node)
	}
}

func (e *Evaluator) evalNumber(node *parse.NumberNode) (interface{}, error) {
	// if node.IsUint {
	// 	return node.Uint64, nil
	// }
	// if node.IsInt {
	// 	return node.Int64, nil
	// }
	if node.IsFloat {
		return node.Float64, nil
	}
	return nil, fmt.Errorf("unexpected number")
}

func (e *Evaluator) evalString(node *parse.StringNode) (string, error) {
	return node.Text, nil
}

func (e *Evaluator) evalBool(node *parse.BoolNode) (bool, error) {
	return node.IsTrue, nil
}

func (e *Evaluator) evalNot(node *parse.NotNode) (bool, error) {
	res, err := e.evalExpression(node.Expression)
	if err != nil {
		return false, err
	}
	if r, ok := isTrue(reflect.ValueOf(res)); ok {
		return !r, nil
	}
	return false, fmt.Errorf("expression is not a boolean expression")
}

func (e *Evaluator) evalIdentifier(node *parse.IdentifierNode) (interface{}, error) {
	if node.IsInput {
		return e.state.inputMap[node.Identifier], nil
	}
	return e.state.outputMap[node.Identifier], nil
}

func (e *Evaluator) evalFunction(node *parse.FunctionNode) (interface{}, error) {
	f := reflect.ValueOf(e.state.funcMap[node.Function])
	if !f.IsValid() {
		return nil, fmt.Errorf("not a valid function")
	}
	switch f.Kind() {
	case reflect.Func:
		var in []reflect.Value
		for _, n := range node.Args {
			if r, err := e.evalExpression(n.Expression); err == nil {
				in = append(in, reflect.ValueOf(r))
			} else {
				return nil, err
			}
		}
		return f.Call(in)[0].Interface(), nil
	default:
		return nil, fmt.Errorf("not a valid function")
	}
	return nil, fmt.Errorf("not a valid function")
}

func (e *Evaluator) evalMathExpression(node *parse.MathExpressionNode) (interface{}, error) {
	l, err := e.evalExpression(node.LeftExpression)
	if err != nil {
		return nil, err
	}
	lv := reflect.ValueOf(l)
	r, err := e.evalExpression(node.RightExpression)
	if err != nil {
		return nil, err
	}
	rv := reflect.ValueOf(r)
	var rvc, lvc reflect.Value
	// if rv.Type().ConvertibleTo(lv.Type()) {
	// 	rvc = rv.Convert(lv.Type())
	// 	lvc = lv
	// } else if lv.Type().ConvertibleTo(rv.Type()) {
	// 	lvc = lv.Convert(rv.Type())
	// 	rvc = rv
	// } else
	if lv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) && rv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
		lvc = lv.Convert(reflect.TypeOf(float64(0)))
		rvc = rv.Convert(reflect.TypeOf(float64(0)))
	} else {
		return nil, fmt.Errorf("not a valid combination")
	}
	switch node.Identifier {
	case "+":
		switch lvc.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return lvc.Uint() + rvc.Uint(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return lvc.Int() + rvc.Int(), nil
		case reflect.Float64, reflect.Float32:
			return lvc.Float() + rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "-":
		switch lvc.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return lvc.Uint() - rvc.Uint(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return lvc.Int() - rvc.Int(), nil
		case reflect.Float64, reflect.Float32:
			return lvc.Float() - rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "*":
		switch lvc.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return lvc.Uint() * rvc.Uint(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return lvc.Int() * rvc.Int(), nil
		case reflect.Float64, reflect.Float32:
			return lvc.Float() * rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "/":
		switch lvc.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return lvc.Uint() / rvc.Uint(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return lvc.Int() / rvc.Int(), nil
		case reflect.Float64, reflect.Float32:
			return lvc.Float() / rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "%":
		switch lvc.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return lvc.Uint() % rvc.Uint(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return lvc.Int() % rvc.Int(), nil
		case reflect.Float64, reflect.Float32:
			return math.Mod(lvc.Float(), rvc.Float()), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	}
	return false, fmt.Errorf("not a valid operator")
}

func (e *Evaluator) evalConditionalExpression(node *parse.ConditionalExpressionNode) (bool, error) {
	l, err := e.evalExpression(node.LeftExpression)
	if err != nil {
		return false, err
	}
	lv := reflect.ValueOf(l)
	r, err := e.evalExpression(node.RightExpression)
	if err != nil {
		return false, err
	}
	rv := reflect.ValueOf(r)
	switch node.Identifier {
	case ">":
		var rvc, lvc reflect.Value
		if lv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) && rv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
			lvc = lv.Convert(reflect.TypeOf(float64(0)))
			rvc = rv.Convert(reflect.TypeOf(float64(0)))
		} else {
			return false, fmt.Errorf("not a valid combination")
		}
		switch lvc.Kind() {
		case reflect.Float64:
			return lvc.Float() > rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case ">=":
		var rvc, lvc reflect.Value
		if lv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) && rv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
			lvc = lv.Convert(reflect.TypeOf(float64(0)))
			rvc = rv.Convert(reflect.TypeOf(float64(0)))
		} else {
			return false, fmt.Errorf("not a valid combination")
		}
		switch lvc.Kind() {
		case reflect.Float64:
			return lvc.Float() >= rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "<":
		var rvc, lvc reflect.Value
		if lv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) && rv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
			lvc = lv.Convert(reflect.TypeOf(float64(0)))
			rvc = rv.Convert(reflect.TypeOf(float64(0)))
		} else {
			return false, fmt.Errorf("not a valid combination")
		}
		switch lvc.Kind() {
		case reflect.Float64:
			return lvc.Float() < rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "<=":
		var rvc, lvc reflect.Value
		if lv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) && rv.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
			lvc = lv.Convert(reflect.TypeOf(float64(0)))
			rvc = rv.Convert(reflect.TypeOf(float64(0)))
		} else {
			return false, fmt.Errorf("not a valid combination")
		}
		switch lvc.Kind() {
		case reflect.Float64:
			return lvc.Float() <= rvc.Float(), nil
		default:
			return false, fmt.Errorf("not a valid combination")
		}
	case "==":
		return reflect.DeepEqual(l, r), nil
	case "!=":
		return !reflect.DeepEqual(l, r), nil
	case "||":
		if la, lok := isTrue(lv); lok {
			if la {
				return la, nil
			}
			if ra, rok := isTrue(rv); rok {
				if ra {
					return ra, nil
				}
			}
		}
		return false, fmt.Errorf("not a valid conditions")
	case "&&":
		if la, lok := isTrue(lv); lok {
			if !la {
				return la, nil
			}
			if ra, rok := isTrue(rv); rok {
				return ra, nil
			}
		}
		return false, fmt.Errorf("not a valid conditions")
	}
	return false, fmt.Errorf("not a valid operator")
}

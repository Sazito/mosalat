package parse

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

func Parse(input []string, funcMap, inputMap, outputMap map[string]interface{}) (AST, error) {
	parser := newParser(lex(input), funcMap, inputMap, outputMap)
	return parser.Parse()
}

type parser struct {
	lex       *lexer
	funcMap   map[string]interface{}
	inputMap  map[string]interface{}
	outputMap map[string]interface{}
	lookahead [2]item
	peekCount int
}

func newParser(lex *lexer, funcMap, inputMap, outputMap map[string]interface{}) *parser {
	oMap := make(map[string]interface{})
	for k, v := range outputMap {
		oMap[k] = v
	}
	return &parser{
		lex:       lex,
		funcMap:   funcMap,
		inputMap:  inputMap,
		outputMap: oMap,
	}
}

func (p *parser) Parse() (ast AST, err error) {
	// Parsing uses panics to bubble up errors
	defer p.recover(&err)

	ast.Node = p.engine()

	return
}

func (p *parser) nextToken() item {
	return p.lex.nextItem()
}

// next returns the next token.
func (p *parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.lookahead[0] = p.nextToken()
	}
	return p.lookahead[p.peekCount]
}

// backup backs the input stream up one token.
func (p *parser) backup() {
	p.peekCount++
}

// peek returns but does not consume the next token.
func (p *parser) peek() item {
	if p.peekCount > 0 {
		return p.lookahead[p.peekCount-1]
	}
	p.peekCount = 1
	p.lookahead[1] = p.lookahead[0]
	p.lookahead[0] = p.nextToken()
	return p.lookahead[0]
}

// errorf formats the error and terminates processing.
func (p *parser) errorf(format string, args ...interface{}) {
	format = fmt.Sprintf("parser: %s", format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (p *parser) error(err error) {
	p.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (p *parser) expect(expected itemType) item {
	t := p.next()
	if t.typ != expected {
		p.unexpected(t, expected)
	}
	return t
}

// unexpected complains about the token and terminates processing.
func (p *parser) unexpected(tok item, expected ...itemType) {
	expectedStrs := make([]string, len(expected))
	for i := range expected {
		expectedStrs[i] = fmt.Sprintf("%q", expected[i])
	}
	expectedStr := strings.Join(expectedStrs, ",")
	p.errorf("unexpected token %d with value %q at line %d char %d, expected: %s", tok.typ, tok.val, tok.pos.Index, tok.pos.Char, expectedStr)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (p *parser) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
	return
}

var positionZero = Position{
	Index: 0,
	Char:  0,
}

func (p *parser) engine() *EngineNode {
	eng := &EngineNode{
		Position: positionZero,
	}
	for {
		switch p.peek().typ {
		case itemEOF:
			return eng
		case itemLeftRuleDelim:
			p.next()
			r := p.rule()
			eng.Rules = append(eng.Rules, *r)
		default:
			p.unexpected(p.next(), itemLeftRuleDelim)
		}
	}
}

func (p *parser) rule() *RuleNode {
	var cond *ExpressionNode
	var actions []AssingmentNode
	pos := p.lookahead[0].pos
	for {
		switch p.next().typ {
		case itemLeftConditionDelim:
			cond = p.condition()
		case itemLeftActionDelim:
			actions = p.actions()
		case itemRightRuleDelim:
			return &RuleNode{
				Position:  pos,
				Condition: cond,
				Actions:   actions,
			}
		}
	}
}

func (p *parser) condition() *ExpressionNode {
	return p.expression()
}

func (p *parser) actions() []AssingmentNode {
	var actions []AssingmentNode
	for {
		switch p.peek().typ {
		case itemRightActionDelim:
			return actions
		default:
			actions = append(actions, *p.assignment())
		}
	}
}

func (p *parser) assignment() *AssingmentNode {
	v := p.expect(itemVariable)

	_, okI := p.inputMap[v.val]
	_, okF := p.funcMap[v.val]

	if okI {
		p.unexpected(v)
	}
	if okF {
		p.unexpected(v)
	}

	p.outputMap[v.val] = true

	p.expect(itemAssign)
	exp := p.expression()
	return &AssingmentNode{
		Position: v.pos,
		Variable: &VariableNode{
			Position:   v.pos,
			Identifier: v.val,
		},
		RightExpression: exp,
	}
}

func (p *parser) expression() *ExpressionNode {
	n := ExpressionNode{
		Position: p.peek().pos,
	}
	for {
		switch p.peek().typ {
		case itemRightParen, itemRightConditionDelim, itemSeprator:
			p.next()
			return &n
		case itemRightFunctionDelim, itemRightActionDelim:
			return &n
		case itemLeftParen:
			p.next()
			n.Expression = p.expression()
		case itemNot:
			if n.Expression == nil {
				n.Expression = p.not()
			} else {
				n.Expression = p.assignToNode(n.Expression, p.not())
			}
		case itemNumber:
			if n.Expression == nil {
				n.Expression = p.number()
			} else {
				n.Expression = p.assignToNode(n.Expression, p.number())
			}
		case itemBool:
			if n.Expression == nil {
				n.Expression = p.bool()
			} else {
				n.Expression = p.assignToNode(n.Expression, p.bool())
			}
		case itemString:
			if n.Expression == nil {
				n.Expression = p.string()
			} else {
				n.Expression = p.assignToNode(n.Expression, p.string())
			}
		case itemFunction:
			if n.Expression == nil {
				n.Expression = p.function()
			} else {
				n.Expression = p.assignToNode(n.Expression, p.function())
			}
		case itemIdentifier:
			if n.Expression == nil {
				n.Expression = p.identifier()
			} else {
				n.Expression = p.assignToNode(n.Expression, p.identifier())
			}
		case itemAdd, itemMinus, itemDiv, itemMod, itemPow:
			if n.Expression == nil {
				p.unexpected(p.peek())
			} else {
				n.Expression = p.addToPrivNode(n.Expression, p.math())
			}
		case itemAnd, itemOr, itemEquals, itemGreaterEquals, itemLowerEquals, itemLowers, itemGreaters, itemNotEquals:
			if n.Expression == nil {
				p.unexpected(p.peek())
			} else {
				n.Expression = p.addToPrivNode(n.Expression, p.conditional())
			}
		default:
			p.unexpected(p.peek())
		}
	}
}

func (p *parser) expressionAfterCondition() Node {
	var n Node
	for {
		switch p.peek().typ {
		case itemRightParen, itemRightConditionDelim, itemSeprator:
			return n
		case itemRightFunctionDelim, itemRightActionDelim:
			return n
		case itemLeftParen:
			p.next()
			n = p.expression()
		case itemNot:
			if n == nil {
				n = p.not()
			} else {
				n = p.assignToNode(n, p.not())
			}
		case itemNumber:
			if n == nil {
				n = p.number()
			} else {
				n = p.assignToNode(n, p.number())
			}
		case itemBool:
			if n == nil {
				n = p.bool()
			} else {
				n = p.assignToNode(n, p.bool())
			}
		case itemString:
			if n == nil {
				n = p.string()
			} else {
				n = p.assignToNode(n, p.string())
			}
		case itemFunction:
			if n == nil {
				n = p.function()
			} else {
				n = p.assignToNode(n, p.function())
			}
		case itemIdentifier:
			if n == nil {
				n = p.identifier()
			} else {
				n = p.assignToNode(n, p.identifier())
			}
		case itemAdd, itemMinus, itemDiv, itemMod, itemPow:
			if n == nil {
				p.unexpected(p.peek())
			} else {
				n = p.addToPrivNode(n, p.math())
			}
		case itemAnd, itemOr, itemEquals, itemGreaterEquals, itemLowerEquals, itemLowers, itemGreaters, itemNotEquals:
			return n
		default:
			p.unexpected(p.peek())
		}
	}
}

func (p *parser) expWithoutDepth() Node {
	var n Node
	switch p.peek().typ {
	case itemRightParen, itemRightConditionDelim, itemRightActionDelim, itemSeprator, itemRightFunctionDelim:
	case itemNot:
		n = p.not()
	case itemNumber:
		n = p.number()
	case itemBool:
		n = p.bool()
	case itemString:
		n = p.string()
	case itemFunction:
		n = p.function()
	case itemIdentifier:
		n = p.identifier()
	case itemAdd, itemMinus, itemDiv, itemMod, itemPow:
	case itemAnd, itemOr, itemEquals, itemGreaterEquals, itemLowerEquals, itemLowers, itemGreaters, itemNotEquals:
	default:
		p.unexpected(p.peek())
	}
	return n
}

func (p *parser) expCall() Node {
	var exp Node
	dwa := p.peek()
	if dwa.typ == itemLeftParen {
		p.next()
		exp = p.expression()
	} else {
		exp = p.expWithoutDepth()
	}
	return exp
}

func (p *parser) assignToNode(parent, new Node) Node {
	switch n := parent.(type) {
	case *MathExpressionNode:
		n.RightExpression = new
	case *ConditionalExpressionNode:
		n.RightExpression = new
	default:
		p.errorf("wrong previous node")
	}
	return parent
}

func (p *parser) addToPrivNode(parent, new Node) Node {
	var n Node
	switch parent.(type) {
	case *MathExpressionNode, *ConditionalExpressionNode:
		n = mergeExp(parent, new)
	case *NotNode, *NumberNode, *BoolNode, *StringNode, *FunctionNode, *IdentifierNode, *ExpressionNode:
		n = assignLeftExp(parent, new)
	}
	return n
}

func assignLeftExp(parent, new Node) Node {
	switch n := new.(type) {
	case *MathExpressionNode:
		n.LeftExpression = parent
	case *ConditionalExpressionNode:
		n.LeftExpression = parent
	}
	return new
}

func mergeExp(parent, new Node) Node {
	switch n := parent.(type) {
	case *MathExpressionNode:
		switch nn := new.(type) {
		case *MathExpressionNode:
			if nn.IsMod {
				nn.LeftExpression = n
				return new
			}
			if nn.IsAditive {
				if n.IsMod {
					nn.LeftExpression = n.RightExpression
					n.RightExpression = nn
					return parent
				}
				nn.LeftExpression = n
				return new
			}
			if n.IsProductive {
				nn.LeftExpression = n
				return new
			}
			nn.LeftExpression = n.RightExpression
			n.RightExpression = nn
			return parent
		case *ConditionalExpressionNode:
			nn.LeftExpression = n
			return new
		}
	case *ConditionalExpressionNode:
		switch nn := new.(type) {
		case *MathExpressionNode:
			nn.LeftExpression = n.RightExpression
			n.RightExpression = nn
			return parent
		case *ConditionalExpressionNode:
			if nn.IsBooleanBase {
				nn.LeftExpression = n
				return new
			}
			nn.LeftExpression = n.RightExpression
			n.RightExpression = nn
			return parent
		}
	}
	return parent
}

func (p *parser) not() *NotNode {
	v := p.expect(itemNot)
	exp := p.expCall()
	return &NotNode{
		Position:   v.pos,
		Expression: exp,
	}
}

func (p *parser) number() *NumberNode {
	v := p.expect(itemNumber)
	n := NumberNode{
		Position: v.pos,
		Text:     v.val,
	}
	// Do integer test first so we get 0x123 etc.
	u, err := strconv.ParseUint(v.val, 0, 64) // will fail for -0; fixed below.
	if err == nil {
		n.IsUint = true
		n.Uint64 = u
	}
	i, err := strconv.ParseInt(v.val, 0, 64)
	if err == nil {
		n.IsInt = true
		n.Int64 = i
		if i == 0 {
			n.IsUint = true // in case of -0.
			n.Uint64 = u
		}
	}
	// If an integer extraction succeeded, promote the float.
	if n.IsInt {
		n.IsFloat = true
		n.Float64 = float64(n.Int64)
	} else if n.IsUint {
		n.IsFloat = true
		n.Float64 = float64(n.Uint64)
	} else {
		f, err := strconv.ParseFloat(v.val, 64)
		if err == nil {
			// If we parsed it as a float but it looks like an integer,
			// it's a huge number too large to fit in an int. Reject it.
			if !strings.ContainsAny(v.val, ".eEpP") {
				p.error(fmt.Errorf("integer overflow: %q", v.val))
			}
			n.IsFloat = true
			n.Float64 = f
			// If a floating-point extraction succeeded, extract the int if needed.
			if !n.IsInt && float64(int64(f)) == f {
				n.IsInt = true
				n.Int64 = int64(f)
			}
			if !n.IsUint && float64(uint64(f)) == f {
				n.IsUint = true
				n.Uint64 = uint64(f)
			}
		}
	}
	if !n.IsInt && !n.IsUint && !n.IsFloat {
		p.error(fmt.Errorf("illegal number syntax: %q", v.val))
	}
	return &n
}

func (p *parser) bool() *BoolNode {
	v := p.expect(itemBool)
	return &BoolNode{
		Position: v.pos,
		IsTrue:   v.val == "true",
	}
}

func (p *parser) string() *StringNode {
	v := p.expect(itemString)
	s, err := strconv.Unquote(v.val)
	if err != nil {
		p.error(err)
	}
	return &StringNode{
		Position: v.pos,
		RawText:  v.val,
		Text:     s,
	}
}

func (p *parser) math() *MathExpressionNode {
	n := p.next()
	exp := p.expCall()
	return &MathExpressionNode{
		IsMod:           n.typ == itemMod,
		IsAditive:       n.typ == itemMinus || n.typ == itemAdd,
		IsProductive:    n.typ == itemPow || n.typ == itemDiv,
		Position:        n.pos,
		Identifier:      n.val,
		Type:            n.typ,
		RightExpression: exp,
	}
}

func (p *parser) conditional() *ConditionalExpressionNode {
	n := p.next()
	exp := p.expressionAfterCondition()
	return &ConditionalExpressionNode{
		IsBooleanBase:   n.typ == itemOr || n.typ == itemAnd,
		IsDiffBase:      n.typ == itemEquals || n.typ == itemNotEquals || n.typ == itemGreaterEquals || n.typ == itemLowerEquals || n.typ == itemGreaters || n.typ == itemLowers,
		Position:        n.pos,
		Identifier:      n.val,
		Type:            n.typ,
		RightExpression: exp,
	}
}

func (p *parser) function() *FunctionNode {
	v := p.expect(itemFunction)
	if _, ok := p.funcMap[v.val]; !ok {
		p.unexpected(v)
	}
	if _, ok := p.inputMap[v.val]; ok {
		p.unexpected(v)
	}
	if _, ok := p.outputMap[v.val]; ok {
		p.unexpected(v)
	}

	p.expect(itemLeftFunctionDelim)
	n := FunctionNode{
		Position: v.pos,
		Function: v.val,
	}
	for {
		switch p.peek().typ {
		case itemRightFunctionDelim:
			p.next()
			return &n
		default:
			n.Args = append(n.Args, *p.expression())
		}
	}
}

func (p *parser) identifier() *IdentifierNode {
	v := p.expect(itemIdentifier)

	_, okO := p.outputMap[v.val]
	_, okI := p.inputMap[v.val]
	_, okF := p.funcMap[v.val]
	if !okI && !okO {
		p.unexpected(v)
	}
	if okI && okO {
		p.unexpected(v)
	}
	if okF {
		p.unexpected(v)
	}
	return &IdentifierNode{
		Position:   v.pos,
		Identifier: v.val,
		IsInput:    okI,
	}
}

package parse

import "fmt"

type Node interface {
	Pos() Position
	String() string
}

type Position struct {
	Index int
	Char  int
}

func (p Position) Pos() Position {
	return p
}

type AST struct {
	Node
}

type EngineNode struct {
	Position
	Rules []RuleNode
}

func (n EngineNode) String() string {
	s := "->EngineNode\n"
	for _, rn := range n.Rules {
		s += fmt.Sprintf("%s", rn)
	}
	s += "<-EngineNode\n"
	return s
}

type RuleNode struct {
	Position
	Condition *ExpressionNode
	Actions   []AssingmentNode
}

func (n RuleNode) String() string {
	s := "->RuleNode\n\n"
	s += fmt.Sprintf("Condition\n%s", n.Condition)
	s += fmt.Sprintf("\nActions->\n")
	for _, an := range n.Actions {
		s += fmt.Sprintf("%s", an)
	}
	s += fmt.Sprintf("<-Actions\n\n")
	s += "<-RuleNode\n"
	return s
}

type ExpressionNode struct {
	Position
	Expression Node
}

func (n ExpressionNode) String() string {
	s := "->ExpressionNode\n"
	s += fmt.Sprintf("Expression\n%s", n.Expression)
	s += "<-ExpressionNode\n"
	return s
}

type FunctionNode struct {
	Position
	Function string
	Args     []ExpressionNode
}

func (n FunctionNode) String() string {
	s := "->FunctionNode\n"
	s += fmt.Sprintf("Function %s\n", n.Function)
	s += fmt.Sprintf("Args->\n")
	for _, an := range n.Args {
		s += fmt.Sprintf("%s", an)
	}
	s += fmt.Sprintf("<-Args\n")
	s += "<-FunctionNode\n"
	return s
}

type NumberNode struct {
	Position
	IsInt   bool
	IsUint  bool
	IsFloat bool
	Int64   int64
	Uint64  uint64
	Float64 float64
	Text    string
}

func (n NumberNode) String() string {
	s := "->NumberNode "
	s += fmt.Sprintf("%s\n", n.Text)
	return s
}

type StringNode struct {
	Position
	RawText string
	Text    string
}

func (n StringNode) String() string {
	s := "->StringNode "
	s += fmt.Sprintf("%s\n", n.Text)
	return s
}

type BoolNode struct {
	Position
	IsTrue bool
}

func (n BoolNode) String() string {
	s := "->BoolNode "
	s += fmt.Sprintf("%v\n", n.IsTrue)
	return s
}

type NotNode struct {
	Position
	Expression Node
}

func (n NotNode) String() string {
	s := "->NotNode\n"
	s += fmt.Sprintf("Expression\n%s", n.Expression)
	s += "<-NotNode\n"
	return s
}

type AssingmentNode struct {
	Position
	Variable        *VariableNode
	RightExpression *ExpressionNode
}

func (n AssingmentNode) String() string {
	s := "->AssingmentNode\n"
	s += fmt.Sprintf("Variable\n%s", n.Variable)
	s += fmt.Sprintf("RightExpression\n%s", n.RightExpression)
	s += "<-AssingmentNode\n"
	return s
}

type VariableNode struct {
	Position
	Identifier string
}

func (n VariableNode) String() string {
	s := "->VariableNode "
	s += fmt.Sprintf("%s\n", n.Identifier)
	return s
}

type IdentifierNode struct {
	Position
	Identifier string
	IsInput    bool
}

func (n IdentifierNode) String() string {
	s := "->IdentifierNode "
	s += fmt.Sprintf("input:%v %s\n", n.IsInput, n.Identifier)
	return s
}

type MathExpressionNode struct {
	Position
	Identifier      string
	IsAditive       bool
	IsProductive    bool
	IsMod           bool
	Type            itemType
	LeftExpression  Node
	RightExpression Node
}

func (n MathExpressionNode) String() string {
	s := fmt.Sprintf("->MathExpressionNode %s\n", n.Identifier)
	s += fmt.Sprintf("LeftExpression\n%s", n.LeftExpression)
	s += fmt.Sprintf("RightExpression\n%s", n.RightExpression)
	s += "<-MathExpressionNode\n"
	return s
}

type ConditionalExpressionNode struct {
	Position
	Identifier      string
	IsBooleanBase   bool
	IsDiffBase      bool
	Type            itemType
	LeftExpression  Node
	RightExpression Node
}

func (n ConditionalExpressionNode) String() string {
	s := fmt.Sprintf("->ConditionalExpressionNode %s\n", n.Identifier)
	s += fmt.Sprintf("LeftExpression\n%s", n.LeftExpression)
	s += fmt.Sprintf("RightExpression\n%s", n.RightExpression)
	s += "<-ConditionalExpressionNode\n"
	return s
}

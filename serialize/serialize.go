package serialize

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"

	"github.com/sazito/mosalat/parse"
)

func init() {
	gob.Register(parse.AST{})

	gob.Register(parse.Position{})

	gob.Register(parse.EngineNode{})
	gob.Register(parse.RuleNode{})
	gob.Register(parse.ExpressionNode{})
	gob.Register(parse.FunctionNode{})
	gob.Register(parse.NumberNode{})
	gob.Register(parse.StringNode{})
	gob.Register(parse.BoolNode{})
	gob.Register(parse.NotNode{})
	gob.Register(parse.AssingmentNode{})
	gob.Register(parse.VariableNode{})
	gob.Register(parse.IdentifierNode{})
	gob.Register(parse.MathExpressionNode{})
	gob.Register(parse.ConditionalExpressionNode{})
}

func SerializeAST(m parse.AST) (string, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func DeSerializeToAST(str string) (parse.AST, error) {
	m := parse.AST{}
	by, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return parse.AST{}, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&m)
	if err != nil {
		return parse.AST{}, err
	}
	return m, nil
}

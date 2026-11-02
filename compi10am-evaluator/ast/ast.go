// Package ast define los nodos del árbol de sintaxis abstracta que construye
// el parser.
package ast

import (
	"bytes"
	"compi/token"
	"strings"
)

// Node es la interfaz común de los nodos del AST. TokenLiteral devuelve el
// literal del token inicial; String reconstruye el nodo para inspeccionarlo.
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement identifica los nodos que se pueden usar como sentencias.
type Statement interface {
	Node
	statementNode()
}

// Expression identifica los nodos que se pueden usar como expresiones.
type Expression interface {
	Node
	expressionNode()
}

// Program es la raíz del AST y contiene las sentencias de la entrada.
// String concatena su representación textual en el orden original.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// ====================================================================
// Sentencias
// ====================================================================

// LetStatement: let <nombre> = <valor>;
type LetStatement struct {
	Token token.Token // el token LET
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode() {}
func (ls *LetStatement) TokenLiteral() string {
	return ls.Token.Literal
}
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

// ReturnStatement: return <valor>;
type ReturnStatement struct {
	Token       token.Token // el token RETURN
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode() {}
func (rs *ReturnStatement) TokenLiteral() string {
	return rs.Token.Literal
}
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	out.WriteString(";")

	return out.String()
}

// BreakStatement: break;  (sale del ciclo actual)
type BreakStatement struct {
	Token token.Token // el token BREAK
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return bs.Token.Literal + ";" }

// ContinueStatement: continue;  (salta a la siguiente iteración)
type ContinueStatement struct {
	Token token.Token // el token CONTINUE
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return cs.Token.Literal + ";" }

// ExpressionStatement envuelve una expresión usada como sentencia.
// Ejemplo: la línea "x + 10;"
type ExpressionStatement struct {
	Token      token.Token // primer token de la expresión
	Expression Expression
}

func (es *ExpressionStatement) statementNode() {}
func (es *ExpressionStatement) TokenLiteral() string {
	return es.Token.Literal
}
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement: lista de sentencias entre llaves { ... }
type BlockStatement struct {
	Token      token.Token // el token {
	Statements []Statement
}

func (bs *BlockStatement) statementNode() {}
func (bs *BlockStatement) TokenLiteral() string {
	return bs.Token.Literal
}
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// ====================================================================
// Literales e identificadores
// ====================================================================

// Identifier: nombre de una variable o función (x, suma, ...)
type Identifier struct {
	Token token.Token // el token IDENT
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) TokenLiteral() string {
	return i.Token.Literal
}
func (i *Identifier) String() string {
	return i.Value
}

// IntegerLiteral: un número entero (5, 42, ...)
type IntegerLiteral struct {
	Token token.Token // el token INT
	Value int64
}

func (il *IntegerLiteral) expressionNode() {}
func (il *IntegerLiteral) TokenLiteral() string {
	return il.Token.Literal
}
func (il *IntegerLiteral) String() string {
	return il.Token.Literal
}

// StringLiteral: una cadena de texto ("hola")
type StringLiteral struct {
	Token token.Token // el token STRING
	Value string
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) TokenLiteral() string {
	return sl.Token.Literal
}
func (sl *StringLiteral) String() string {
	return "\"" + sl.Value + "\""
}

// BooleanStatement permanece por compatibilidad; el parser construye Boolean
// para los literales true y false.
type BooleanStatement struct {
	Token token.Token
	Value bool
}

func (b *BooleanStatement) expressionNode() {}
func (b *BooleanStatement) TokenLiteral() string {
	return b.Token.Literal
}
func (b *BooleanStatement) String() string {
	return b.Token.Literal
}

// Boolean: literal true o false
type Boolean struct {
	Token token.Token // El token TRUE o FALSE
	Value bool        // true o false como valor real de Go
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// NullLiteral: el valor null
type NullLiteral struct {
	Token token.Token // el token NULL
}

func (n *NullLiteral) expressionNode()      {}
func (n *NullLiteral) TokenLiteral() string { return n.Token.Literal }
func (n *NullLiteral) String() string       { return n.Token.Literal }

// ArrayLiteral: [1, 2, 3]
type ArrayLiteral struct {
	Token    token.Token // el token [
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// HashPair es una pareja clave: valor dentro de un HashLiteral.
type HashPair struct {
	Key   Expression
	Value Expression
}

// HashLiteral guarda pares clave-valor en el orden en que aparecen en la entrada.
type HashLiteral struct {
	Token token.Token // el token {
	Pairs []HashPair
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	pairs := []string{}
	for _, pair := range hl.Pairs {
		pairs = append(pairs, pair.Key.String()+": "+pair.Value.String())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// FunctionLiteral: function(x, y) { ... }
type FunctionLiteral struct {
	Token      token.Token // el token FUNCTION
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode() {}
func (fl *FunctionLiteral) TokenLiteral() string {
	return fl.Token.Literal
}
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	out.WriteString(fl.TokenLiteral())
	out.WriteString(fl.ParametersString())
	out.WriteString(" ")
	out.WriteString(fl.Body.String())

	return out.String()
}

// ParametersString imprime solo la lista de parámetros: (x, y)
func (fl *FunctionLiteral) ParametersString() string {
	var out bytes.Buffer

	out.WriteString("(")
	for i, p := range fl.Parameters {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(p.String())
	}
	out.WriteString(")")

	return out.String()
}

// ====================================================================
// Expresiones compuestas
// ====================================================================

// PrefixExpression: <operador><expresión>, por ejemplo -5 o !true
type PrefixExpression struct {
	Token    token.Token // el token prefijo, p. ej. ! o -
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode() {}
func (pe *PrefixExpression) TokenLiteral() string {
	return pe.Token.Literal
}
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

// InfixExpression: <izquierda> <operador> <derecha>, por ejemplo 5 + 3.
// Se imprime entre paréntesis para que se vea la precedencia aplicada.
type InfixExpression struct {
	Token    token.Token // el token operador, p. ej. +, -, *, /
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode() {}
func (ie *InfixExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

// AssignExpression: <nombre> = <valor>, reasigna una variable existente.
// Ejemplo: i = i + 1
type AssignExpression struct {
	Token token.Token // el token =
	Name  *Identifier
	Value Expression
}

func (ae *AssignExpression) expressionNode()      {}
func (ae *AssignExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpression) String() string {
	return ae.Name.String() + " = " + ae.Value.String()
}

// IndexExpression: <expresión>[<índice>], por ejemplo arr[0]
type IndexExpression struct {
	Token token.Token // el token [
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// IfExpression representa una rama if con su alternativa opcional.
// Cada elif se guarda como otro IfExpression dentro de Alternative.
type IfExpression struct {
	Token       token.Token // el token IF (o ELIF si es anidado)
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode() {}
func (ie *IfExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

// CallExpression: <función>(<argumentos>), por ejemplo add(1, 2)
type CallExpression struct {
	Token     token.Token // el token (
	Function  Expression  // Identifier o FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}
func (ce *CallExpression) TokenLiteral() string {
	return ce.Token.Literal
}
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

// WhileExpression: while (cond) { ... }
type WhileExpression struct {
	Token     token.Token // el token WHILE
	Condition Expression
	Body      *BlockStatement
}

func (we *WhileExpression) expressionNode() {}
func (we *WhileExpression) TokenLiteral() string {
	return we.Token.Literal
}
func (we *WhileExpression) String() string {
	var out bytes.Buffer

	out.WriteString("while")
	out.WriteString(we.Condition.String())
	out.WriteString(" ")
	out.WriteString(we.Body.String())

	return out.String()
}

// ForExpression: for (<inicio>; <condición>; <paso>) { ... }
// Las tres partes son opcionales: for (;;) { ... }
type ForExpression struct {
	Token       token.Token // el token FOR
	Initializer Statement
	Condition   Expression
	Post        Statement
	Body        *BlockStatement
}

func (fe *ForExpression) expressionNode() {}
func (fe *ForExpression) TokenLiteral() string {
	return fe.Token.Literal
}
func (fe *ForExpression) String() string {
	var out bytes.Buffer

	out.WriteString("for")
	out.WriteString("(")
	if fe.Initializer != nil {
		// LetStatement ya termina en ";", se quita para no duplicarlo
		out.WriteString(strings.TrimSuffix(fe.Initializer.String(), ";"))
	}
	out.WriteString("; ")
	if fe.Condition != nil {
		out.WriteString(fe.Condition.String())
	}
	out.WriteString("; ")
	if fe.Post != nil {
		out.WriteString(strings.TrimSuffix(fe.Post.String(), ";"))
	}
	out.WriteString(") ")
	out.WriteString(fe.Body.String())

	return out.String()
}

// Package parser construye el AST a partir de los tokens del lexer.
// Usa un parser Pratt: las funciones prefijas e infijas y las precedencias
// determinan cómo se agrupan las expresiones.
package parser

import (
	"compi/ast"
	"compi/lexer"
	"compi/token"
	"fmt"
	"strconv"
)

// Niveles de precedencia, de menor a mayor. Un nivel más alto agrupa antes.
const (
	_ int = iota
	LOWEST
	ASSIGN      // x = 5
	LOGICAL_OR  // ||
	LOGICAL_AND // &&
	EQUALS      // == o !=
	LESSGREATER // >, <, >= o <=
	SUM         // + o -
	PRODUCT     // *, / o %
	POWER       // ** o ^
	PREFIX      // -X o !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

// precedences asocia cada operador infijo con su nivel de precedencia.
var precedences = map[token.TokenType]int{
	token.ASSIGN:   ASSIGN,
	token.OR:       LOGICAL_OR,
	token.AND:      LOGICAL_AND,
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.LTE:      LESSGREATER,
	token.GTE:      LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.MODULO:   PRODUCT,
	token.POW:      POWER,
	token.POW2:     POWER,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
}

type (
	// prefixParseFn analiza un token que inicia una expresión.
	prefixParseFn func() ast.Expression

	// infixParseFn extiende una expresión izquierda con el operador actual.
	infixParseFn func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer // El lexer que nos da los tokens
	errors []string     // Errores acumulados durante el parseo

	curToken  token.Token // El token actual que estamos analizando
	peekToken token.Token // El siguiente token (para mirar hacia adelante)

	// Cada token se asocia con la función que analiza su posición.
	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

// New crea un parser y registra las funciones de parseo de cada token.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// Tokens que pueden iniciar una expresión
	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.NULL, p.parseNull)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.WHILE, p.parseWhileExpression)
	p.registerPrefix(token.FOR, p.parseForExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)

	// Tokens que aparecen entre dos expresiones
	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	for _, tt := range []token.TokenType{
		token.PLUS, token.MINUS, token.SLASH, token.ASTERISK, token.MODULO,
		token.POW, token.POW2,
		token.EQ, token.NOT_EQ, token.LT, token.GT, token.LTE, token.GTE,
		token.AND, token.OR,
	} {
		p.registerInfix(tt, p.parseInfixExpression)
	}
	p.registerInfix(token.ASSIGN, p.parseAssignExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	// Mantiene un token actual y otro de anticipación.
	p.nextToken()
	p.nextToken()

	return p
}

// ====================================================================
// Utilidades
// ====================================================================

// Errors retorna los errores encontrados durante el parseo.
func (p *Parser) Errors() []string {
	return p.errors
}

// peekError registra un error cuando el siguiente token no es el esperado.
func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// noPrefixParseFnError registra un error cuando un token no puede
// iniciar una expresión (p. ej. una línea que empieza con "*").
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// nextToken avanza una posición en la secuencia de tokens.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek avanza solo si el siguiente token es del tipo esperado;
// si no, registra un error y retorna false.
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekPrecedence retorna la precedencia del siguiente token.
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curPrecedence retorna la precedencia del token actual.
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// ====================================================================
// Programa y sentencias
// ====================================================================

// ParseProgram analiza sentencias hasta EOF y devuelve la raíz del AST.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement decide qué tipo de sentencia parsear según el token actual.
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement: let <identificador> = <expresión>;
func (p *Parser) parseLetStatement() ast.Statement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	// Saltamos el "=" para quedar en el primer token del valor
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// parseReturnStatement: return <expresión>;
func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseBreakStatement: break;
func (p *Parser) parseBreakStatement() ast.Statement {
	stmt := &ast.BreakStatement{Token: p.curToken}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// parseContinueStatement: continue;
func (p *Parser) parseContinueStatement() ast.Statement {
	stmt := &ast.ContinueStatement{Token: p.curToken}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// parseExpressionStatement: una expresión suelta, con ";" opcional.
func (p *Parser) parseExpressionStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseBlockStatement parsea sentencias hasta encontrar "}".
// Se llama con curToken en "{".
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	if !p.curTokenIs(token.RBRACE) {
		p.errors = append(p.errors, "expected } to close block, got EOF instead")
	}

	return block
}

// ====================================================================
// Expresiones (núcleo del Pratt parser)
// ====================================================================

// parseExpression empieza con el token actual y añade operadores mientras
// la precedencia del siguiente sea mayor que el mínimo recibido.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()
	if leftExp == nil {
		// El error ya quedó registrado por la función prefija
		return nil
	}

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// parsePrefixExpression: -x o !x
func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)
	if expression.Right == nil {
		return nil
	}

	return expression
}

// parseInfixExpression: <izquierda> <op> <derecha>
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	// Reducir la precedencia hace que 2 ** 3 ** 2 se agrupe como
	// 2 ** (3 ** 2). También se aplica al operador ^.
	if p.curTokenIs(token.POW) || p.curTokenIs(token.POW2) {
		precedence--
	}
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	if expression.Right == nil {
		return nil
	}

	return expression
}

// parseAssignExpression acepta solo identificadores a la izquierda.
// La asignación se agrupa por la derecha: a = b = 5 equivale a a = (b = 5).
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	name, ok := left.(*ast.Identifier)
	if !ok {
		msg := fmt.Sprintf("cannot assign to %s, expected an identifier", left.String())
		p.errors = append(p.errors, msg)
		return nil
	}

	expression := &ast.AssignExpression{Token: p.curToken, Name: name}

	p.nextToken()
	expression.Value = p.parseExpression(ASSIGN - 1)
	if expression.Value == nil {
		return nil
	}

	return expression
}

// parseGroupedExpression: ( <expresión> )
// Los paréntesis solo alteran la precedencia; no generan nodo propio.
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// ====================================================================
// Literales
// ====================================================================

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value

	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNull() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

// parseArrayLiteral: [elem1, elem2, ...]
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	if array.Elements == nil {
		return nil
	}
	return array
}

// parseHashLiteral: {clave: valor, clave: valor, ...}
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken, Pairs: []ast.HashPair{}}

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)
		if key == nil {
			return nil
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}

		hash.Pairs = append(hash.Pairs, ast.HashPair{Key: key, Value: value})

		// Después de cada pareja debe venir "," o "}"
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

// parseFunctionLiteral: function(<parámetros>) { <cuerpo> }
func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()
	if lit.Parameters == nil {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseFunctionParameters: (x, y, z). Cada parámetro debe ser un identificador.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	identifiers = append(identifiers, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		identifiers = append(identifiers, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// ====================================================================
// Llamadas e índices
// ====================================================================

// parseCallExpression: <función>(<argumentos>)
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	if exp.Arguments == nil {
		return nil
	}
	return exp
}

// parseIndexExpression: <expresión>[<índice>]
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if exp.Index == nil {
		return nil
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

// parseExpressionList parsea expresiones separadas por coma hasta el
// token de cierre "end". Se usa para argumentos de llamadas y arreglos.
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// ====================================================================
// Control de flujo
// ====================================================================

// parseIfExpression: if (cond) { ... } [elif (cond) { ... }]* [else { ... }]
// También se usa para cada "elif", que se trata como un if anidado.
func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)
	if expression.Condition == nil {
		return nil
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Consequence = p.parseBlockStatement()

	switch {
	case p.peekTokenIs(token.ELIF):
		// elif (cond) { ... } se convierte en: else { if (cond) { ... } }
		p.nextToken()
		elifToken := p.curToken
		nested := p.parseIfExpression()
		if nested == nil {
			return nil
		}
		expression.Alternative = &ast.BlockStatement{
			Token: elifToken,
			Statements: []ast.Statement{
				&ast.ExpressionStatement{Token: elifToken, Expression: nested},
			},
		}

	case p.peekTokenIs(token.ELSE):
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

// parseWhileExpression: while (cond) { ... }
func (p *Parser) parseWhileExpression() ast.Expression {
	expression := &ast.WhileExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)
	if expression.Condition == nil {
		return nil
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Body = p.parseBlockStatement()

	return expression
}

// parseForExpression: for (<inicio>; <condición>; <paso>) { ... }
// Ejemplo: for (let i = 0; i < 10; i = i + 1) { ... }
// Cualquiera de las tres partes puede omitirse: for (;;) { ... }
func (p *Parser) parseForExpression() ast.Expression {
	expression := &ast.ForExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// 1. Inicialización (termina en ";")
	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		expression.Initializer = p.parseStatement()
		if expression.Initializer == nil {
			return nil
		}
		// El inicializador debe dejar el parser sobre su ";" final.
		if !p.curTokenIs(token.SEMICOLON) {
			p.peekError(token.SEMICOLON)
			return nil
		}
	}

	// 2. Condición (termina en ";")
	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		expression.Condition = p.parseExpression(LOWEST)
		if expression.Condition == nil {
			return nil
		}
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}

	// 3. Paso / actualización (termina en ")")
	p.nextToken()
	if !p.curTokenIs(token.RPAREN) {
		postToken := p.curToken
		post := p.parseExpression(LOWEST)
		if post == nil {
			return nil
		}
		expression.Post = &ast.ExpressionStatement{Token: postToken, Expression: post}
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}

	// 4. Cuerpo
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Body = p.parseBlockStatement()

	return expression
}

// Package token define los tipos y literales que produce el lexer.
package token

// TokenType identifica la categoría de un token, como INT o +.
type TokenType string

// Token contiene el tipo y el literal de un elemento reconocido por el lexer.
// Para las cadenas, Literal contiene el texto sin las comillas.
type Token struct {
	Type    TokenType
	Literal string
}

const (
	// Tokens especiales
	ILLEGAL = "ILLEGAL" // carácter que el lexer no reconoce
	EOF     = "EOF"     // fin de la entrada

	// Identificadores + literales
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	STRING = "STRING" // "foobar"

	// Operadores aritméticos y de asignación
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	MODULO   = "%"
	POW      = "**" // potencia estilo Python: 2 ** 3
	POW2     = "^"  // potencia alternativa: 2 ^ 3

	// Operadores de comparación
	LT     = "<"
	GT     = ">"
	LTE    = "<="
	GTE    = ">="
	EQ     = "=="
	NOT_EQ = "!="

	// Operadores lógicos
	AND = "&&"
	OR  = "||"

	// Delimitadores
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Palabras clave
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELIF     = "ELIF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	FOR      = "FOR"
	WHILE    = "WHILE"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	NULL     = "NULL"
)

// Keywords asocia las palabras reservadas con sus tipos de token.
var Keywords = map[string]TokenType{
	"function": FUNCTION,
	"let":      LET,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"elif":     ELIF,
	"else":     ELSE,
	"return":   RETURN,
	"for":      FOR,
	"while":    WHILE,
	"break":    BREAK,
	"continue": CONTINUE,
	"null":     NULL,
}

// LookupIdent decide si una palabra es una palabra clave o un
// identificador definido por el usuario.
func LookupIdent(ident string) TokenType {
	if tok, ok := Keywords[ident]; ok {
		return tok
	}
	return IDENT
}

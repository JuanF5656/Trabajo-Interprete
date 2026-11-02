// Package lexer convierte el texto de entrada en tokens para el parser.
package lexer

import "compi/token"

// Lexer recorre la entrada byte a byte. Los identificadores admiten letras
// ASCII, dígitos después del primer carácter y guiones bajos.
type Lexer struct {
	input        string
	position     int  // posición actual en la entrada (apunta a ch)
	readPosition int  // siguiente posición a leer (después de ch)
	ch           byte // carácter que se está examinando
}

// New crea un lexer listo para leer el primer token.
func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar avanza un byte. Al terminar la entrada, ch vale 0 (EOF).
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

// NextToken devuelve el siguiente token, incluido ILLEGAL para bytes no
// reconocidos. Después del último token devuelve EOF.
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	switch l.ch {
	// Algunos operadores tienen versiones de uno y dos caracteres.
	case '=':
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.EQ)
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.NOT_EQ)
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case '*':
		if l.peekChar() == '*' {
			tok = l.makeTwoCharToken(token.POW)
		} else {
			tok = newToken(token.ASTERISK, l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.LTE)
		} else {
			tok = newToken(token.LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.GTE)
		} else {
			tok = newToken(token.GT, l.ch)
		}
	// Operadores lógicos: solo existen en su forma doble (&& y ||).
	// Un & o | aislado es ILLEGAL.
	case '&':
		if l.peekChar() == '&' {
			tok = l.makeTwoCharToken(token.AND)
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			tok = l.makeTwoCharToken(token.OR)
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}

	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '/':
		tok = newToken(token.SLASH, l.ch)
	case '%':
		tok = newToken(token.MODULO, l.ch)
	case '^':
		tok = newToken(token.POW2, l.ch)

	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case ':':
		tok = newToken(token.COLON, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)

	// Las cadenas terminan en la siguiente comilla doble; no hay escapes.
	case '"':
		str, ok := l.readString()
		if ok {
			tok = token.Token{Type: token.STRING, Literal: str}
		} else {
			// Una cadena sin comilla de cierre es un token ILLEGAL.
			tok = token.Token{Type: token.ILLEGAL, Literal: "\"" + str}
		}

	case 0:
		tok.Literal = ""
		tok.Type = token.EOF

	default:
		if isLetter(l.ch) {
			// readIdentifier ya avanzó hasta el byte posterior al nombre.
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			tok.Type = token.INT
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

// makeTwoCharToken construye un token de dos caracteres (==, <=, &&, ...)
// usando el carácter actual y el siguiente.
func (l *Lexer) makeTwoCharToken(tokenType token.TokenType) token.Token {
	ch := l.ch
	l.readChar()
	return token.Token{Type: tokenType, Literal: string(ch) + string(l.ch)}
}

// readIdentifier consume letras ASCII, guiones bajos y dígitos.
// El llamador comprueba que el primer byte no sea un dígito.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber lee una secuencia de dígitos.
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readString devuelve el contenido sin comillas y deja ch en la comilla de
// cierre. Devuelve false si llega a EOF antes de encontrarla.
func (l *Lexer) readString() (string, bool) {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' {
			return l.input[position:l.position], true
		}
		if l.ch == 0 {
			return l.input[position:l.position], false
		}
	}
}

// skipWhitespace ignora espacios, tabs y saltos de línea.
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// peekChar mira el siguiente carácter sin avanzar.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// newToken crea un token de un solo carácter.
func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

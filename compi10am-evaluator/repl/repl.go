// Package repl conecta la entrada del usuario con lexer, parser y evaluador.
// Cada vuelta del bucle corresponde a una línea escrita en la consola.
package repl

import (
	"bufio"           // Lee la entrada una línea a la vez.
	"compi/ast"       // Distingue declaraciones de expresiones al imprimir.
	"compi/evaluator" // Ejecuta el AST en el modo 3.
	"compi/lexer"     // Convierte la línea en tokens.
	"compi/object"    // Crea y conserva el entorno de variables.
	"compi/parser"    // Convierte los tokens en un programa AST.
	"compi/token"     // Reconoce el token de fin de entrada en modo lexer.
	"fmt"             // Escribe el prompt y los resultados.
	"io"              // Permite usar consola o cualquier Reader/Writer.
	"strings"         // Limpia el comando de salida.
)

// RunSource procesa un programa completo, incluidas las sentencias repartidas
// en varias líneas. Devuelve errores para que el modo de archivo pueda salir
// con un código distinto de cero.
func RunSource(source string, out io.Writer, mode int) error {
	l := lexer.New(source)
	if mode == LEXER_MODE {
		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			fmt.Fprintf(out, "%+v\n", tok)
		}
		return nil
	}

	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return fmt.Errorf("error de sintaxis:\n  - %s", strings.Join(p.Errors(), "\n  - "))
	}
	if mode == PARSER_MODE {
		fmt.Fprintf(out, "AST Generado: \n%s\n", program.String())
		return nil
	}
	if mode != EVALUATOR_MODE {
		return fmt.Errorf("modo desconocido: %d", mode)
	}

	result := evaluator.Eval(program, object.NewEnviroment())
	if result != nil && result.Type() == object.ERROR_OBJ {
		return fmt.Errorf("%s", result.Inspect())
	}
	// Igual que en el REPL, solo se muestra el valor de una expresión final
	// o de un return final. Las declaraciones y los ciclos previos se ejecutan.
	if len(program.Statements) > 0 && result != nil {
		switch program.Statements[len(program.Statements)-1].(type) {
		case *ast.ExpressionStatement, *ast.ReturnStatement:
			fmt.Fprintln(out, result.Inspect())
		}
	}
	return nil
}

// PROMPT es el texto que se muestra antes de cada línea de entrada.
const PROMPT = ">> "

// Los modos se enumeran desde cero con iota. main traduce las opciones
// visibles 1, 2 y 3 a estas constantes internas.
const (
	LEXER_MODE     = iota // imprime la lista de tokens
	PARSER_MODE           // imprime el AST generado
	EVALUATOR_MODE        // ejecuta el programa
)

// Start procesa una línea por vez y escribe la salida en out. En modo evaluador
// conserva un único entorno para que las variables persistan entre líneas.
// Termina al recibir EOF, "exit" o "salir".
func Start(in io.Reader, out io.Writer, mode int) {
	// Scanner separa la entrada al encontrar cada salto de línea.
	scanner := bufio.NewScanner(in)
	// Se crea una sola vez para recordar let y funciones entre líneas.
	env := object.NewEnviroment()

	// El REPL se repite hasta que el usuario salga o cierre la entrada.
	for {
		// Se muestra el prompt antes de esperar la siguiente línea.
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			// Scan devuelve false al llegar a EOF.
			return
		}

		// Text devuelve la línea sin el salto final.
		line := scanner.Text()

		// Se ignoran espacios y mayúsculas solo para reconocer exit/salir.
		// El código del usuario se analiza con su texto original.
		cleanedLine := strings.TrimSpace(strings.ToLower(line))
		if cleanedLine == "exit" || cleanedLine == "salir" {
			fmt.Fprintf(out, "¡Miau! Nos vemos. Saliendo de CatCompiler...\n")
			return
		}

		// Cada línea necesita un lexer nuevo, desde su primer carácter.
		l := lexer.New(line)

		if mode == LEXER_MODE {
			// Este modo lee tokens hasta EOF y muestra cada uno sin parsear.
			for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
				fmt.Fprintf(out, "%+v\n", tok)
			}
		} else {
			// Parser y evaluador comparten la construcción del AST.
			p := parser.New(l)
			program := p.ParseProgram()

			// Nunca se ejecuta un programa cuyo análisis sintáctico falló.
			if len(p.Errors()) != 0 {
				printParserErrors(out, p.Errors())
				continue
			}

			if mode == EVALUATOR_MODE {
				// El entorno compartido recibe los cambios de esta línea.
				result := evaluator.Eval(program, env)
				// Las expresiones y los errores se muestran; una declaración let
				// correcta solo actualiza el entorno y no imprime null.
				show := result != nil && result.Type() == object.ERROR_OBJ
				if len(program.Statements) > 0 {
					// La última sentencia determina si hay un valor que mostrar.
					switch program.Statements[len(program.Statements)-1].(type) {
					case *ast.ExpressionStatement, *ast.ReturnStatement:
						// Las expresiones muestran incluso un resultado null.
						show = true
					}
				}
				if show && result != nil {
					fmt.Fprintln(out, result.Inspect())
				}
			} else {
				// El modo parser muestra el árbol como texto y no lo ejecuta.
				fmt.Fprintf(out, "AST Generado: \n")
				fmt.Fprintf(out, "%s\n", program.String())
			}
		}
	}
}

// printParserErrors muestra todos los errores sintácticos de la línea actual.
func printParserErrors(out io.Writer, errors []string) {
	// La cabecera deja claro que el programa no llegó a evaluarse.
	fmt.Fprintf(out, "¡Miau! Ocurrió un error al intentar parsear el código:\n")
	for _, msg := range errors {
		// Cada mensaje del parser aparece en una línea separada.
		fmt.Fprintf(out, "\t- %s\n", msg)
	}
}

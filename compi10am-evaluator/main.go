// CatCompiler ofrece una consola para inspeccionar tokens, AST o ejecutar código.
// Este archivo solo muestra el menú; el trabajo del intérprete vive en repl.
package main

import (
	"bufio"      // Lee la opción inicial sin perder líneas ya recibidas.
	"compi/repl" // Ejecuta el modo elegido.
	"flag"
	"fmt"     // Imprime el saludo y el menú.
	"os"      // Accede a entrada y salida estándar.
	"os/user" // Obtiene el nombre de usuario del sistema.
	"strings" // Elimina espacios alrededor de la opción.
)

func main() {
	if len(os.Args) > 1 {
		if err := runFile(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// El saludo usa el nombre de usuario del sistema.
	user, err := user.Current()
	if err != nil {
		// Si no se puede determinar el usuario, se detiene el programa.
		panic(err)
	}
	// Estas líneas construyen el dibujo y el texto de bienvenida.
	fmt.Printf("********************************************\n")
	fmt.Printf("**                                        **\n")
	fmt.Printf("**  -- BIENVENIDO A CATCOMPILER REPL --   **\n")
	fmt.Printf("**                                        **\n")
	fmt.Printf("**      /\\_/\\     ¡Miau! Soy GoGoGo,       **\n")
	fmt.Printf("**     ( o.o )    tu asistente felino.    **\n")
	fmt.Printf("**      > ^ <     ¡Dame esos tokens!      **\n")
	fmt.Printf("**                                        **\n")
	fmt.Printf("********************************************\n\n")
	fmt.Printf("¡Hola %s! Estás en CatCompiler.\n",
		user.Username)

	// El menú traduce una opción visible a una constante del paquete repl.
	fmt.Printf("\nSelecciona el modo de interacción:\n")
	fmt.Printf("1. Modo Lexer (Muestra los Tokens generados)\n")
	fmt.Printf("2. Modo Parser (Construye y muestra el Árbol AST)\n")
	fmt.Printf("3. Modo Evaluador (Ejecuta el código)\n")
	fmt.Printf("Opción [1, 2 o 3]: ")

	// Se conserva reader para pasarlo después al REPL; bufio puede leer
	// caracteres adicionales al salto de línea de la opción.
	reader := bufio.NewReader(os.Stdin)
	option, _ := reader.ReadString('\n')
	// TrimSpace quita el salto de línea y posibles espacios escritos.
	option = strings.TrimSpace(option)

	// Las opciones 2 y 3 activan el parser y el evaluador respectivamente.
	// La opción 1 y cualquier valor no reconocido usan el lexer.
	mode := repl.LEXER_MODE
	if option == "3" {
		mode = repl.EVALUATOR_MODE
		fmt.Printf("\n¡Iniciando en Modo Evaluador! Escribe código para ejecutarlo.\n")
	} else if option == "2" {
		mode = repl.PARSER_MODE
		fmt.Printf("\n¡Iniciando en Modo Parser! Escribe código para ver el árbol.\n")
	} else {
		fmt.Printf("\n¡Iniciando en Modo Lexer! Escribe código para ver los tokens.\n")
	}

	// Reutilizar reader conserva las líneas que bufio haya leído por adelantado.
	repl.Start(reader, os.Stdout, mode)
}

// runFile ejecuta un archivo como un único programa. Sin -modo se evalúa.
func runFile(args []string) error {
	flags := flag.NewFlagSet("catcompiler", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	modeName := flags.String("modo", "evaluador", "lexer, parser o evaluador")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("uso: catcompiler [-modo lexer|parser|evaluador] archivo.cat")
	}
	mode := repl.EVALUATOR_MODE
	switch *modeName {
	case "lexer":
		mode = repl.LEXER_MODE
	case "parser":
		mode = repl.PARSER_MODE
	case "evaluador":
	default:
		return fmt.Errorf("modo desconocido: %s (usa lexer, parser o evaluador)", *modeName)
	}
	path := flags.Arg(0)
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", path, err)
	}
	if err := repl.RunSource(string(source), os.Stdout, mode); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

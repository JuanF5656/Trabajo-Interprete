# CatCompiler

CatCompiler es un intérprete escrito en Go. Incluye un analizador léxico, un parser, un evaluador y un REPL.

## Requisitos

- Go 1.26.3 o una versión compatible con la indicada en `go.mod`.

## Ejecución interactiva

Desde la raíz del proyecto:

```bash
go run .
```

Al iniciar, el programa pide elegir un modo:

1. **Lexer:** muestra el tipo y el texto de cada token.
2. **Parser:** construye e imprime una representación textual del AST.
3. **Evaluador:** ejecuta el código y muestra su resultado.

Escribe una instrucción por línea en el prompt `>>`. Para terminar, escribe `exit` o `salir` (también puedes enviar EOF). En modo evaluador, las variables y funciones persisten entre líneas. Si eliges una opción distinta de `2` o `3`, se inicia el modo lexer. Los bloques de varias líneas se deben ejecutar desde un archivo.

## Ejecutar un archivo

Guarda el programa en un archivo de texto y pasa su ruta como argumento. Se analiza y ejecuta todo el archivo como un solo programa, sin menú ni prompt:

```bash
go run . ejemplos/demo.cat
```

El ejemplo incluido imprime `[17, 7, 55]`: el total calculado con `for`, `hackerman(2, 2)` y `fibonacci(10)`. También puedes inspeccionar los tokens o el AST del archivo:

```bash
go run . -modo lexer ejemplos/demo.cat
go run . -modo parser ejemplos/demo.cat
go run . -modo evaluador ejemplos/demo.cat
```

La opción `-modo` debe ir antes de la ruta. Si falta el archivo, hay un error de sintaxis o falla la ejecución, el programa informa el error y termina con código distinto de cero. En modo evaluador se imprime el valor de la última expresión; las declaraciones `let` y las asignaciones anteriores sí se ejecutan. Para ver un resultado al final, termina el archivo con una expresión como `total;`.

## Ejemplos del lenguaje

Selecciona el modo **3 (Evaluador)** para probar estas instrucciones sencillas en el REPL:

```text
let resultado = 2 + 3 * 4;
resultado;
```

Resultado: `14`. Para funciones y bloques largos, crea un archivo `.cat`, por ejemplo:

```text
let suma = function(a, b) {
    return a + b;
};
suma(3, 4);
```

Resultado: `7`. La sintaxis para definir una función es `let nombre = function(...) { ... };`. El ciclo `for` usa inicialización, condición y paso separados por `;`:

```text
let total = 0;
for (let i = 1; i <= 5; i = i + 1) {
    total = total + i;
}
total;
```

Resultado: `15`. También hay `while`, `break` y `continue`:

```text
let i = 0;
let total = 0;
while (i < 5) {
    i = i + 1;
    if (i == 3) { continue; }
    total = total + i;
}
total;
```

Resultado: `12`. Las condiciones admiten `if`, `elif` y `else`:

```text
let nota = 85;
if (nota >= 90) { "excelente"; }
elif (nota >= 60) { "aprobado"; }
else { "reprobado"; }
```

Resultado: `aprobado`. Para arreglos y diccionarios:

```text
let numeros = [10, 20, 30];
let persona = {"nombre": "Ana", "activo": true};
numeros[1] + 5;
```

Resultado: `25`. Puedes consultar `persona["nombre"]` o `persona["activo"]` como última expresión para ver sus valores.

El archivo [ejemplos/demo.cat](ejemplos/demo.cat) también define dos funciones recursivas. `hackerman` implementa la función de Ackermann y `fibonacci` calcula el término de Fibonacci:

```text
let hackerman = function(m, n) {
    if (m == 0) { return n + 1; }
    if (n == 0) { return hackerman(m - 1, 1); }
    return hackerman(m - 1, hackerman(m, n - 1));
};

let fibonacci = function(n) {
    if (n <= 1) { return n; }
    return fibonacci(n - 1) + fibonacci(n - 2);
};

[hackerman(2, 2), fibonacci(10)];
```

Este fragmento imprime `[7, 55]`. Usa valores pequeños para `hackerman`, porque el número de llamadas recursivas crece muy rápido.

El lenguaje reconoce enteros, cadenas, booleanos, `null`, arreglos, diccionarios, funciones, llamadas, índices, asignaciones, operaciones aritméticas y lógicas, condicionales (`if`, `elif`, `else`) y ciclos (`while`, `for`).

Los arreglos y diccionarios admiten índices; una posición o clave ausente devuelve `null`. Los diccionarios admiten claves enteras, booleanas y de texto. Las operaciones inválidas producen un error de ejecución.

## Estructura

| Ruta | Función |
| --- | --- |
| `main.go` | Menú inicial, arranque del REPL y ejecución de archivos. |
| `token/` | Tipos de tokens y palabras reservadas. |
| `lexer/` | Conversión del texto de entrada en tokens. |
| `parser/` | Construcción del AST mediante análisis por precedencia. |
| `ast/` | Nodos y representación textual del AST. |
| `repl/` | Lectura de líneas y salida de los tres modos. |
| `object/` | Tipos de valores y entornos del intérprete. |
| `evaluator/` | Ejecución del AST. |
| `ejemplos/` | Programas listos para ejecutar desde archivo. |

## Cómo reconstruir el intérprete

El código está comentado siguiendo el recorrido de una línea de entrada. Para implementarlo desde cero, conviene avanzar en este orden:

1. En `object/object.go`, definir la interfaz `Object` y los valores simples (`Integer`, `Boolean`, `String`, `Null` y `Error`). Cada valor debe indicar su tipo y cómo se muestra.
2. Crear `Environment` con un mapa de nombres a valores. `Get` busca también en el entorno exterior; `Set` declara localmente y `Assign` actualiza una declaración existente.
3. En `evaluator/evaluator.go`, crear `Eval` y el recorrido `eval`. Empezar por `Program`, `BlockStatement`, literales, identificadores, `let` y expresiones aritméticas.
4. Añadir `ReturnValue` para que `return` atraviese bloques sin ejecutar las sentencias siguientes. Los errores usan el mismo mecanismo de propagación.
5. Añadir condicionales y la función `truthy`, que decide si un valor cuenta como verdadero.
6. Añadir funciones. Una función guarda sus parámetros, cuerpo y entorno de definición; cada llamada crea un entorno hijo para los argumentos.
7. Añadir arreglos, diccionarios e índices. Para los diccionarios, convertir solo enteros, booleanos y cadenas en claves comparables.
8. Añadir `while`, `for`, `break` y `continue`. Los ciclos consumen esas señales; los errores y `return` continúan hacia afuera.
9. En `repl/repl.go`, conectar la secuencia `línea → lexer → parser → evaluador`. Reutilizar un mismo entorno para recordar las variables entre líneas.

Por ejemplo, al escribir `let x = 2 + 3;`, el lexer produce tokens, el parser construye un `LetStatement`, el evaluador calcula `2 + 3` como `Integer(5)` y `Environment.Set` guarda ese objeto con el nombre `x`. En la siguiente línea, el nodo `Identifier` de `x` recupera ese valor mediante `Environment.Get`.

## Verificación

```bash
go test ./...
```

// Package evaluator recorre el AST y produce valores del lenguaje.
// Recibe un Environment para conservar variables entre evaluaciones.
package evaluator

import (
	"compi/ast"    // Tipos de nodos que construye el parser.
	"compi/object" // Valores, funciones y entornos que existen durante la ejecución.
	"fmt"          // Formato de los mensajes de error.
)

// Estos valores no tienen estado mutable y se reutilizan en cada evaluación.
var (
	trueValue     = &object.Boolean{Value: true}       // Único objeto usado para true.
	falseValue    = &object.Boolean{Value: false}      // Único objeto usado para false.
	nullValue     = &object.Null{}                     // Representa ausencia de resultado.
	breakValue    = &signal{kind: object.BREAK_OBJ}    // Orden de salir del ciclo actual.
	continueValue = &signal{kind: object.CONTINUE_OBJ} // Orden de pasar a la siguiente vuelta.
)

// signal transporta break y continue hasta el ciclo que debe atenderlos.
// No son valores que el usuario pueda crear directamente.
type signal struct{ kind object.ObjectType } // El tipo distingue break de continue.

// Type permite que signal cumpla la interfaz object.Object.
func (s *signal) Type() object.ObjectType { return s.kind }

// Inspect representa la señal si llega a convertirse en un mensaje de error.
func (s *signal) Inspect() string { return string(s.kind) }

// Eval evalúa un programa o un nodo del AST en el entorno recibido.
// Si env es nil, crea un entorno nuevo para esa evaluación.
func Eval(node ast.Node, env *object.Environment) object.Object {
	// Un entorno nil permite usar Eval sin preparar previamente las variables.
	if env == nil {
		env = object.NewEnviroment()
	}
	// El trabajo recursivo se hace en eval, que reutiliza este mismo entorno.
	return eval(node, env)
}

// eval despacha según el tipo de nodo. Los errores y las señales de control
// se devuelven como objetos para que los nodos padres puedan propagarlos.
func eval(node ast.Node, env *object.Environment) object.Object {
	// El type switch selecciona la regla de ejecución correspondiente al nodo.
	switch n := node.(type) {
	case *ast.Program:
		// En la raíz, return entrega su valor; break y continue sin ciclo son errores.
		// Si no hay sentencias, el resultado del programa es null.
		result := object.Object(nullValue)
		// Las sentencias se ejecutan en el orden en que fueron escritas.
		for _, stmt := range n.Statements {
			result = eval(stmt, env)
			// Una señal de control impide ejecutar las sentencias restantes.
			switch result.(type) {
			case *object.ReturnValue:
				// En la raíz ya no hace falta el envoltorio de return.
				return result.(*object.ReturnValue).Value
			case *object.Error:
				// Se entrega el error original al REPL.
				return result
			case *signal:
				// Ningún ciclo recibió esta señal, por eso es un uso inválido.
				return newError("%s fuera de un ciclo", result.Inspect())
			}
		}
		// Si no hubo interrupciones, prevalece el valor de la última sentencia.
		return result
	case *ast.BlockStatement:
		// Un bloque detiene sus sentencias al encontrar return, break, continue o error.
		result := object.Object(nullValue)
		for _, stmt := range n.Statements {
			// El entorno se comparte con el bloque que llamó a eval.
			result = eval(stmt, env)
			if isControl(result) {
				// El bloque no consume la señal: la entrega a la función o al ciclo.
				return result
			}
		}
		return result
	case *ast.ExpressionStatement:
		// Una expresión usada como sentencia conserva su resultado.
		return eval(n.Expression, env)
	case *ast.LetStatement:
		// let declara en el entorno actual; la declaración no produce un valor visible.
		// Primero se calcula el lado derecho para guardar un valor ya evaluado.
		value := eval(n.Value, env)
		if isControl(value) {
			// Un error impide crear la variable.
			return value
		}
		env.Set(n.Name.Value, value)
		return nullValue
	case *ast.ReturnStatement:
		// Se evalúa la expresión que seguirá a return.
		value := eval(n.ReturnValue, env)
		if isControl(value) {
			return value
		}
		// El envoltorio indica a los bloques exteriores que deben detenerse.
		return &object.ReturnValue{Value: value}
	case *ast.BreakStatement:
		// El ciclo exterior transformará esta señal en una salida del ciclo.
		return breakValue
	case *ast.ContinueStatement:
		// El ciclo exterior transformará esta señal en una nueva iteración.
		return continueValue
	case *ast.IntegerLiteral:
		// El AST guarda el número como int64; se convierte en valor de ejecución.
		return &object.Integer{Value: n.Value}
	case *ast.StringLiteral:
		// Las cadenas del AST también se envuelven en un objeto del lenguaje.
		return &object.String{Value: n.Value}
	case *ast.Boolean:
		// El parser usa Boolean para los literales true y false.
		return nativeBool(n.Value)
	case *ast.BooleanStatement:
		// Este nodo antiguo se conserva por compatibilidad con el AST.
		return nativeBool(n.Value)
	case *ast.NullLiteral:
		// Todos los literales null comparten el mismo valor sin estado.
		return nullValue
	case *ast.Identifier:
		// Get busca primero en el entorno actual y luego en los exteriores.
		if value, ok := env.Get(n.Value); ok {
			return value
		}
		// Usar una variable no declarada detiene la evaluación.
		return newError("identificador no definido: %s", n.Value)
	case *ast.PrefixExpression:
		// Los operadores prefijos evalúan su único operando antes de aplicarse.
		right := eval(n.Right, env)
		if isControl(right) {
			return right
		}
		switch n.Operator {
		case "!":
			// ! invierte la condición según las reglas de truthy.
			return nativeBool(!truthy(right))
		case "-":
			// El signo negativo solo se admite para enteros.
			if value, ok := right.(*object.Integer); ok {
				return &object.Integer{Value: -value.Value}
			}
		}
		// Cualquier otra combinación de operador y tipo es inválida.
		return newError("operador desconocido: %s%s", n.Operator, right.Type())
	case *ast.InfixExpression:
		// La mayoría de los operadores evalúan izquierda y luego derecha.
		left := eval(n.Left, env)
		if isControl(left) {
			return left
		}
		// && y || evitan evaluar la derecha cuando la izquierda basta.
		if n.Operator == "&&" && !truthy(left) {
			return falseValue
		}
		if n.Operator == "||" && truthy(left) {
			return trueValue
		}
		right := eval(n.Right, env)
		if isControl(right) {
			return right
		}
		// Si se llegó aquí, ambos operandos lógicos eran necesarios.
		if n.Operator == "&&" {
			return nativeBool(truthy(right))
		}
		if n.Operator == "||" {
			return nativeBool(truthy(right))
		}
		// Los demás operadores se resuelven según el tipo de los dos valores.
		return evalInfix(n.Operator, left, right)
	case *ast.AssignExpression:
		// La asignación actualiza la declaración más cercana, incluso si está
		// en un entorno exterior capturado por una función.
		// Primero se calcula el nuevo valor; si falla, la variable queda intacta.
		value := eval(n.Value, env)
		if isControl(value) {
			return value
		}
		// Assign devuelve false cuando el nombre no existe en ningún ámbito.
		if !env.Assign(n.Name.Value, value) {
			return newError("identificador no definido: %s", n.Name.Value)
		}
		// Una asignación es una expresión y devuelve el valor asignado.
		return value
	case *ast.IfExpression:
		// Solo se ejecuta la rama cuya condición sea verdadera.
		condition := eval(n.Condition, env)
		if isControl(condition) {
			return condition
		}
		if truthy(condition) {
			// Consequence contiene las sentencias entre las primeras llaves.
			return eval(n.Consequence, env)
		}
		if n.Alternative != nil {
			// El parser representa elif como otro if dentro de Alternative.
			return eval(n.Alternative, env)
		}
		// Un if sin rama ejecutada no produce valor.
		return nullValue
	case *ast.WhileExpression:
		// El cuerpo conserva el mismo entorno en todas las iteraciones.
		// result retiene el último valor normal producido por el cuerpo.
		result := object.Object(nullValue)
		for {
			// while vuelve a comprobar la condición antes de cada iteración.
			condition := eval(n.Condition, env)
			if isControl(condition) {
				return condition
			}
			if !truthy(condition) {
				// La condición falsa termina el ciclo.
				return result
			}
			// El bloque puede devolver un valor normal o una señal de control.
			value := eval(n.Body, env)
			switch value.Type() {
			case object.BREAK_OBJ:
				// break termina este while y consume la señal.
				return result
			case object.CONTINUE_OBJ:
				// continue salta directamente a la siguiente condición.
				continue
			case object.ERROR_OBJ, object.RETURN_VALUE_OBJ:
				// Un error o return debe salir también de los bloques exteriores.
				return value
			default:
				// Guardamos el valor ordinario de la vuelta actual.
				result = value
			}
		}
	case *ast.ForExpression:
		// El inicializador se ejecuta una vez; condición, cuerpo y paso se
		// repiten en ese orden. Un continue también ejecuta el paso.
		if n.Initializer != nil {
			// El inicializador puede ser, por ejemplo, let i = 0.
			value := eval(n.Initializer, env)
			if isControl(value) {
				return value
			}
		}
		// Se devuelve el último valor normal del cuerpo al terminar el ciclo.
		result := object.Object(nullValue)
		for {
			// Una condición omitida equivale a un ciclo sin límite propio.
			if n.Condition != nil {
				condition := eval(n.Condition, env)
				if isControl(condition) {
					return condition
				}
				if !truthy(condition) {
					return result
				}
			}
			// Se ejecuta el cuerpo antes del paso.
			value := eval(n.Body, env)
			switch value.Type() {
			case object.BREAK_OBJ:
				// break sale sin ejecutar el paso.
				return result
			case object.ERROR_OBJ, object.RETURN_VALUE_OBJ:
				// Errores y return atraviesan el ciclo.
				return value
			case object.CONTINUE_OBJ:
				// Se ignora el resto del cuerpo, pero todavía se ejecuta Post.
			default:
				result = value
			}
			// Post suele incrementar el contador: i = i + 1.
			if n.Post != nil {
				post := eval(n.Post, env)
				if isControl(post) {
					return post
				}
			}
		}
	case *ast.FunctionLiteral:
		// Guardar el entorno de definición permite que la función use variables
		// de ámbitos exteriores cuando se llame después.
		// Aquí se crea la función; su cuerpo todavía no se ejecuta.
		return &object.Function{Parameters: n.Parameters, Body: n.Body, Env: env}
	case *ast.CallExpression:
		// La parte izquierda de una llamada puede ser un nombre o una función literal.
		function := eval(n.Function, env)
		if isControl(function) {
			return function
		}
		// Los argumentos se evalúan antes de entrar en la función.
		args, failure := evalExpressions(n.Arguments, env)
		if failure != nil {
			return failure
		}
		// Se comprueba que el valor llamado sea realmente una función.
		fn, ok := function.(*object.Function)
		if !ok {
			return newError("no es una función: %s", function.Type())
		}
		// Cada parámetro necesita exactamente un argumento.
		if len(args) != len(fn.Parameters) {
			return newError("número de argumentos incorrecto: se esperaban %d, se recibieron %d", len(fn.Parameters), len(args))
		}
		// El entorno hijo contiene parámetros y puede consultar la clausura.
		callEnv := object.NewEnclosedEnviroment(fn.Env)
		// Cada llamada obtiene un entorno propio para sus parámetros.
		for i, param := range fn.Parameters {
			callEnv.Set(param.Value, args[i])
		}
		// Solo ahora se ejecutan las sentencias del cuerpo.
		value := eval(fn.Body, callEnv)
		if returned, ok := value.(*object.ReturnValue); ok {
			// El llamador recibe el valor, sin el envoltorio de return.
			return returned.Value
		}
		if _, ok := value.(*signal); ok {
			// break/continue no pueden atravesar una llamada de función.
			return newError("%s fuera de un ciclo", value.Inspect())
		}
		// Sin return explícito, se entrega el último valor del bloque.
		return value
	case *ast.ArrayLiteral:
		// Se evalúan sus elementos en orden y se detiene ante el primer error.
		elements, failure := evalExpressions(n.Elements, env)
		if failure != nil {
			return failure
		}
		return &object.Array{Elements: elements}
	case *ast.HashLiteral:
		// Order conserva el orden de entrada al mostrar el diccionario; Pairs
		// permite buscar una clave sin recorrer todas las parejas.
		hash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		// Cada clave y cada valor son expresiones que se deben evaluar.
		for _, pair := range n.Pairs {
			key := eval(pair.Key, env)
			if isControl(key) {
				return key
			}
			// Los mapas de Go necesitan claves comparables.
			hashKey, ok := object.KeyFor(key)
			if !ok {
				return newError("clave de diccionario no válida: %s", key.Type())
			}
			value := eval(pair.Value, env)
			if isControl(value) {
				return value
			}
			// Una clave repetida reemplaza su valor sin cambiar su posición.
			if _, exists := hash.Pairs[hashKey]; !exists {
				hash.Order = append(hash.Order, hashKey)
			}
			hash.Pairs[hashKey] = object.HashPair{Key: key, Value: value}
		}
		return hash
	case *ast.IndexExpression:
		// Un índice fuera del arreglo o una clave ausente devuelve null.
		// Se evalúa primero la colección y luego la expresión del índice.
		left := eval(n.Left, env)
		if isControl(left) {
			return left
		}
		index := eval(n.Index, env)
		if isControl(index) {
			return index
		}
		switch value := left.(type) {
		case *object.Array:
			// Los arreglos solo admiten posiciones enteras desde cero.
			i, ok := index.(*object.Integer)
			if !ok {
				return newError("índice de arreglo debe ser INTEGER, recibido %s", index.Type())
			}
			if i.Value < 0 || i.Value >= int64(len(value.Elements)) {
				return nullValue
			}
			return value.Elements[i.Value]
		case *object.Hash:
			// Se convierte el índice al mismo formato usado al crear el diccionario.
			key, ok := object.KeyFor(index)
			if !ok {
				return newError("clave de diccionario no válida: %s", index.Type())
			}
			if pair, exists := value.Pairs[key]; exists {
				return pair.Value
			}
			return nullValue
		default:
			// Un entero, booleano o función no se puede indexar.
			return newError("índice no válido para %s", left.Type())
		}
	}
	// Si se agrega un nodo al AST, también debe añadirse su caso aquí.
	return newError("nodo no soportado: %T", node)
}

// evalExpressions evalúa argumentos o elementos de izquierda a derecha.
// El segundo resultado contiene el primer error o señal de control, si aparece.
func evalExpressions(expressions []ast.Expression, env *object.Environment) ([]object.Object, object.Object) {
	// La capacidad inicial evita ampliar el slice mientras añadimos valores.
	values := make([]object.Object, 0, len(expressions))
	for _, expression := range expressions {
		// Se usa el mismo entorno para que los efectos laterales sean visibles.
		value := eval(expression, env)
		if isControl(value) {
			// nil en la primera posición indica que la lista quedó incompleta.
			return nil, value
		}
		values = append(values, value)
	}
	// nil en la segunda posición indica que todo salió bien.
	return values, nil
}

// evalInfix aplica operadores según los tipos ya evaluados de ambos operandos.
// Los tipos u operadores no admitidos producen un error del lenguaje.
func evalInfix(op string, left, right object.Object) object.Object {
	// Primero se prueban operaciones entre dos enteros.
	if a, ok := left.(*object.Integer); ok {
		if b, ok := right.(*object.Integer); ok {
			switch op {
			case "+":
				// Suma entera.
				return &object.Integer{Value: a.Value + b.Value}
			case "-":
				// Resta entera.
				return &object.Integer{Value: a.Value - b.Value}
			case "*":
				// Multiplicación entera.
				return &object.Integer{Value: a.Value * b.Value}
			case "/":
				// La división entre enteros trunca la parte decimal.
				if b.Value == 0 {
					return newError("división por cero")
				}
				return &object.Integer{Value: a.Value / b.Value}
			case "%":
				// El residuo tampoco está definido cuando el divisor es cero.
				if b.Value == 0 {
					return newError("módulo por cero")
				}
				return &object.Integer{Value: a.Value % b.Value}
			case "**", "^":
				// El lenguaje solo admite exponentes enteros no negativos.
				if b.Value < 0 {
					return newError("exponente negativo: %d", b.Value)
				}
				return &object.Integer{Value: power(a.Value, b.Value)}
			case "<":
				// Las comparaciones devuelven objetos Boolean del lenguaje.
				return nativeBool(a.Value < b.Value)
			case "<=":
				return nativeBool(a.Value <= b.Value)
			case ">":
				return nativeBool(a.Value > b.Value)
			case ">=":
				return nativeBool(a.Value >= b.Value)
			case "==":
				return nativeBool(a.Value == b.Value)
			case "!=":
				return nativeBool(a.Value != b.Value)
			}
		}
	}
	// Las cadenas admiten concatenación y comparación de igualdad.
	if a, ok := left.(*object.String); ok {
		if b, ok := right.(*object.String); ok {
			switch op {
			case "+":
				return &object.String{Value: a.Value + b.Value}
			case "==":
				return nativeBool(a.Value == b.Value)
			case "!=":
				return nativeBool(a.Value != b.Value)
			}
		}
	}
	// La igualdad se define también para booleanos, null y referencias.
	if op == "==" || op == "!=" {
		equal := false
		switch a := left.(type) {
		case *object.Boolean:
			// Se comparan los valores bool, no las direcciones de memoria.
			if b, ok := right.(*object.Boolean); ok {
				equal = a.Value == b.Value
			}
		case *object.Null:
			// Cualquier objeto null es igual a otro objeto null.
			_, equal = right.(*object.Null)
		default:
			// Para otros objetos, la igualdad es identidad de objeto.
			equal = left == right
		}
		if op == "==" {
			return nativeBool(equal)
		}
		return nativeBool(!equal)
	}
	// Los tipos distintos no admiten operadores aritméticos entre sí.
	if left.Type() != right.Type() {
		return newError("tipos incompatibles: %s %s %s", left.Type(), op, right.Type())
	}
	// Dos valores del mismo tipo también pueden recibir un operador inválido.
	return newError("operador desconocido: %s %s %s", left.Type(), op, right.Type())
}

// power calcula potencias enteras mediante cuadrados sucesivos.
func power(base, exponent int64) int64 {
	// Se empieza en 1 porque cualquier número elevado a cero vale 1.
	result := int64(1)
	for exponent > 0 {
		// Un bit final igual a 1 indica que esta potencia aporta al resultado.
		if exponent&1 == 1 {
			result *= base
		}
		// Cuadrar la base y desplazar el exponente reduce el trabajo a log(n).
		base *= base
		exponent >>= 1
	}
	return result
}

// nativeBool convierte un bool de Go en el objeto Boolean del lenguaje.
func nativeBool(value bool) object.Object {
	if value {
		return trueValue
	}
	return falseValue
}

// truthy define las condiciones: solo false y null se consideran falsos.
func truthy(value object.Object) bool {
	// Se usa el tipo concreto, porque solo Boolean y Null tienen reglas propias.
	switch v := value.(type) {
	case *object.Null:
		return false
	case *object.Boolean:
		return v.Value
	default:
		// Enteros, cadenas, arreglos, diccionarios y funciones son verdaderos.
		return true
	}
}

// isControl identifica objetos que interrumpen la evaluación normal.
func isControl(value object.Object) bool {
	// nil se interpreta como ausencia de señal.
	if value == nil {
		return false
	}
	switch value.Type() {
	case object.ERROR_OBJ, object.RETURN_VALUE_OBJ, object.BREAK_OBJ, object.CONTINUE_OBJ:
		// Estos tipos requieren detener o redirigir el flujo de ejecución.
		return true
	default:
		return false
	}
}

// newError usa el mismo formato de fmt.Sprintf para añadir datos al mensaje.
func newError(format string, args ...any) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, args...)}
}

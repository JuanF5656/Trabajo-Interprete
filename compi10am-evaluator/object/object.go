// Package object define los valores que produce el intérprete y el lugar donde
// se guardan las variables. El AST describe código; estos objetos representan
// datos concretos obtenidos al ejecutar ese código.
package object

import (
	"compi/ast" // Permite guardar el cuerpo y los parámetros de una función.
	"fmt"       // Convierte números y booleanos a texto.
	"strings"   // Une los elementos al mostrar arreglos, diccionarios y funciones.
)

// ObjectType identifica la clase de un valor del lenguaje durante la ejecución.
type ObjectType string

// Cada constante nombra un tipo que puede devolver Type(). RETURN_VALUE,
// BREAK y CONTINUE son señales internas de control, no datos escritos por el usuario.
const (
	INTEGER_OBJ      = "INTEGER"      // Número entero.
	BOOLEAN_OBJ      = "BOOLEAN"      // Valor verdadero o falso.
	STRING_OBJ       = "STRING"       // Texto.
	RETURN_VALUE_OBJ = "RETURN_VALUE" // Salida de una función o programa.
	ERROR_OBJ        = "ERROR"        // Fallo en tiempo de ejecución.
	FUNCTION_OBJ     = "FUNCTION"     // Función definida por el usuario.
	NULL_OBJ         = "NULL"         // Ausencia de valor.
	ARRAY_OBJ        = "ARRAY"        // Lista indexada por posición.
	HASH_OBJ         = "HASH"         // Diccionario indexado por clave.
	BREAK_OBJ        = "BREAK"        // Salida de un ciclo.
	CONTINUE_OBJ     = "CONTINUE"     // Siguiente iteración de un ciclo.
)

// Object es la interfaz común de todos los valores del lenguaje. Type permite
// decidir qué operaciones admite un valor; Inspect lo convierte en texto.
type Object interface {
	Type() ObjectType // Devuelve su clase durante la ejecución.
	Inspect() string  // Devuelve una representación legible para el REPL.
}

// Integer envuelve un int64 de Go para que cumpla la interfaz Object.
type Integer struct {
	Value int64 // Número que apareció en el código o resultó de una operación.
}

// Type identifica este objeto como entero.
func (i *Integer) Type() ObjectType {
	return INTEGER_OBJ
}

// Inspect convierte el entero a su forma decimal para mostrarlo.
func (i *Integer) Inspect() string {
	return fmt.Sprintf("%d", i.Value)
}

// Boolean envuelve un bool de Go para usarlo dentro del lenguaje.
type Boolean struct {
	Value bool // true o false después de evaluar una condición.
}

// Type identifica este objeto como booleano.
func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }

// Inspect escribe el booleano como true o false.
func (b *Boolean) Inspect() string { return fmt.Sprintf("%t", b.Value) }

// String guarda el contenido de una cadena sin las comillas del código fuente.
type String struct {
	Value string // Texto evaluado.
}

// Type identifica este objeto como cadena.
func (s *String) Type() ObjectType { return STRING_OBJ }

// Inspect muestra el contenido de la cadena.
func (s *String) Inspect() string { return s.Value }

// Array conserva sus elementos en orden y admite índices enteros.
type Array struct{ Elements []Object } // Cada elemento puede ser de un tipo distinto.

// Type identifica este objeto como arreglo.
func (a *Array) Type() ObjectType { return ARRAY_OBJ }

// Inspect recorre el arreglo y une las representaciones de sus elementos.
func (a *Array) Inspect() string {
	// Se reserva espacio para una cadena de salida por elemento.
	parts := make([]string, 0, len(a.Elements))
	for _, element := range a.Elements {
		parts = append(parts, element.Inspect())
	}
	// Los corchetes hacen visible que el resultado es un arreglo.
	return "[" + strings.Join(parts, ", ") + "]"
}

// HashKey combina tipo y valor para que, por ejemplo, 1 y "1" sean claves distintas.
type HashKey struct {
	Type  ObjectType // Tipo original: entero, booleano o cadena.
	Value string     // Valor convertido a texto para usarlo como clave de mapa.
}

// HashPair conserva los objetos originales para poder mostrarlos en Inspect.
type HashPair struct{ Key, Value Object } // Por ejemplo, "edad": 20.

// Hash almacena las parejas por clave y recuerda su orden de inserción.
type Hash struct {
	Pairs map[HashKey]HashPair // Acceso rápido a una pareja mediante su clave.
	Order []HashKey            // Orden de las claves para una salida estable.
}

// Type identifica este objeto como diccionario.
func (h *Hash) Type() ObjectType { return HASH_OBJ }

// Inspect muestra las parejas en el orden en que se insertaron.
func (h *Hash) Inspect() string {
	parts := make([]string, 0, len(h.Order))
	for _, key := range h.Order {
		// Se busca la pareja con la clave guardada en Order.
		pair := h.Pairs[key]
		parts = append(parts, pair.Key.Inspect()+": "+pair.Value.Inspect())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// KeyFor convierte enteros, booleanos y cadenas a claves de mapa comparables.
// Devuelve false para tipos que no se pueden usar como claves de diccionario.
func KeyFor(obj Object) (HashKey, bool) {
	// Solo estos tres tipos tienen una representación de clave estable.
	switch v := obj.(type) {
	case *Integer:
		// El tipo distingue el número 1 de la cadena "1".
		return HashKey{INTEGER_OBJ, v.Inspect()}, true
	case *Boolean:
		return HashKey{BOOLEAN_OBJ, v.Inspect()}, true
	case *String:
		return HashKey{STRING_OBJ, v.Value}, true
	default:
		// Arreglos, funciones y otros objetos no pueden ser claves.
		return HashKey{}, false
	}
}

// ReturnValue envuelve el resultado de return para atravesar bloques y ciclos
// hasta llegar al límite de la función o del programa.
type ReturnValue struct {
	Value Object // Resultado real de la expresión que sigue a return.
}

// Type indica que este objeto debe atravesar bloques hasta su función.
func (rv *ReturnValue) Type() ObjectType {
	return RETURN_VALUE_OBJ
}

// Inspect muestra el valor interno, sin exponer el envoltorio de control.
func (rv *ReturnValue) Inspect() string {
	return rv.Value.Inspect()
}

// Null representa la ausencia de valor.
type Null struct{}

// Type identifica la ausencia de valor.
func (n *Null) Type() ObjectType { return NULL_OBJ }

// Inspect permite mostrar explícitamente null en el REPL.
func (n *Null) Inspect() string { return "null" }

// Error representa un error en tiempo de ejecución.
type Error struct {
	Message string // Detalle de la operación que falló.
}

// Type permite al evaluador reconocer un fallo y dejar de ejecutar.
func (e *Error) Type() ObjectType {
	return ERROR_OBJ
}

// Inspect añade un prefijo visible cuando el REPL imprime el error.
func (e *Error) Inspect() string {
	return "ERROR: " + e.Message
}

// Function guarda los parámetros, el cuerpo y el entorno de definición de
// una función. Conservar ese entorno permite implementar clausuras.
type Function struct {
	Parameters []*ast.Identifier   // Nombres que recibirán los argumentos.
	Body       *ast.BlockStatement // Sentencias que se ejecutan al llamar.
	Env        *Environment        // Ámbito capturado cuando se creó la función.
}

// Type identifica este objeto como función.
func (f *Function) Type() ObjectType {
	return FUNCTION_OBJ
}

// Inspect resume la firma; el cuerpo puede ser largo y no se imprime entero.
func (f *Function) Inspect() string {
	params := []string{}
	for _, p := range f.Parameters {
		// Se conserva el orden de los nombres de parámetros.
		params = append(params, p.String())
	}
	return fmt.Sprintf("function(%s){...}", strings.Join(params, ", "))
}

// Environment almacena los nombres y valores de un ámbito. Si un nombre no
// existe aquí, Get lo busca en los entornos exteriores.
type Environment struct {
	store map[string]Object // Variables declaradas en este ámbito.
	outer *Environment      // Ámbito exterior; nil si este es el global.
}

// NewEnviroment crea un entorno sin ámbito exterior.
func NewEnviroment() *Environment {
	// El mapa vacío recibe las futuras declaraciones let.
	s := make(map[string]Object)
	// El entorno global no tiene padre.
	return &Environment{store: s, outer: nil}
}

// NewEnclosedEnviroment crea un entorno hijo que puede consultar outer.
func NewEnclosedEnviroment(outer *Environment) *Environment {
	// Cada llamada empieza con sus propias variables locales.
	env := NewEnviroment()
	// La referencia al exterior permite buscar variables de una clausura.
	env.outer = outer
	return env
}

// Get busca una variable en este entorno y, si no está, en los exteriores.
func (e *Environment) Get(name string) (Object, bool) {
	// El bool indica si el nombre estaba declarado.
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		// Si no existe localmente, se busca en el ámbito padre.
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

// Set declara o reemplaza un nombre solo en el ámbito actual.
func (e *Environment) Set(name string, val Object) Object {
	// La escritura local no modifica un nombre igual del entorno padre.
	e.store[name] = val
	return val
}

// Assign modifica el ámbito más cercano donde exista el nombre.
// Devuelve false si nunca se declaró, para que el evaluador emita un error.
func (e *Environment) Assign(name string, val Object) bool {
	if _, ok := e.store[name]; ok {
		// Se actualiza el primer ámbito donde aparece el nombre.
		e.store[name] = val
		return true
	}
	if e.outer != nil {
		// Si no estaba aquí, se continúa la búsqueda hacia afuera.
		return e.outer.Assign(name, val)
	}
	// No se encontró ninguna declaración que pueda actualizarse.
	return false
}

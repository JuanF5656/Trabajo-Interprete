package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunSourceMultiline(t *testing.T) {
	tests := []struct {
		name, source, want string
	}{
		{
			name: "function and for",
			source: `let suma = function(a, b) {
    return a + b;
};
let total = 0;
for (let i = 1; i <= 5; i = i + 1) {
    total = suma(total, i);
}
total;`,
			want: "15\n",
		},
		{
			name: "while and continue",
			source: `let i = 0;
let total = 0;
while (i < 5) {
    i = i + 1;
    if (i == 3) { continue; }
    total = total + i;
}
total;`,
			want: "12\n",
		},
		{
			name: "elif",
			source: `let nota = 85;
if (nota >= 90) { "excelente"; }
elif (nota >= 60) { "aprobado"; }
else { "reprobado"; }`,
			want: "aprobado\n",
		},
		{
			name: "recursive ackermann and fibonacci",
			source: `let hackerman = function(m, n) {
    if (m == 0) { return n + 1; }
    if (n == 0) { return hackerman(m - 1, 1); }
    return hackerman(m - 1, hackerman(m, n - 1));
};
let fibonacci = function(n) {
    if (n <= 1) { return n; }
    return fibonacci(n - 1) + fibonacci(n - 2);
};
[hackerman(2, 2), fibonacci(10)];`,
			want: "[7, 55]\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := RunSource(tc.source, &out, EVALUATOR_MODE); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunSourceErrors(t *testing.T) {
	for _, tc := range []struct {
		source, want string
	}{
		{"let x = ;", "error de sintaxis"},
		{"desconocido;", "identificador no definido"},
	} {
		var out bytes.Buffer
		err := RunSource(tc.source, &out, EVALUATOR_MODE)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("source %q: error = %v, want %q", tc.source, err, tc.want)
		}
	}
}

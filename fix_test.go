package kubevelafix

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
)

var fixTests = []struct {
	name string
	in   string
	out  string
}{{
	name: "simple",
	in: `
x: 212343
parameter: _
y: {
	for k, v in parameter.p {
		"\(k)": v
	}
}
`,
	out: `
x:         212343
parameter: _
y: {
	for k, v in *parameter.p | {} {
		"\(k)": v
	}
}
`,
}, {
	name: "deep",
	in: `
{
	for k, v in parameter.p.q {
		"\(k)": v
	}
}
`,
	out: `
{
	for k, v in *parameter.p.q | {} {
		"\(k)": v
	}
}
`,
}, {
	name: "several",
	in: `
{
	for k, v in parameter.p {
		"\(k)": v
	}
	for k, v in parameter.q {
		"\(k)": v
	}
}
`,
	out: `
{
	for k, v in *parameter.p | {} {
		"\(k)": v
	}
	for k, v in *parameter.q | {} {
		"\(k)": v
	}
}
`,
}, {
	name: "otherElements",
	in: `
{
	for k, v in parameter.p {
		"\(k)": v
	}
	x: "foo"
	y: {
		for k, v in parameter.p {
			"\(k)": v
		}
	}
	for k, v in other.x {
		"\(k)": v
	}
}
`,
	out: `
{
	for k, v in *parameter.p | {} {
		"\(k)": v
	}
	x: "foo"
	y: {
		for k, v in *parameter.p | {} {
			"\(k)": v
		}
	}
	for k, v in other.x {
		"\(k)": v
	}
}
`}, {
	name: "comments",
	in: `
// comment0
{
	// comment1
	// comment2
	for k, v in parameter.p {
		"\(k)": v
	}

	// comment3
	x: "foo" // comment 4

	// comment5
	y: {
		for k, v in parameter.p {
			"\(k)": v
		}
	}
	// comment 6
	for k, v in other.x {
		"\(k)": v // comment 7
	}
}
`,
	out: `
// comment0
{
	// comment1
	// comment2
	for k, v in *parameter.p | {} {
		"\(k)": v
	}

	// comment3
	x: "foo" // comment 4

	// comment5
	y: {
		for k, v in *parameter.p | {} {
			"\(k)": v
		}
	}
	// comment 6
	for k, v in other.x {
		"\(k)": v // comment 7
	}
}
`}, {
	name: "comprehensionInComprehension",
	in: `
{
	foo: {
		for k, v in parameter.p {
			if v.something {
				"\(k)": v
			}
		}
	}
}
`,
	out: `
{
	foo: {
		for k, v in *parameter.p | {} {
			if v.something {
				"\(k)": v
			}
		}
	}
}
`,
}, {
	name: "fileLevel",
	in: `
import "strings"

for k, v in parameter.p {
	if v.something {
		"\(k)": strings.X(v)
	}
}
`,
	out: `
import "strings"

for k, v in *parameter.p | {} {
	if v.something {
		"\(k)": strings.X(v)
	}
}
`,
}, {
	name: "ifGuarded",
	in: `
foo: {
	if parameter.p != _|_ for k, v in parameter.p {
		"\(k)": v
	}
}
`,
	out: `
foo: {
	if parameter.p != _|_ for k, v in parameter.p {
		"\(k)": v
	}
}
`}, {
	name: "ifGuardedNoMatch",
	in: `
foo: {
	if parameter.p != _|_ for k, v in parameter.q {
		"\(k)": v
	}
}
`,
	out: `
foo: {
	if parameter.p != _|_ for k, v in *parameter.q | {} {
		"\(k)": v
	}
}
`}, {
	name: "ifGuardedSeveralGuards",
	in: `
foo: {
	if parameter.a != _|_ if parameter.b != _|_ for k, v in parameter.a {
		"\(k)": v
	}
}
`,
	out: `
foo: {
	if parameter.a != _|_ if parameter.b != _|_ for k, v in parameter.a {
		"\(k)": v
	}
}
`}, {
	name: "usingIndex",
	in: `
foo: {
	for k, v in parameter["p"] {
		"\(k)": v
	}
}
`,
	out: `
foo: {
	for k, v in *parameter["p"] | {} {
		"\(k)": v
	}
}
`,
}, {
	name: "guardUsingIndex",
	in: `
foo: {
	if parameter["p"] != _|_ for k, v in parameter.p {
		"\(k)": v
	}
}
`,
	out: `
foo: {
	if parameter["p"] != _|_ for k, v in parameter.p {
		"\(k)": v
	}
}
`,
}}

func TestFix(t *testing.T) {
	for _, tc := range fixTests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := parser.ParseFile("x.cue", tc.in, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			f = Fix(f).(*ast.File)
			data, err := format.Node(f)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := strings.TrimSpace(string(data)), strings.TrimSpace(tc.out); got != want {
				t.Fatalf("unexpected output; got:\n%s\nwant:\n%s", got, want)
			}

			// Check that it doesn't change again when round-tripped.
			f, err = parser.ParseFile("x.cue", data, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			f = Fix(f).(*ast.File)
			data1, err := format.Node(f)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(data1), string(data); got != want {
				t.Fatalf("round trip changed output; got: \n%s\nwant:\n%s", got, want)
			}
		})
	}
}

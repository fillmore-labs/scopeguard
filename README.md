# ScopeGuard

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/scopeguard.svg)](https://pkg.go.dev/fillmore-labs.com/scopeguard)
[![Test](https://github.com/fillmore-labs/scopeguard/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/fillmore-labs/scopeguard/actions/workflows/test.yml)
[![CodeQL](https://github.com/fillmore-labs/scopeguard/actions/workflows/github-code-scanning/codeql/badge.svg?branch=main)](https://github.com/fillmore-labs/scopeguard/actions/workflows/github-code-scanning/codeql)
[![Coverage](https://codecov.io/gh/fillmore-labs/scopeguard/branch/main/graph/badge.svg?token=D7ZKQQKAIG)](https://codecov.io/gh/fillmore-labs/scopeguard)
[![Go Report Card](https://goreportcard.com/badge/fillmore-labs.com/scopeguard)](https://goreportcard.com/report/fillmore-labs.com/scopeguard)
[![Codeberg CI](https://ci.codeberg.org/api/badges/15593/status.svg?branch=main)](https://ci.codeberg.org/repos/15593/branches/main)
[![License](https://img.shields.io/github/license/fillmore-labs/scopeguard)](https://www.apache.org/licenses/LICENSE-2.0)

A Go static analyzer that identifies variables with unnecessarily wide scope and suggests moving them to tighter scopes,
following Go's idiomatic scoping patterns.

## Why Narrow Scope Matters

Have you ever scrolled through a long function to find where a variable was last modified, only to find its declaration
200 lines earlier? Wide variable scopes create cognitive overhead, make refactoring harder, and can introduce bugs from
stale data.

Go was designed with narrow scoping in mind — from the `:=` operator to init statements in control structures.
`scopeguard` helps you follow these idiomatic patterns by automatically detecting opportunities to tighten variable
scope. Write Go the way it was meant to be written.

### Examples

**Before:**

```go
func TestProcessor(t *testing.T) {
	// ...
	got, want := spyCC.Charges, charges
	if !cmp.Equal(got, want) {
		t.Errorf("spyCC.Charges = %v, want %v", got, want)
	}
}
```

**After:**

```go
func TestProcessor(t *testing.T) {
	// ...
	if got, want := spyCC.Charges, charges; !cmp.Equal(got, want) {
		t.Errorf("spyCC.Charges = %v, want %v", got, want)
	}
}
```

This pattern is used in
[Go Style Best Practices](https://google.github.io/styleguide/go/best-practices#local-variables-in-tests).

**Before:**

```go
func process(data []byte) error {
	var config Config
	err := json.Unmarshal(data, &config) // err lives in the entire function scope
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// ... rest of function
}
```

**After:**

```go
func process(data []byte) error {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil { // err scoped to if statement
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// ... rest of function
}
```

## Benefits

- **Simplifies refactoring** — Minimizes dependencies when extracting code blocks
- **Reduces cognitive load** — Readers can forget variables once their block ends
- **Enables shorter names** — Variables with small scope can use concise names (as Go style guides recommend)
- **Clearer intent** — Makes the relationship between variables and control structures explicit
- **Prevents reuse errors** — Eliminates accidental reuse of a variable from a previous operation
- **Less pollution** — Avoids cluttering broader scopes with temporary variables
- **Idiomatic Go** — Follows patterns explicitly encouraged by Effective Go and major style guides

## Installation

Choose one of the following installation methods:

### Homebrew

```console
brew install fillmore-labs/tap/scopeguard
```

### Go

```console
go install fillmore-labs.com/scopeguard@latest
```

### Eget

[Install `eget`](https://github.com/zyedidia/eget#how-to-get-eget), then

```console
eget fillmore-labs/scopeguard
```

## What Gets Detected

`scopeguard` analyzes your code to find the smallest possible scope for each variable declaration. It detects
opportunities to move variables to:

- **Initializers** — for `if`, `for`, or `switch` statements
- **Block scopes** — Explicit blocks and case clauses
- **Both `:=` and `var`** — Short declarations and explicit variable declarations

## What Gets Excluded

To ensure correctness, `scopeguard` excludes variables crossing loop or closure boundaries.

## Usage

To analyze your entire project, run:

```console
scopeguard ./...
```

### With automatic fixes

```console
scopeguard -fix ./...
```

**Note:** The `-fix` flag automates refactoring, but some edge cases require manual review. Always verify changes before
committing. See the [Limitations](#limitations) section for details.

### Check generated files

By default, generated files are skipped. To analyze them:

```console
scopeguard -generated ./...
```

## Configuration

You can suppress diagnostics for specific lines using linter comments:

```go
//nolint:scopeguard
x, err := someFunction()
```

This is useful when you've intentionally chosen a wider scope for readability or other reasons.

## When Wider Scope Is Fine

Not every suggestion improves readability. Legitimate patterns where a slightly wider scope makes code clearer include:

- **Reducing nesting** — Early returns that
  [reduce nesting](https://google.github.io/styleguide/go/decisions#indent-error-flow) take priority over artificially
  narrow scopes
- **Sequential error handling** — Reusing `err` across multiple operations is idiomatic in Go
- **Accumulator variables** — Variables that persist across loop iterations

Use your judgment. The tool highlights opportunities; you decide what makes your code clearer.

## Limitations

Always review automated changes. Some cases require manual work after applying `-fix`.

### Shadowed variables

```go
	x, err := a()
	if err != nil {
		fmt.Println(x)
	}

	y, err := b()
	if y != 0 {
		fmt.Println(y)
	}
```

... will be transformed to:

```go
	if x, err := a(); err != nil {
		fmt.Println(x)
	}

	if y, err := b(); y != 0 {
		fmt.Println(y)
	}
```

The compiler will complain because `err` is declared but not used in the second block. You must manually change the
second `err` to `_` to fix this.

### Side effect dependencies

`scopeguard` does not consider implicit dependencies on side effects:

```go
	called := false

	f := func() string {
		called = true
		return "test"
	}

	got, want := f(), "test"

	if !called {
		t.Error("Expected f to be called")
	}

	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
```

... will be replaced by:

```go
	// ... previous code
	if !called {
		t.Error("Expected f to be called")
	}

	if got, want := f(), "test"; got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
```

The call to `f()` is moved after the check for `called`, so the test will fail.

To fix this, either rework your logic not to depend on the side effect so early (testing whether the function has been
called _after_ validating the result), use the result before testing the side effect (`_ = got` is enough and can also
be used to document your dependency on a side effect here) or suppress diagnostic with `//nolint:scopeguard`.

### Variable modification after use

Similarly, the fix can break code that modifies variables used in the calculation:

```go
	const s = "abcd"

	i := 1
	got, want := s[i], byte('b')

	i++

	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
```

In the example above, moving the declaration of `got` and `want` into the `if` statement changes _when_ `s[i]` is
evaluated. The fix places it after `i` is incremented, altering the result and breaking the logic.

Generally, treat `-fix` as a powerful suggestion, not a command. You may need to rework your logic for the suggestion to
be correct.

## Integration

### `go vet`

```console
go vet -vettool=$(which scopeguard) ./...
```

### `golangci-lint` Module Plugin

Add a file `.custom-gcl.yaml` to your source with

```yaml
---
version: v2.6.2

name: golangci-lint
destination: .

plugins:
  - module: fillmore-labs.com/scopeguard
    import: fillmore-labs.com/scopeguard/gclplugin
    version: v0.0.1
```

Then, run `golangci-lint custom` from your project root. You get a custom `golangci-lint` executable that can be
configured in `.golangci.yaml`:

```YAML
---
version: "2"
linters:
  enable:
    - scopeguard
  settings:
    custom:
      scopeguard:
        type: module
        description: "scopeguard identifies variables with unnecessarily wide scope and suggests moving them to tighter scopes."
        original-url: "https://fillmore-labs.com/scopeguard"
```

and can be used like `golangci-lint`:

```shell
./golangci-lint run .
```

See also the golangci-lint
[module plugin system documentation](https://golangci-lint.run/plugins/module-plugins/#the-automatic-way).

## Related Tools

- [`ineffassign`](https://github.com/gordonklaus/ineffassign) — Detect ineffectual assignments.
- [`shadow`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/shadow) - Check for possible unintended shadowing
  of variables.

## Future Work

Since `shadow` might [get deprecated](http://go.dev/issue/75342) and the issues of scope and shadowing are tightly
related, shadow detection might get integrated into a future version. This could also help with the error of moving
shadowed variables.

## Links

- [Blog post about Go scope](https://blog.fillmore-labs.com/posts/scope-1/)

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

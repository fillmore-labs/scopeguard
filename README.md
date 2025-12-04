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
`scopeguard` helps you follow these patterns by automatically detecting opportunities to tighten variable scope, helping
you write more idiomatic Go.

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
	err := json.Unmarshal(data, &config)
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
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// ... rest of function
}
```

## Benefits

- **Simplifies refactoring** — Minimizes dependencies when extracting code blocks
- **Reduces cognitive load** — Readers can forget variables once their block ends
- **Enables shorter names** — Variables with narrow scope can use concise names (as Go style guides recommend)
- **Clearer intent** — Makes the relationship between variables and control structures explicit
- **Prevents reuse errors** — Eliminates accidental reuse of a variable from a previous operation
- **Less pollution** — Avoids cluttering broader scopes with temporary variables
- **Idiomatic Go** — Follows patterns explicitly encouraged by Effective Go and major style guides

## Installation

Choose one of the following installation methods:

### Homebrew

```shell
brew install fillmore-labs/tap/scopeguard
```

### Go

```shell
go install fillmore-labs.com/scopeguard@latest
```

### Eget

[Install `eget`](https://github.com/zyedidia/eget#how-to-get-eget), then

```shell
eget fillmore-labs/scopeguard
```

## What Gets Detected

Opportunities to move variables to initializers of `if`, `for`, or `switch` statements, or to block scopes and case
clauses. It supports both short declarations (`:=`) and explicit variable declarations.

## What Gets Excluded

To ensure correctness, `scopeguard` excludes variables crossing loop or closure boundaries.

## Usage

To analyze your entire project, run:

```shell
scopeguard ./...
```

### With automatic fixes

```shell
scopeguard -fix ./...
```

**Note:** The `-fix` flag automates refactoring, but some cases require manual review. Always verify changes before
committing. See the [Limitations](#limitations) section for details.

### Check generated files

By default, generated files are skipped. To analyze them:

```shell
scopeguard -generated ./...
```

## Configuration

You can suppress diagnostics for specific lines using linter comments:

```go
x, err := someFunction() //nolint:scopeguard
```

This is useful when you've intentionally chosen a wider scope for readability or other reasons.

You can configure the maximum number of lines of a declaration to move:

```shell
scopeguard -max-lines 5 ./...
```

## When Wider Scope Is Fine

Not every suggestion improves readability. Legitimate patterns where a slightly wider scope makes code clearer include
[early returns](https://google.github.io/styleguide/go/decisions#indent-error-flow) that
[reduce nesting](https://github.com/uber-go/guide/blob/2023-05-09/style.md#reduce-nesting).

Use your judgment. The tool highlights opportunities; you decide what makes your code clearer.

## Limitations

Generally, treat `-fix` as a suggestion, not a command. You may need to rework your logic for the suggestion to be
correct.

Always review automated changes. Some cases require manual intervention after applying `-fix`.

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

The call to `f()` is moved after the check for `called`, causing the test to fail.

To fix this, either rework your logic not to depend on the side effect so early (e.g. test whether the function has been
called _after_ validating the result), use the result before testing the side effect (`_ = got` is enough and can also
be used to document your dependency on a side effect), or suppress the diagnostic with `//nolint:scopeguard`.

### Evaluation order changes

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

### Type Changes of Untyped Expressions

When moving a variable declaration, the inferred type may change if the original declaration specified an explicit type.
Consider:

```go
	var a, b int

	a, c := 3.0+1.0, 4.5

	fmt.Println(1 / a)

	if true {
		b = 5.0
		fmt.Println(b, c)
	}
```

... will be transformed to:

```go
	a, c := 3.0+1.0, 4.5

	fmt.Println(1 / a)

	if true {
		var b int
		b = 5.0
		fmt.Println(b, c)
	}
```

Moving the declaration changes `a`'s type from `int` to `float64`, causing a different result for `1 / a`.

This should be rare in practice. To avoid this, ensure variables that need a specific type are declared as narrowly as
possible or use `//nolint:scopeguard` at the declaration.

## Integration

### `go vet`

```shell
go vet -vettool=$(which scopeguard) ./...
```

### `golangci-lint` Module Plugin

Add a file `.custom-gcl.yaml` to your source with

```yaml
---
version: v2.7.0

name: golangci-lint
destination: .

plugins:
  - module: fillmore-labs.com/scopeguard
    import: fillmore-labs.com/scopeguard/gclplugin
    version: v0.0.2
```

Then, run `golangci-lint custom` from your project root. You get a custom `golangci-lint` executable that can be
configured in `.golangci.yaml`:

```yaml
---
version: "2"
linters:
  enable:
    - scopeguard
  settings:
    custom:
      scopeguard:
        type: module
        description:
          "scopeguard identifies variables with unnecessarily wide scope and suggests moving them to tighter scopes."
        original-url: "https://fillmore-labs.com/scopeguard"
        settings:
          max-lines: 10
```

and can be used like `golangci-lint`:

```shell
./golangci-lint run .
```

See also the golangci-lint
[module plugin system documentation](https://golangci-lint.run/plugins/module-plugins/#the-automatic-way).

## Related Tools

- [`ineffassign`](https://github.com/gordonklaus/ineffassign) — Detect ineffectual assignments.
- [`shadow`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/shadow) — Check for possible unintended shadowing
  of variables.
- [`ifshort`](https://github.com/esimonov/ifshort) — Deprecated linter that checks if code uses short syntax for `if`
  statements (Archived).
- [noinlineerr](https://github.com/AlwxSin/noinlineerr) — Linter that prefers wider variable scope (the opposite
  philosophy).

## Future Work

Since `shadow` might [get deprecated](https://go.dev/issue/75342) and the issues of scope and shadowing are tightly
related, shadow detection may be integrated into a future version. This could also help with the error of moving
shadowed variables.

## Links

- [Blog post about Go scope](https://blog.fillmore-labs.com/posts/scope-1/)

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

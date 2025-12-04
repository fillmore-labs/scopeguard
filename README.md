# ScopeGuard

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/scopeguard.svg)](https://pkg.go.dev/fillmore-labs.com/scopeguard)
[![Test](https://github.com/fillmore-labs/scopeguard/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/fillmore-labs/scopeguard/actions/workflows/test.yml)
[![CodeQL](https://github.com/fillmore-labs/scopeguard/actions/workflows/github-code-scanning/codeql/badge.svg?branch=main)](https://github.com/fillmore-labs/scopeguard/actions/workflows/github-code-scanning/codeql)
[![Coverage](https://codecov.io/gh/fillmore-labs/scopeguard/branch/main/graph/badge.svg?token=D7ZKQQKAIG)](https://codecov.io/gh/fillmore-labs/scopeguard)
[![Go Report Card](https://goreportcard.com/badge/fillmore-labs.com/scopeguard)](https://goreportcard.com/report/fillmore-labs.com/scopeguard)
[![Codeberg CI](https://ci.codeberg.org/api/badges/15593/status.svg?branch=main)](https://ci.codeberg.org/repos/15593/branches/main)
[![License](https://img.shields.io/github/license/fillmore-labs/scopeguard)](https://www.apache.org/licenses/LICENSE-2.0)

A Go static analyzer that finds variables declared with unnecessarily wide scope and suggests moving them into tighter
scopes, following Go’s idiomatic scoping patterns.

## Why Narrow Scope Matters

Have you ever scrolled through a long function to find where a variable was last modified, only to discover its
declaration 200 lines earlier? Wide variable scopes add cognitive overhead, make refactoring harder, and can introduce
bugs.

Go was designed with narrow scoping in mind — from the `:=` operator to initialization statements in control structures.
ScopeGuard helps you follow these patterns by detecting opportunities to tighten variable scope so you can write more
idiomatic Go.

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

This pattern appears in
[Go Style Best Practices](https://google.github.io/styleguide/go/best-practices#local-variables-in-tests).

**Before:**

```go
func process(data []byte) error {
	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// ... rest of the function
}
```

**After:**

```go
func process(data []byte) error {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// ... rest of the function
}
```

## Benefits

- **Simplifies refactoring:** Minimizes dependencies when extracting code blocks.
- **Reduces cognitive load:** Readers can forget variables once their block ends.
- **Enables shorter names:** Narrow scope allows concise names (as Go style guides recommend).
- **Clarifies intent:** Makes the relationship between variables and control structures explicit.
- **Prevents reuse errors:** Avoids accidentally reusing values from previous operations.
- **Reduces pollution:** Keeps broader scopes free of temporary variables.
- **Aligns with idiomatic Go:** Follows patterns encouraged by Effective Go and major style guides.

## Installation

Choose one of the following:

### Homebrew

```shell
brew install fillmore-labs/tap/scopeguard
```

### Go

```shell
go install fillmore-labs.com/scopeguard@latest
```

### Eget

[Install `eget`](https://github.com/zyedidia/eget#how-to-get-eget), then:

```shell
eget fillmore-labs/scopeguard
```

## What It Detects

Opportunities to move variables into the initializers of `if`, `for`, or `switch` statements, or into narrower block
scopes and `case` clauses. Both short declarations (`:=`) and explicit variable declarations are supported.

To ensure correctness, ScopeGuard excludes moves that would cross loop or closure boundaries.

ScopeGuard also diagnoses usage-after-shadow and nested assignments.

## Usage

Analyze your project:

```shell
scopeguard ./...
```

### Automatic Fixes

To apply fixes automatically:

```shell
scopeguard -fix ./...
```

> [!NOTE]
>
> Always verify changes before committing. The `-fix` flag automates refactoring, but some cases require manual review.
> See the [limitations](#limitations) section for details.

> [!TIP]
>
> For a safer initial run, use `-scope=conservative` with `-fix`. This only applies changes that don't cross other
> statements:
>
> ```shell
> scopeguard -fix -scope=conservative ./...
> ```

## Configuration

### Command Line Flags

#### Analysis Scope

Control which variables are candidates for moving with the `-scope` flag:

- `full` (default): Analyzes all eligible variable declarations.
- `conservative`: Only suggests moves that don't cross other statements, ensuring no changes to evaluation order or side
  effects.
- `off`: Disables scope analysis.

```shell
scopeguard -scope conservative ./...
```

#### Shadowing Detection

Variable shadowing occurs when a variable declared in an inner scope has the same name as a variable in an outer scope.
This can lead to subtle bugs where you accidentally use the wrong variable. The standard `shadow` tool might
[be deprecated](https://go.dev/issue/75342). Since shadowing is closely related to scope reduction, ScopeGuard includes
shadow detection.

By default, ScopeGuard flags variables that are **used after** being shadowed in an inner scope. While this is legal Go,
it can be difficult to understand:

```go
func example() error {
	var err error

	if err := work(); err == nil {
		fmt.Println("work done")
	}

	return err // Returns nil, regardless of what work() returns
}
```

Control this behavior with the `-shadow` flag:

- `full` (default): Flag variables that are used after being shadowed in an inner scope.
- `off`: Disables shadowing diagnostics.

```shell
scopeguard -shadow off ./...
```

Note that this feature checks for existing shadowing issues and is independent of scope analysis. ScopeGuard's core
analysis will never suggest moving a variable into an inner scope if it is used after that block, preventing this class
of bugs by design.

#### Nested Assignments

Modifying a variable within its own assignment statement (a nested assignment) is hard to read and error-prone during
refactoring. This pattern can introduce subtle bugs when code is parallelized or restructured:

```go
func example() (string, error) {
	var (
		result string
		err    error
	)

	err = retry(func() error {
		result, err = lookup() // Nested reassignment of variable 'err'
		return err
	})

	return result, err
}
```

Fix this by shadowing `err` and assigning the result to the captured outer variable explicitly:

```go
func example() (string, error) {
	var result string

	err := retry(func() error {
		res, err := lookup()
		if err != nil {
			return err
		}

		result = res
		return nil
	})

	return result, err
}
```

Control this behavior with the `-nested-assign` flag:

- `full` (default): Flag nested assignments where a variable is modified within its own assignment expression.
- `off`: Disables diagnostics.

```shell
scopeguard -nested-assign off ./...
```

#### Analysis Targets

- **Generated Files:** By default, generated files are skipped. Include them with `-generated`:

  ```shell
  scopeguard -generated ./...
  ```

- **Test Files:** Tests are included by default. Skip them with `-test=false`:

  ```shell
  scopeguard -test=false ./...
  ```

- **Declaration Length Limit:** Only move declarations up to N lines long into control flow initializers. This prevents
  moving large multi-line declarations (like function literals), which could make them harder to read (default:
  unlimited):

  ```shell
  scopeguard -max-lines 10 ./...
  ```

### Linter Directives

Suppress diagnostics for specific lines using linter comments:

```go
x, err := someFunction() //nolint:scopeguard
```

This is useful when you’ve intentionally chosen a wider scope for readability or other reasons.

## When a Wider Scope Is Fine

Not every suggestion improves readability. Legitimate patterns where a slightly wider scope can be clearer include
[early returns](https://google.github.io/styleguide/go/decisions#indent-error-flow) that
[reduce nesting](https://github.com/uber-go/guide/blob/2023-05-09/style.md#reduce-nesting).

Use your judgment — the tool highlights opportunities; you decide what makes your code clearer.

## Limitations

Always review automated changes from `-fix`. You may need to rework your logic for the suggestion to be correct.

The limitations below (side effect dependencies, evaluation order changes, type changes) do not apply when using
`-scope=conservative`.

### Side Effect Dependencies

ScopeGuard does not account for implicit dependencies on side effects:

```go
called := false

f := func() string {
	called = true
	return "test"
}

got, want := f(), "test"

if !called {
	t.Error("expected f to be called")
}

if got != want {
	t.Errorf("expected %q, got %q", want, got)
}
```

… will be rewritten as:

```go
// ... previous code

if !called {
	t.Error("expected f to be called")
}

if got, want := f(), "test"; got != want {
	t.Errorf("expected %q, got %q", want, got)
}
```

The call to `f()` is moved after the check for `called`, causing the test to fail.

To fix this, either:

- Rework the logic so the side effect is observed at the correct time (e.g., validate the result first, then check the
  side effect), or
- Use the result before testing the side effect (e.g., `_ = got` to document the dependency), or
- Suppress the diagnostic with `//nolint:scopeguard`.

### Evaluation Order Changes

Similarly, a fix can break code that modifies variables used in a calculation:

```go
const s = "abcd"

i := 1
got, want := s[i], byte('b')

i++

if got != want {
	t.Errorf("expected %q, got %q", want, got)
}
```

In this example, moving the declaration of `got` and `want` into the `if` changes _when_ `s[i]` is evaluated. The fix
places it after `i` is incremented, altering the result and breaking the logic.

### Type Changes From Untyped Expressions

When moving a variable declaration, the inferred type can change if the original declaration specified an explicit type.
For example:

```go
var a, b int

a, c := 3.0+1.0, 4.5

fmt.Println(1 / a)

if true {
	b = 5.0
	fmt.Println(b, c)
}
```

… will be transformed to:

```go
a, c := 3.0+1.0, 4.5

fmt.Println(1 / a)

if true {
	var b int
	b = 5.0
	fmt.Println(b, c)
}
```

Moving the declaration changes `a`’s type from `int` to `float64`, causing a different result for `1 / a`.

This should be rare in practice. To avoid it, ensure variables that need a specific type are declared as narrowly as
possible, or use `//nolint:scopeguard` at the declaration.

## Integration

### `go vet`

```shell
go vet -vettool=$(which scopeguard) ./...
```

### `golangci-lint` Module Plugin

Add a `.custom-gcl.yaml` file to your project root:

```yaml
---
version: v2.7.2

name: golangci-lint
destination: .

plugins:
  - module: fillmore-labs.com/scopeguard
    import: fillmore-labs.com/scopeguard/gclplugin
    version: v0.0.3
```

Then run `golangci-lint custom` from your project root. This produces a custom `golangci-lint` executable that can be
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
        description: >-
          scopeguard identifies variables with unnecessarily wide scope and suggests moving them to tighter scopes.
        original-url: https://fillmore-labs.com/scopeguard
        settings:
          scope: conservative
          shadow: full
          max-lines: 10
```

Use it like `golangci-lint`:

```shell
./golangci-lint run ./...
```

The GitHub [golangci-lint-action](https://github.com/golangci/golangci-lint-action#module-plugin-system) will
automatically run the custom `golangci-lint`.

See also the `golangci-lint`
[module plugin system](https://golangci-lint.run/docs/plugins/module-plugins/#the-automatic-way) documentation.

## Related Tools

- [`shadow`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/shadow): Checks for possible unintended shadowing
  of variables.
- [`ifshort`](https://github.com/esimonov/ifshort): Deprecated linter that checks short syntax for `if` statements
  (archived).
- [`noinlineerr`](https://github.com/AlwxSin/noinlineerr): Linter that prefers wider variable scope (the opposite
  philosophy).
- [`ineffassign`](https://github.com/gordonklaus/ineffassign): Detects ineffectual assignments.

## Future Work

Combine multiple short declarations (e.g., `got := f(); want := 42` → `got, want := f(), 42`) to enable moving them into
initializers.

## Links

- [Blog post about Go Scope](https://blog.fillmore-labs.com/posts/scope-1/)
- [Blog post about Shadowing](https://blog.fillmore-labs.com/posts/scope-2/)
- [Blog post about Nested Assignments](https://blog.fillmore-labs.com/posts/scope-3/)

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

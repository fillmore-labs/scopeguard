# ScopeGuard

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/scopeguard.svg)](https://pkg.go.dev/fillmore-labs.com/scopeguard)
[![Test](https://github.com/fillmore-labs/scopeguard/actions/workflows/test.yaml/badge.svg?branch=dev)](https://github.com/fillmore-labs/scopeguard/actions/workflows/test.yaml?query=branch%3Adev)
[![CodeQL](https://github.com/fillmore-labs/scopeguard/actions/workflows/github-code-scanning/codeql/badge.svg?branch=dev)](https://github.com/fillmore-labs/scopeguard/actions/workflows/github-code-scanning/codeql?query=branch%3Adev)
[![Coverage](https://codecov.io/gh/fillmore-labs/scopeguard/branch/dev/graph/badge.svg?token=D7ZKQQKAIG)](https://codecov.io/gh/fillmore-labs/scopeguard/tree/dev)
[![Go Report Card](https://goreportcard.com/badge/fillmore-labs.com/scopeguard)](https://goreportcard.com/report/fillmore-labs.com/scopeguard)
[![Codeberg CI](https://ci.codeberg.org/api/badges/15593/status.svg?branch=dev)](https://ci.codeberg.org/repos/15593/branches/dev)
[![License](https://img.shields.io/badge/license-Apache--2.0-green)](https://www.apache.org/licenses/LICENSE-2.0)

ScopeGuard is a Go static analyzer that identifies variables with unnecessarily wide scopes and suggests moving them
closer to their usage for cleaner, more maintainable code.

## Why Narrow Scope Matters

Have you ever scrolled through a long function trying to find where a variable was last used, only to discover it was
declared 200 lines earlier?

Unnecessarily wide variable scopes increase cognitive overhead and complicate refactoring. When a variable is declared
far from its use, readers must track its lifecycle across many unrelated lines of code.

Narrowing scopes solves this: variables don't need to be tracked once their block ends, code extraction becomes simpler
with fewer dependencies, and stale data can't be accidentally reused.

By placing declarations close to their usage, you make the relationship between variables and control structures
explicit, aligning with patterns from [Effective Go](https://go.dev/doc/effective_go) and major style guides.

Go's design encourages narrow scoping through the `:=` operator and initialization statements in control structures.
ScopeGuard helps you apply these idioms automatically by identifying opportunities to tighten your code.

## Features

ScopeGuard identifies three main categories of code quality improvements:

- **Scope narrowing**: Moves declarations into initializers of `if`, `for`, or `switch` statements, or into narrower
  block scopes and `case` clauses. Supports both short declarations (`:=`) and explicit variable declarations.
- **Shadow detection**: Identifies variables that shadow outer ones in the same function, preventing accidental usage of
  stale data and subtle bugs.
- **Nested assignments**: Flags variables modified within their own assignment statement (e.g., inside a closure), a
  pattern that is often error-prone during refactoring.

## Examples

**Before**:

```go
func TestProcessor(t *testing.T) {
	// ...
	got, want := spyCC.Charges, charges
	if !cmp.Equal(got, want) {
		t.Errorf("spyCC.Charges = %v, want %v", got, want)
	}
}
```

**After**:

```go
func TestProcessor(t *testing.T) {
	// ...
	if got, want := spyCC.Charges, charges; !cmp.Equal(got, want) {
		t.Errorf("spyCC.Charges = %v, want %v", got, want)
	}
}
```

Variables are moved into the `if` initializer, scoped exactly where needed; a practice from
[Go Style Best Practices](https://google.github.io/styleguide/go/best-practices#local-variables-in-tests).

**Before**:

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

**After**:

```go
func process(data []byte) error {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// ... rest of the function
}
```

The `err` variable is scoped to the error-handling block.

## Installation

Choose one of the following:

### Go

```shell
go install fillmore-labs.com/scopeguard@latest
```

### Homebrew

```shell
brew install fillmore-labs/tap/scopeguard
```

### Eget

[Install `eget`](https://github.com/zyedidia/eget#how-to-get-eget), then:

```shell
eget fillmore-labs/scopeguard
```

## Usage

Analyze your code:

```shell
scopeguard ./...
```

Apply fixes automatically:

```shell
scopeguard -fix ./...
```

> Review automated changes before committing. See [Safety and Manual Review](#safety-and-manual-review) for cases
> requiring careful attention.

### Recommended Workflow

1. Start with the default safe pass:

   ```shell
   scopeguard -fix ./...
   ```

2. Review and commit the changes.

3. Run a comprehensive pass that also applies fixes flagged as potentially unsafe:

   ```shell
   scopeguard -fix -unsafe ./...
   ```

4. Review the remaining changes carefully, manually refactoring where narrowing isn't straightforward.

See [Unsafe Fixes](#unsafe-fixes) for a detailed explanation of what "unsafe" means in this context.

## When to Keep a Wider Scope

Not every suggestion improves readability. Patterns like
[early returns](https://google.github.io/styleguide/go/decisions#indent-error-flow) that
[reduce nesting](https://github.com/uber-go/guide/blob/2023-05-09/style.md#reduce-nesting) may benefit from a wider
scope. Always review suggestions to determine if narrowing truly improves clarity.

## Advanced Configuration

ScopeGuard provides additional flags for fine-tuning analysis behavior.

### Scope Analysis

**Flag**: `-scope` (default: `true`)

The eponymous analysis: this is ScopeGuard's core check. Disable this when you only want to check shadowing.

### Shadowing Detection

**Flag**: `-shadow` (default: `true`)

Detects variables **used after** being shadowed in inner scopes. Although legal in Go, this can cause bugs:

```go
func example() error {
	var err error

	if err := work(); err == nil {
		fmt.Println("work done")
	}

	return err // Returns nil, regardless of what work() returns
}
```

ScopeGuard's scope analysis never introduces shadowing issues; it only moves variables when safe.

### Renaming Shadowed Variables

**Flag**: `-rename` (default: `true` with `-fix`)

Automatically renames shadowed variables when using `-fix`:

**Before**:

```go
func transform(x int) int {
	switch x {
	case 1:
		x := x + 1
		return x

	case 2:
		x := x + 2
		if x > 2 {
			x := x + 3
			process(x)
		}

		return x

	default:
		x := x + 4
		process(x)
	}

	return x
}
```

**After**:

```go
func transform(x_2 int) int {
	switch x_2 {
	case 1:
		x := x_2 + 1
		return x

	case 2:
		x_1 := x_2 + 2
		if x_1 > 2 {
			x := x_1 + 3
			process(x)
		}

		return x_1

	default:
		x := x_2 + 4
		process(x)
	}

	return x_2
}
```

The fix appends numeric suffixes (`_1`, `_2`) to outer variables. Replace these with descriptive names during code
review.

This is safe: it only renames variables that already have different scopes, so program semantics don't change.

To disable: `scopeguard -fix -rename=false ./...`

> [!NOTE]
>
> Variable renaming is skipped in functions where scope-narrowing fixes are applied during the same run. Run
> `scopeguard -fix` a second time to rename variables.

> [!TIP]
>
> For safe renaming without scope transformations, run `scopeguard -scope=false -fix ./...`

### Manual Shadow Resolution

Some shadowing issues are better resolved manually rather than by renaming:

**Flagged code**:

```go
func validate(data []byte) ([]byte, error) {
	value, err := retrieve(data)
	if err != nil {
		return nil, err
	}

	if err := check(value); err != nil {
		return nil, err
	}

	return value, err // Flagged: err is used after shadowing
}
```

At the return statement, `err` is `nil`; stale data from previous operations. Explicitly returning `nil` clarifies the
intent:

**Fixed**:

```go
func validate(data []byte) ([]byte, error) {
	value, err := retrieve(data)
	if err != nil {
		return nil, err
	}

	if err := check(value); err != nil {
		return nil, err
	}

	return value, nil // Explicitly return nil
}
```

This makes it immediately obvious that this is the success path without needing to trace `err` backwards through the
function.

### Nested Assignments

**Flag**: `-nested-assign` (default: `true`)

Detects variables modified within their own assignment expression. This pattern is error-prone when code is parallelized
or restructured:

**Before**:

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

Fix this manually by shadowing `err` and explicitly assigning the result to the captured outer variable:

**Fixed**:

```go
func example() (string, error) {
	var result string

	err := retry(func() error {
		res, err := lookup() // Shadow the outer err
		if err != nil {
			return err
		}

		result = res // Explicitly assign the return value only in the success case
		return nil
	})

	return result, err
}
```

### Declaration Combining

**Flag**: `-combine` (default: `true`)

Combines multiple declarations when moving to the same initializer:

**Before**:

```go
got := f(x)
want := "result"
if got != want {
	t.Errorf("got %q, expected %q", got, want)
}
```

**After**:

```go
if got, want := f(x), "result"; got != want {
	t.Errorf("got %q, expected %q", got, want)
}
```

Set to `false` to report candidates without combining them.

### Analysis Targets

- **`-generated`** (default: `false`): Include generated files
- **`-test`** (default: `true`): Include test files
- **`-max-lines N`** (default: unlimited): Skip declarations longer than N lines

### Suppressing Diagnostics

Use `//nolint:scopeguard` to suppress diagnostics on specific lines:

```go
x, err := someFunction() //nolint:scopeguard
```

## Unsafe Fixes

Some scope-narrowing transformations could change program semantics; for example, by reordering evaluation relative to
side effects (see [Side Effect Dependencies](#side-effect-dependencies)). ScopeGuard classifies these as _unsafe_ and
controls them with two flags:

- **`-unsafe-diagnostics`** (default: `true`): report unsafe candidates as diagnostics without offering an automatic
  fix.
- **`-unsafe`** (default: `false`): additionally provide automatic fixes for those candidates. Implies
  `-unsafe-diagnostics`.

Without `-fix`, the two flags are equivalent and both surface the same diagnostics. The difference only matters when
applying fixes:

- `scopeguard -fix ./...`: apply only safe fixes. Unsafe candidates still appear as diagnostics for manual review.
- `scopeguard -fix -unsafe ./...`: apply all fixes, including unsafe ones. Review the resulting diff carefully.
- `scopeguard -unsafe-diagnostics=false ./...`: suppress unsafe diagnostics entirely. Useful in CI, where unfixable
  diagnostics would otherwise add noise.

## Safety and Manual Review

Always review automated changes from `-fix`. In some cases, you may need to restructure your code for the transformation
to be semantically correct.

The scenarios below are examples of what ScopeGuard classifies as [unsafe](#unsafe-fixes), except for rare cases of
[pointer aliasing](#pointer-aliasing) or closure capture, which cannot be detected reliably and can occur even with
otherwise safe fixes.

### Side Effect Dependencies

ScopeGuard doesn't track implicit side effect dependencies:

**Before**:

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
	t.Errorf("got %q, expected %q", got, want)
}
```

**After (breaks test)**:

```go
called := false
f := func() string {
	called = true
	return "test"
}

if !called {
	t.Error("expected f to be called")
}

if got, want := f(), "test"; got != want {
	t.Errorf("got %q, expected %q", got, want)
}
```

The call to `f()` moves after the `called` check, breaking the test.

**Fixes**:

- Rework the logic so the side effect is observed at the correct time (e.g., validate the result first, then check the
  side effect)
- Use the result before testing the side effect (e.g., `_ = got` with a comment to document the dependency)
- Suppress with `//nolint:scopeguard`

### Evaluation Order Changes

Fixes can break code when variables are modified between declaration and use:

```go
const s = "abcd"

i := 1
got, want := s[i], byte('b')

i++

if got != want {
	t.Errorf("got %q, expected %q", got, want)
}
```

Moving the declaration into the `if` evaluates `s[i]` after `i++`, changing the result.

### Implicit Type Changes

Moving declarations can change inferred types when the original specified an explicit type:

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

Moving the declaration changes `a` from `int` to `float64`, altering the result of `1 / a` (integer vs. float division).

This is rare. To avoid it, declare type-specific variables as narrowly as possible or use `//nolint:scopeguard`.

### Pointer Aliasing

Moving declarations can change behavior with pointer aliasing or closure captures.

**Before** (prints 2):

```go
x := 1
px, x := &x, 2
if x == 2 {
	fmt.Println(*px)
}
```

**After** (prints 1):

```go
x := 1
if px, x := &x, 2; x == 2 {
	fmt.Println(*px)
}
```

Use `//nolint:scopeguard` to suppress, or avoid complex aliasing in declarations.

## Integration

### `go vet`

```shell
go vet -vettool=$(which scopeguard) ./...
```

### `golangci-lint` Module Plugin

Add a `.custom-gcl.yaml` file to your project root:

```yaml
---
version: v2.12.2

name: golangci-lint
destination: .

plugins:
  - module: fillmore-labs.com/scopeguard
    import: fillmore-labs.com/scopeguard/gclplugin
    version: v0.0.8
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
          Identifies variables with unnecessarily wide scope and suggests tighter scoping.
        original-url: https://fillmore-labs.com/scopeguard
        settings:
          scope: true
          shadow: true
          nested-assign: true
          unsafe: false
          unsafe-diagnostics: true
          rename: true
          combine: true
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

## MCP Server

The distribution includes an Model Context Protocol (MCP) server designed to help AI agents identify issues or write
more idiomatic code.

The MCP server is included in the Homebrew distribution, or you can install it from source:

```shell
go install fillmore-labs.com/scopeguard/scopeguard-mcp@latest
```

You can then add it to [Claude Code](https://code.claude.com/docs/en/mcp#option-3-add-a-local-stdio-server) with:

```shell
claude mcp add --transport stdio --scope project ScopeGuard -- scopeguard-mcp
```

[Codex](https://developers.openai.com/codex/mcp#add-an-mcp-server):

```shell
codex mcp add ScopeGuard -- scopeguard-mcp
```

or adding

```json
{
  "mcpServers": {
    "ScopeGuard": {
      "command": "scopeguard-mcp"
    }
  }
}
```

to [`.mcp.json`](https://code.claude.com/docs/en/mcp#project-scope),
[`mcp_config.json`](https://antigravity.google/docs/mcp) or the configuration for your tool of choice.

## Related Tools

- [`ifshort`](https://github.com/esimonov/ifshort): Checks short syntax for `if` statements (archived)
- [`shadow`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/shadow): Checks for possible unintended shadowing
  of variables. Expected [to be deprecated](https://go.dev/issue/75342).
- [`ineffassign`](https://github.com/gordonklaus/ineffassign): Detects ineffectual assignments
- [`wastedassign`](https://github.com/sanposhiho/wastedassign): Detects wasted assignments
- [`noinlineerr`](https://github.com/AlwxSin/noinlineerr): Linter that prefers wider variable scope (the opposite
  philosophy).
- [Custom linter](https://github.com/microsoft/typescript-go/pull/365) for Microsoft TypeScript by
  [Jake Bailey](https://jakebailey.dev/posts/go-shadowing/).

## Links

- [Blog post about Go Scope](https://blog.fillmore-labs.com/posts/scope-1/)
- [Blog post about Shadowing](https://blog.fillmore-labs.com/posts/scope-2/)
- [Blog post about Nested Assignments](https://blog.fillmore-labs.com/posts/scope-3/)

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

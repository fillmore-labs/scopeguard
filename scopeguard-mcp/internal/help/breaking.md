# Breaking scope fixes

ScopeGuard guarantees that every automatically-applied fix compiles. Breaking fixes are a stricter tier: structurally
valid edits that could break compilation if applied as-is. They are always generated and always returned by `scope`
preview unless filtered out; use `safety: ["breaking"]` to narrow the response to just this tier. Treat the diff as a
**scaffold**: a starting point for a manual rewrite, not a one-click solution.

## Tradeoff

| Option                | When to choose                                                                                                                                |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| **Leave wide scope**  | Follow-up is non-trivial, or narrower scope doesn't help readability.                                                                         |
| **Apply + edit**      | Follow-up is mechanical (rename, add type annotation, split a declaration). Apply the fix, make the follow-up change immediately, then build. |
| **Refactor manually** | The move is one input among several; restructure the whole block instead.                                                                     |

Leaving wide scope is a valid default. These fixes exist for when the scope win is worth a small manual edit.

## How to request and apply

To narrow the preview to just breaking fixes:

```json
{ "safety": ["breaking"] }
```

Breaking fixes appear in preview with `safety: "breaking"` and a non-empty `reason` field explaining the compilation
risk. They are **never applied by `apply_safe`** (which only writes safe fixes); you must approve each one explicitly:

```json
{ "mode": "apply", "apply": ["<id>"] }
```

After applying, the code likely won't compile. Use the diff as a blueprint, fix the surrounding code, then run
`go build` and tests.

## Decision rule

1. Read the `reason` field and the `diff`.
2. If the required follow-up is trivial, approve the edit and make the follow-up change immediately after.
3. If the required change is non-trivial or unclear, skip the edit and report it to the user.
4. Always run `go build` and tests after applying any breaking fix.

## Categories

### `dec`: Redeclaration in target scope

**Diagnostic**: the variable can move into a tighter block, but the same name is already declared there.

**Breaks with**: `x redeclared in this block`

**Follow-up**: rename one of the two; usually the outer one, since the outer name describes the broader role.

**Skip if**: both names are load-bearing and no good rename is obvious.

```go
func reDeclared() {
    x := 1 // would move inside the if, but x := 2 below already declares x there.
    if true {
        _ = x
        x := 2 // redeclared here → applying the fix causes: "x redeclared in this block"
        _ = x
    }
}
```

---

### `shw`: Shadowed identifier binds to a different variable after move

**Diagnostic**: an identifier used in the initializer is redeclared between the declaration site and the target. Moving
past that redeclaration changes which variable the identifier refers to, silently altering semantics or causing a
compile error.

**Breaks with**: type mismatch or wrong variable used in initializer.

**Follow-up**: ensure the initializer still refers to the intended variable (e.g. introduce a temporary that captures
the original value before the shadowing point).

**Skip if**: the semantic change is hard to reason about without reading more context.

```go
type B struct{ s string }
func (b B) String() string { return b.s }
func (b B) GoString() string { return b.s }

func goStringer() {
    var b fmt.Stringer = B{"test"}
    s := b.String() // uses the outer b (fmt.Stringer)
    if b, ok := b.(fmt.GoStringer); ok { // inner b shadows outer b
        if s == b.GoString() { // moving s := b.String() here uses the inner b instead
            fmt.Println("equal")
        }
    }
}
```

---

### `typ`: Type information lost at new declaration site

**Diagnostic**: the first declaration carries a named type that a later `:=` statement relies on. Removing it causes
type inference to change for all variables in that later statement.

**Breaks with**: missing method or constant truncation; e.g. `i.String()` fails when `i` is inferred as `untyped int`
instead of the named type `I`.

**Follow-up**: add an explicit type annotation at the new declaration site (`var i I`), or split the combined
declaration so only the used variables are moved.

**Skip if**: the type dependency spans many lines and a clean split isn't obvious.

```go
type I int
func (i I) String() string { return strconv.Itoa(int(i)) }

func intString() {
    i, j := I(4), 4 // i has type I; j has type int
    i, k := 2, 2  // i is redeclared: type I is inherited from the line above

    if k != 0 { fmt.Println(i, j) }
    fmt.Println(i.String(), k) // i.String() requires type I
}
// Moving i, j := I(4), 4 into the inner scope drops the type anchor for i, k := two, 2.
```

---

### `tch`: Inferred type changes for a later reassignment

**Diagnostic**: a later `:=` in the same scope reassigns the variable and relies on the type established by the first
declaration. Moving the first declaration changes the inferred type of the reassigned variable.

**Breaks with**: type mismatch in assignment or method-set error.

**Follow-up**: add an explicit type annotation to the `:=` statement, or keep the original declaration in place and move
only the narrower usage.

**Skip if**: the type dependency is indirect or the follow-up risks changing semantics elsewhere.

```go
func untypedNilUse() {
    var err error // establishes err's type as error

    err, ptr := nil, *new(error) // nil is untyped; err's type error makes this valid
    if err == ptr {
        fmt.Println(err)
    }
}
// Moving var err error into the if scope removes the type context for err, ptr := nil, ...
```

---

For the tier below breaking fixes (moves that compile but may reorder side effects) see help topic `unsafe`.

# Unsafe scope fixes

The `scope` tool has three modes, controlled by the `mode` field:

| mode              | behavior                                                                                                                                             |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| omitted (default) | Preview: returns every fix (safe, unsafe, and breaking) as a unified diff with a stable `id` and a `safety`/`reason` pair. No files written.         |
| `"apply"`         | Applies only the fixes whose `id` values appear in the `apply` field. Unknown IDs are an error. All other fixes are still diff-rendered.             |
| `"apply_safe"`    | Applies every **safe** fix in one shot. Unsafe and breaking fixes are still diff-rendered, never written; use `"apply"` with explicit IDs for those. |

Unsafe moves, most commonly `xst` (intervening statements with side effects) are structurally valid edits but could
change the order in which side-effecting calls are evaluated relative to a variable initializer. The analyzer cannot
prove that it is harmless; judgment is required.

**Decision rule for agents:**

1. Run `analyze` to see all issues.
2. Run `scope` (default / preview) to get diffs and IDs for every fix.
3. Classify each edit by its `safety` field:
   - **`safe`**: select automatically.
   - **`unsafe`**: read the diff and the `reason` field. If the move looks correct given the surrounding code (no
     observable reordering of side effects), include the ID. If uncertain, skip it and report it to the user.
4. Run `scope` with `mode: "apply"` and `apply: [<approved IDs>]`.
   - Use `mode: "apply_safe"` as a shortcut to apply every safe fix at once; unsafe and breaking fixes still need
     explicit approval via `"apply"`.
5. After any apply, run tests.

To see only unsafe fixes, use the safety filter:

```json
{ "safety": ["unsafe"] }
```

## What makes an unsafe move risky

Moving a declaration past intervening statements is unsafe when those statements would change what the initializer
produces, either by mutating something the initializer reads, or by having observable side effects whose order matters.

Example: `a` cannot be moved inside the `if`, `i++` between the declaration and the use would change `str[i]`:

```go
// Unsafe: side-effect reordering
func unsafeMove() {
	str := "ab"
	i := 0
	a := str[i]
	i++
	if a == 'a' {
		fmt.Printf("%c\n", str[i])
	}
}
```

A second risk is **pointer aliasing**: moving a declaration across a pointer or slice operation can change aliasing
behavior.

When the initializer is a simple literal, a field read, or a pure function call with no shared mutable state, an unsafe
move is almost always safe to approve.

For a stricter tier of fixes that are likely to break compilation, see help topic `breaking`.

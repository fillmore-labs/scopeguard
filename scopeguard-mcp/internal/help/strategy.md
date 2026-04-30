# Working through large codebases (strategy)

When `summary.total` is high (200+), working on everything at once wastes context. Choose one of the two strategies
below.

## Strategy 1: blind safe-apply

Apply all safe fixes in one shot without reviewing individual diffs. Safe fixes are guaranteed to compile and preserve
behavior, so review is optional.

1. Call `scope` with `safety: ["safe"]` and `mode: "apply_safe"`: applies every safe fix. `summary.by_safety.safe` shows
   how many exist beforehand.

   ```json
   { "safety": ["safe"], "mode": "apply_safe" }
   ```

2. Re-run `analyze`. Safe issues are gone; the remaining count is smaller.
3. Repeat until `summary.total` is manageable, then review unsafe and breaking fixes.

Best for: initial cleanup of an existing codebase where review of individual safe fixes is not required.

## Strategy 2: function-scoped iteration

Work one function at a time. All three tools accept a `functions` filter that restricts analysis and edits to named
functions, so `analyze` → `scope` → `shadow` can target the same function in sequence.

1. Pick a function; start with one you are about to modify anyway, or pick any from a busy file.
2. Run `analyze` with `functions: ["funcName"]` to see its issues.
3. Run `scope` and/or `shadow` with the same filter to fix them.
4. Move to the next function.

```json
{ "dir": "./internal/api", "functions": ["submit"] }
```

The per-function scope is small enough that `summary.total` stays within the default limit and unsafe/breaking diffs are
easy to review.

Best for: targeted refactoring, code review, or any situation where understanding each change matters.

## Choosing a strategy

| Situation                           | Recommended                                                           |
| ----------------------------------- | --------------------------------------------------------------------- |
| Large existing codebase, first pass | Blind safe-apply to clear the bulk, then function-scoped for the rest |
| PR review or targeted change        | Function-scoped from the start                                        |
| Only safe fixes wanted              | `scope` with `safety: ["safe"]` and `mode: "apply_safe"`              |

For limit mechanics and understanding truncated responses, see help topic `limits`.

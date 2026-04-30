# Handling large codebases (limits)

Every tool response includes a `summary` block with aggregate counts. `total` covers all matching diagnostics
(independent of the limit; safety-filtered when a `safety` filter is in effect). `dropped` is the number not returned
because the response was truncated to the limit:

```json
{
  "summary": {
    "total": 142,
    "dropped": 92,
    "by_category": { "scope": 120, "shadow": 22 },
    "by_safety": { "safe": 85, "unsafe": 30, "breaking": 27 },
    "applied": 0
  }
}
```

When `summary.dropped > 0` the response is truncated. The recommended workflow:

1. **Auto-apply safe fixes**: call `scope` with `safety: ["safe"]` and `mode: "apply_safe"` (applies only safe fixes).
   `by_safety.safe` tells you how many exist across the whole codebase:

   ```json
   { "safety": ["safe"], "mode": "apply_safe" }
   ```

2. **Re-analyze**: run `analyze` again. Safe fixes are mostly gone; the remaining issues are smaller in number and may
   now fit within the default limit.

## Raising the limit

Pass `limit: N` to any tool to increase (or decrease) the number of returned issues. The default is **50**. There is no
"unlimited" mode; set a large value (e.g. `limit: 10000`) if you need everything at once, but be aware that large
responses consume significant context.

## Targeting a specific area

Use `functions: ["funcName"]` to restrict any tool to a single function, or `dir: "./pkg/foo"` to target a specific
package. Narrowing scope keeps `total` within the default limit and makes unsafe and breaking diffs easy to review. See
help topic `strategy`.

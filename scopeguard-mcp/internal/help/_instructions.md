# ScopeGuard

Go scope/shadow linter for **code quality**, not bug hunting. Produces more readable, idiomatic code: tighter scopes and
unambiguous names.

## Workflow Integration

Whenever you author or refactor Go code, you MUST run `analyze` on the affected package(s) before reporting the task
complete. `analyze` is fast and purely diagnostic (no file changes); skipping it leaves avoidable quality regressions.
This catches scope and shadow issues that the compiler and gopls do not flag.

This tool acts as a final quality gate. **Step 8** in the gopls 'Editing workflow': you MUST run `analyze` after tests,
before declaring the task complete. Outside that workflow, treat it as the post-edit quality gate.

By default, providing a `dir` only analyzes the Go files directly in that directory. To recursively analyze all
subdirectories across the entire project, you MUST provide `packages: ["./..."]` along with the root `dir`.

Example: After editing `src/server.go`, run `analyze({ "dir": "/absolute/path/to/src" })`. To check the whole workspace,
run `analyze({ "dir": "/absolute/path/to/workspace", "packages": ["./..."] })`.

If `analyze` returns no issues, you're done. If it returns issues, follow the workflow below before declaring the task
finished; do not leave reported issues unaddressed without a specific reason documented to the user.

## Workflow

Every response includes a `next_step` field with a structured recommendation. Follow it.

1. `analyze` → no issues: done.
2. Scope issues → `scope` (default = preview) returns all fixes with IDs, diffs, and an explicit `safety` field per
   edit. Auto-select edits where `safety == "safe"`. For `safety == "unsafe"`, read the diff and reason before
   approving. For `safety == "breaking"`, see help topic `breaking`. Then `scope` with `mode: "apply"` and
   `apply: [<approved IDs>]`. As a shortcut for safe-only batches, `mode: "apply_safe"` writes every safe fix and leaves
   unsafe/breaking diffs untouched.
3. Shadow issues → preview first by calling `shadow` (without `write` and without `renames`) to see which outer
   variables need names and in what order. Then call `shadow` again with `renames` and `write: true` to commit the
   renames to disk. See help topic `naming` for how to construct the rename list.

## Safety tiers

Every edit and issue carries a `safety` field:

| `safety`   | Meaning                                                                         | Default action          |
| ---------- | ------------------------------------------------------------------------------- | ----------------------- |
| `safe`     | Fix can be applied automatically                                                | Apply                   |
| `unsafe`   | Structurally valid but may reorder side effects (see help topic `unsafe`)       | Review diff, then apply |
| `breaking` | Likely to break compilation; treat diff as scaffold (see help topic `breaking`) | Review then apply + fix |

Use the `safety` filter to target a specific tier:

```json
{ "safety": ["unsafe"] }
```

Omitting `safety` returns all tiers. On the `scope` tool, including `"breaking"` also enables generation of
breaking-tier fixes; excluding it suppresses them.

## Summary and limits

Every response includes a `summary` block with `total`, `by_category`, `by_safety`, `dropped`, and `applied` counts.
`total` covers all matching diagnostics across the analyzed packages, independent of the limit. `dropped` is the number
of items not included in the returned list because the response was truncated. When `summary.dropped > 0`, apply the
safe subset (`scope` with `safety: ["safe"]` and `mode: "apply_safe"`) and re-analyze rather than raising the limit. See
help topic `limits`.

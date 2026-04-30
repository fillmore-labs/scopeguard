# Naming shadowed variables

The `shadow` tool renames the **outer** (first-declared) variable; the one with wider scope that is still used after the
shadow. The inner declaration keeps its short name.

## Choosing names

**Ask: why is this value still in scope?** Name the outer variable after the answer: after the role it plays across the
shadow, not after the call that first produced it (the inner variable already owns that).

- **Errors**: follow the [Go error-naming convention](https://go.dev/wiki/Errors#naming): `err`-prefixed camelCase
  describing the phase or purpose (`errSetup`, `errEncode`, `errCleanup`). Avoid source-based names and suffixes like
  `marshalErr`.
- **Other variables**: name by role: `result`, `total`, `prev`, `first`.

Always pass `renames`. The fallback numeric suffixes (`err_1`, `err_2`) are not idiomatic Go.

## How `renames` works

Every function receives an individual rename list. `{"B.process": {"err": ["errSetup", "errEncode"]}}` renames the first
outer `err` in each function to `errSetup` and the second to `errEncode` in method "process" of struct "B".

## Recommended workflow

1. **Preview first.** Call `shadow` without `write` (preview is the default) and without `renames` on the function to
   see which variables will be renamed and in what order.
2. **Read the function** to understand why the outer variable is still in scope.
3. **Write.** Call `shadow` with `functions`, `renames`, and `write: true`:

   ```json
   { "dir": "...", "functions": ["submit"], "renames": { "B.process": { "err": ["errEncode"] }, "write": true } }
   ```

4. Repeat for each function with shadow issues.

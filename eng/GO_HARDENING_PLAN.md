# Go SDK hardening — analysis & plan (epic #19439)

Working plan for cloud-product epic **#19439** (Go SDK idiom & portability
hardening) and its 4 sub-issues. Every claim below was verified against current
`signalwire-go` HEAD (2026-06-16) — the issues are accurate and the cited
file:lines match. This doc is for review; **open questions are collected at the
end** (saved for the review pass, not decided unilaterally).

Recommended order mirrors the epic: **#19435 → #19437 → #19438 → #19436**
(I reorder #19436 last — it's the only deliberate public-API addition and
benefits from the Windows CI job landing first so the new ctx code is exercised
cross-platform).

---

## #19435 — Windows build break + Linux-only CI  ·  verdict: FIX, do first

**Verified:** 4 unconstrained `syscall.SysProcAttr{Setpgid: true}` sites —
`pkg/relay/internal/mocktest/mocktest.go:47`, `pkg/rest/internal/mocktest/mocktest.go:47`,
`pkg/relay/tls_support_test.go:178`, `pkg/rest/namespaces/tls_support_test.go:188`.
All 3 CI workflows (`test.yml`, `doc-audit.yml`, `surface-audit.yml`) pin
`runs-on: ubuntu-latest`. `Setpgid` is absent from `syscall.SysProcAttr` on
Windows → these packages **fail to compile** on `GOOS=windows`; a `runtime.GOOS`
branch can't help (compile-time field absence).

**Recommendation (matches the issue):**
- Add a `setProcessGroup(cmd *exec.Cmd)` helper split by build constraint, in
  EACH mocktest package: `mocktest_unix.go` (`//go:build unix`) sets
  `Setpgid: true`; `mocktest_windows.go` (`//go:build windows`) is a no-op.
- Route the two `tls_support_test.go` sites through that helper (export it from
  the mocktest package, or add a local constrained helper per test package).
- Add a `windows-latest` job to `test.yml`: `go build/vet/test ./...`. Keep the
  bash/python `run-ci.sh` gate Linux-only (the cross-port mocks are POSIX).

**Effort:** small. **Risk:** low (additive; no Linux behavior change).
**Note:** the no-op on Windows means the mock isn't isolated in its own process
group there — acceptable for a test helper (Windows kills the child directly).

---

## #19437 — server timeouts, execute() timer leak, DataMap nil panic  ·  verdict: FIX (all three)

Three contained defects, all verified:

**1. `http.Server` no timeouts (`pkg/server/server.go:375`)** — only `Addr` +
`Handler` set. Slowloris exposure on every hosted agent.
- *Rec:* set `ReadHeaderTimeout` (the important one), plus `ReadTimeout`,
  `WriteTimeout`, `IdleTimeout`. **Q1 (values)** — see questions.

**2. `time.After` timer leak in `execute()` (`pkg/relay/client.go:866`)** — hot
RPC path; the 30s timer is never stopped on the success path and lingers,
accumulating under load. The repo already does it right at `client.go:442`
(`time.NewTimer` + `defer timer.Stop()`).
- *Rec:* mirror the :442 pattern — `timer := time.NewTimer(30*time.Second);
  defer timer.Stop()`, select on `timer.C`. Mechanical, behavior-identical.

**3. `DataMap.Expression` nil-output panic (`pkg/datamap/datamap.go:340`)** —
`expr.output.ToMap()` is unconditional while `nomatchOutput` is nil-checked at
:342; `Expression(... output *swaig.FunctionResult ...)` accepts `nil` →
panics at serialize time.
- *Rec:* make the branches symmetric — nil-check `expr.output` before
  `.ToMap()`. **Q2 (semantics)** — omit the `output` key vs validate-and-error
  at `Expression()` call time; see questions.

**Effort:** small. **Risk:** low. **Testing:** unit test each (cancel-free timer
stop is hard to assert directly — assert no goroutine/timer leak via a tight
loop, or just rely on the pattern; server timeouts via a slow-client test;
datamap nil via a `Expression(..., nil)` round-trip that no longer panics).

---

## #19438 — repo hygiene (.gitattributes, go.mod, example tags)  ·  verdict: FIX, with one judgment call

**1. No `.gitattributes`** — Windows `core.autocrlf=true` → `.go` checks out
CRLF → `gofmt -l` FMT gate flags every file. *Rec:* add root `.gitattributes`
with `* text=auto eol=lf` + `*.go text eol=lf`. Unambiguous.

**2. `go.mod` `go 1.25.0` patch-pinned (`go.mod:3`)** — conventional form is
`go 1.25`. *Rec:* change to `go 1.25`. **Q3** — whether to add an explicit
`toolchain` directive (only if a pinned toolchain is desired); default = no.

**3. Example build tags inconsistent** — verified **21 untagged** + **32 tagged**
(`//go:build ignore`) dirs under `examples/`; the other three example trees
(`relay/`, `rest/`, `livewire/`) are uniformly tagged. `compile_examples.sh`'s
comment claims all are tagged (false).
- **Judgment call (Q4):** the issue recommends tagging ALL `examples/` with
  `//go:build ignore` to match the other trees. BUT — 21 dirs are currently
  compiled by `go build ./...`, which is real (if accidental) compile-coverage
  of those examples. Tagging them all removes that coverage. Options in
  questions; my lean is **tag-all + add the examples to `compile_examples.sh`'s
  explicit build loop** so coverage is kept but the convention is uniform.

**Effort:** trivial (1+2), small (3). **Risk:** low; (3) touches what
`go build ./...` compiles — verify the build/test gates still pass.

---

## #19436 — thread context.Context through the REST client  ·  verdict: FIX, do last (API addition)

**Verified:** `pkg/rest/client.go:190` builds via `http.NewRequest` (no ctx);
`doRequest` (`:167`) takes no ctx. Public verb methods on **`HTTPClient`**
(`Get/Post/Put/Patch/Delete`, `:138–160`), `CrudResource` (`:239`), and
`PaginatedIterator.Next` take no ctx. RELAY client already threads ctx (good
precedent; 2 sites). (Issue says `Client`; the type is `HTTPClient` — cosmetic.)

**Recommendation (matches the issue — additive, non-breaking):**
- `doRequest` → `doRequestContext(ctx, ...)` using `http.NewRequestWithContext`;
  keep `doRequest` as `doRequestContext(context.Background(), ...)`.
- Add `...Context` variants: `GetContext(ctx, ...)`, `PostContext`, etc.,
  `CrudResource` CRUD `...Context` variants, `PaginatedIterator.NextContext(ctx)`.
  Existing methods delegate with `context.Background()`. No exported signature
  changes → source-compatible; DRIFT-safe (additions, doc in PORT_ADDITIONS.md).
- Keep `http.Client.Timeout` as a backstop.
- **Q5 (shape):** additive `...Context` variants (recommended, non-breaking) vs
  changing signatures to take `ctx` (breaking, more idiomatic). Also: does the
  cross-port surface oracle expect these (would Python have ctx equivalents)?
  — likely PORT_ADDITIONS (Go-idiomatic, no Python counterpart).

**Effort:** medium (touches the REST public surface + namespaces that call the
verbs). **Risk:** medium — surface additions must be documented for DRIFT/SURFACE
gates; verify those stay green.

---

## Cross-cutting

- **Windows CI (#19435) is the linchpin** — it's what makes #19435 *and* every
  future portability fix durable. Land it first.
- **All changes are SDK source + CI + repo config**; none customer-facing on the
  Linux happy path. EMISSION/DRIFT must stay green throughout (these are
  behavior/idiom changes, not wire changes).
- **Sequencing:** #19435 (+ Windows CI) → #19437 → #19438 → #19436. Could be one
  PR per sub-issue (clean review) or one epic PR with per-sub-issue commits.
  **Q6.**

---

## Open questions (for the review pass — NOT yet decided)

- **Q1 — server timeout values (#19437.1): DECIDED (2026-06-16) — match the
  Python posture.** Reference (Python) sets no explicit timeouts → uvicorn
  defaults (~5s idle keep-alive, no read/write/header bounds). Go currently sets
  none → fully unbounded. Fix: `ReadHeaderTimeout: 10s` + `IdleTimeout: 120s`;
  `ReadTimeout` and `WriteTimeout` stay **0** (unbounded — the reference doesn't
  bound them, and WriteTimeout=0 avoids truncating long/streaming AI responses).
- **Q2 — DataMap nil-output (#19437.3): DECIDED — panic at the `Expression()`
  call on nil output.** Python's `expression()` calls `output.to_dict()`
  UNCONDITIONALLY and IMMEDIATELY in the builder (data_map.py:195), so a `None`
  output raises `AttributeError` right there at the call site (build time), not
  at serialize time. Go's current bug is worse: it stores the nil and panics
  far away at serialize time (datamap.go:340). Faithful port = fail at the call
  site: `Expression()` (and `ExpressionRegexp`) panic with a clear message
  (`"datamap: Expression output must not be nil"`) when output is nil. This
  mirrors Python's immediate exception, keeps the chainable `*DataMap` signature
  (no error return), and does NOT break v1.1.0 semver — consistent with the
  #19436 no-break decision. Move the failure from serialize-time to call-time.
- **Q3 — go.mod `go 1.25.0` (#19438.2): REVERSED — leave it `go 1.25.0`.**
  The issue called `1.25.0` "a mistake," but it is REQUIRED here: a dependency
  (`golang.org/x/net@v0.56.0`, also x/text) declares `go 1.25.0` in its own
  go.mod, and Go requires the main module's `go` directive to be >= the max of
  all deps' directives — so `go mod tidy` rewrites `go 1.25` back to `go 1.25.0`
  on every mod operation. Forcing `1.25` is non-idiomatic FOR THIS DEP SET and
  unstable. NOT changed. (#19438 ships parts 1 + 3 only.)
- **Q4 — examples convention (#19438.3): DECIDED — tag all 53 `examples/` dirs
  `//go:build ignore`.** Key finding: `compile_examples.sh` ALREADY compiles
  every example individually via `go build -o /dev/null FILE.go`, which honors
  even `//go:build ignore`-tagged files — so tagging the 21 currently-untagged
  dirs loses NO compile coverage. No separate compile-loop change needed; just
  fix the inaccurate comment. Result: uniform with the other 3 example trees,
  demos no longer pulled into `go build ./...`, coverage preserved.
- **Q5 — REST context API shape (#19436): DECIDED — additive `...Context`
  variants.** Evidence: the RELAY client in this SDK already uses the additive
  pattern (`Run()`/`RunContext(ctx)`, `Dial()`/`DialContext(ctx,...)`) — so this
  matches the SDK's own established convention + the Go stdlib idiom; the repo is
  tagged **v1.1.0** (NOT pre-1.0 → a breaking ctx-first change would violate
  semver); 37 internal namespace files call the verbs and stay untouched.
  Add `GetContext/PostContext/PutContext/PatchContext/DeleteContext`,
  `CrudResource.{List,Create,Get,Update,Delete}Context`, and
  `PaginatedIterator.NextContext`; existing methods delegate with
  `context.Background()`. `doRequest` → `doRequestContext` via
  `http.NewRequestWithContext`. Document the additions in PORT_ADDITIONS.md.
- **Q6 — delivery: DECIDED — ONE PR, multiple commits** (one commit per
  sub-issue), in the order #19435 → #19437 → #19438 → #19436.
- **Q7 — timing: DECIDED — run now** (Python #27 is green/done; this is
  independent of the remaining-ports gate rollout).

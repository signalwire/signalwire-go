# Plan: move skill dependencies out of forced-core in signalwire-go

**Status:** proposal — not started. Reviewed against the Python reference for parity.

## The principle (from the user)

- The **default / full install must stay one step** — `go get` the SDK and *all* skills work, no
  extra ceremony. Don't make the common case harder.
- **Within that, push deps to where they're actually needed.** Don't put a dep in core just because
  you can — only if core genuinely needs it.
- Move *toward* optional deps where we can, carefully.

## Ground truth (verified, file:line)

- **Single Go module** (one `go.mod`). Go has no per-feature optional deps *within* a module — the
  only mechanisms are (a) **separate modules**, or (b) **package-level import isolation** (importing a
  package compiles in that package + its imports, and *only* that).
- **Core does NOT import `pkg/skills/builtin`** — only tests + examples do. So builtin is already
  opt-in at the package boundary. The forcing is *inside* builtin.
- **`pkg/skills/builtin` is a monolith: all 18 skills in ONE package**, each self-registering via
  `func init()` → `skills.RegisterSkill(name, factory)` (e.g. `spider.go:727`). Because the imports are
  package-level, importing builtin for *any* skill compiles in *every* skill's deps.
- **Only ONE skill carries external deps that aren't already in core:** `spider.go` →
  `goquery` + `htmlquery` + `golang.org/x/net/html`. (`claude_skills.go` imports `yaml.v3`, but yaml.v3
  is already a core dep via pom/swml — so it adds nothing.) The other 16 skills are stdlib + internal.
- **Registration is already a side-effect import**: every example does
  `_ "github.com/signalwire/signalwire-go/v3/pkg/skills/builtin"` (e.g. `examples/web_search/main.go:17`).
  `AddSkill(name)` resolves via `GetSkillFactory(name)`, which returns nil (logs "unknown skill") if the
  owning package's `init()` never ran (`agent.go:2249`).

**So the entire forced-dep problem reduces to one skill (spider) and three libs.** This is a small,
contained refactor — not a sweeping re-architecture.

## Parity check vs Python (settled)

Python (verified in signalwire-python):
- **Skill *loading* is lazy** — `importlib` loads a skill module only on `add_skill('spider')`
  (`skills/registry.py:32-66`); importing the SDK does not import spider/web_search.
- **Dependency *install* is forced** — `beautifulsoup4` + `lxml` are **core** deps in
  `pyproject.toml:34,36`, not extras. So a base install of the Python package always pulls lxml+bs4.
- Net: Python is **lazy at import, forced at install.**

Implications for Go:
- At the **install** layer, keeping the HTML libs available by default = **parity** (Python ships them
  to everyone too). A *separate-module* split that removes them from the default install would
  **diverge** from Python — so we do NOT do that as a parity move.
- At the **import/compile** layer, Go is currently *worse* than Python: Python isolates per-skill (use
  `joke`, don't import lxml); Go's monolith compiles spider's deps in even for `math`. **Matching
  Python's per-skill laziness is the parity-aligned fix.**

## The design: split the monolith, keep one-step full install

**Mechanism: package-level import isolation (Option A), one module — NOT separate modules.**

1. **Carve the heavy skill(s) into their own sub-package(s).** Move `spider.go` (the only external-dep
   carrier) from `pkg/skills/builtin/` to `pkg/skills/builtin/spider/`. Its `init()` →
   `skills.RegisterSkill("spider", NewSpider)` moves with it. Now `goquery`/`htmlquery`/`x/net` are
   imported *only* by `pkg/skills/builtin/spider`, not by the base `builtin` package.
   - The 16 light skills stay in `pkg/skills/builtin` (no external deps to isolate; not worth churning).
   - If we want to go further later, each remaining skill *could* get its own sub-package, but there's
     no dep payoff — defer unless we want uniform structure.

2. **Preserve the one-step full install** with an aggregator side-effect package. Add
   `pkg/skills/all` (or keep `builtin` as the "everything" umbrella) that blank-imports every skill
   sub-package:
   <!-- snippet: no-compile illustrative import-path sketch (abbreviated `…/pkg/...` paths, not real module paths) -->
   ```go
   package all
   import (
       _ "…/pkg/skills/builtin"          // the 16 light skills
       _ "…/pkg/skills/builtin/spider"   // + the heavy one
   )
   ```
   A consumer who wants everything writes `_ "…/pkg/skills/all"` — one import, all skills registered,
   identical to today's `_ "…/pkg/skills/builtin"`. **The full path stays one step.** (We can even keep
   `builtin` itself as the umbrella so existing `_ ".../builtin"` imports keep registering all skills —
   see "Compatibility" below.)

3. **The lean path becomes available, not mandatory.** A consumer who wants the SDK *without* the HTML
   stack imports only the light set (`_ "…/pkg/skills/builtin"` if builtin = light-only) and simply
   does not import `…/spider`. `go mod tidy` then drops goquery/htmlquery/x/net from *their* build.
   They opt **out** by not importing — the default (umbrella) still opts everyone **in**.

### Compatibility decision (needs a call — see Open Questions)
Two ways to keep existing `_ ".../pkg/skills/builtin"` imports working:
- **(A) `builtin` stays the umbrella** (re-exports/blank-imports spider). Zero breakage, but `builtin`
  still transitively pulls spider's deps — so the lean path needs a *different* import
  (`…/builtin/light`), inverting the names. Safe, but the "default" keeps the heavy deps.
- **(B) `builtin` becomes light-only; new `…/skills/all` is the umbrella.** Cleaner end state (lean is
  the base, all is the opt-in-everything), but existing `_ ".../builtin"` imports silently stop
  registering spider → `AddSkill("spider")` logs "unknown skill" until they switch to `…/all`. A
  one-line migration, but it IS a behavior change for current code (examples + any consumer).

Recommended: **(B)**, because it makes the dep-minimal path the *default* (matches the user's "don't put
deps in core just because you can") and the full set an explicit opt-in — and update all in-repo
examples to `…/all` in the same PR so nothing in-tree breaks. External consumers get a one-line change,
documented in the PR + CHANGELOG.

## Cost / risk

- **Surface/emission/drift:** spider's *public API* (the `spider` skill name, its tools, `NewSpider`)
  is unchanged — only its package path moves. The audit gates police the SWAIG surface + emission, not
  Go import paths, so **drift/emission/surface gates are unaffected**. (Verify by running run-ci.sh.)
- **Registration:** the `init()`-side-effect model already exists (examples use it); we're extending it,
  not inventing it. Low risk, idiomatic Go (`database/sql`, `image/png` precedent).
- **Tests:** `spider_test.go` moves with the package; any test that imported `builtin` to reach spider
  needs its import path updated.
- **Examples:** every `_ ".../builtin"` example that uses spider/web_search must import the umbrella
  (`…/all`) — mechanical, same PR.
- **Effort:** small. One file moves, one umbrella package added, import-path updates in examples +
  tests. No logic changes.

## Steps

1. Create `pkg/skills/builtin/spider/` ; move `spider.go` + `spider_test.go` there; fix package
   name + any now-cross-package references (spider used only internal helpers + the public `skills`
   registry, so likely none).
2. Add the umbrella `pkg/skills/all` blank-importing `builtin` + `builtin/spider` (+ any future heavy
   sub-packages).
3. Decide compatibility (A)/(B); if (B), repoint in-repo examples/tests from `_ ".../builtin"` to
   `_ ".../skills/all"`.
4. `go mod tidy` — confirm goquery/htmlquery/x/net are now pulled only through the spider sub-package /
   umbrella, and that a build importing only `builtin` (light) drops them.
5. `bash scripts/run-ci.sh` — all 8 gates green (esp. DRIFT/SURFACE/EMISSION unchanged; FMT/LINT clean).
6. Doc: note the import-path change + the lean-vs-full pattern in README/CHANGELOG.

## Explicitly NOT doing (and why)

- **Separate modules** for spider or lambda: diverges from Python (which keeps heavy deps in core
  install) and adds release/versioning overhead, for a payoff (removing libs from the *download*) that
  the user explicitly does NOT want to force. One module stays.
- **Removing lambda from the build:** `pkg/lambda` is already isolated (nothing in core/skills imports
  it; only `cmd/swaig-test` + an example). It pulls `aws-lambda-go`, a light pure-Go dep. Leave it —
  splitting it to a module would be the same divergence + churn for a tiny win. (Watch-only.)
- **Per-skill sub-packages for the 16 light skills:** no external-dep payoff; pure churn. Defer unless
  we want uniform structure for its own sake.

## Open questions for the user

1. **Compatibility (A) vs (B)** above — keep `builtin` as the heavy umbrella (zero breakage, deps stay
   default), or make `builtin` light + `…/skills/all` the opt-in-everything umbrella (cleaner, default
   is lean, one-line consumer migration)? Plan recommends **(B)**.
2. **How far to go now:** just isolate **spider** (the only real dep carrier — 90% of the benefit for
   10% of the work), or also restructure the other 16 light skills into sub-packages for uniformity
   (no dep benefit)? Plan recommends **spider only**.

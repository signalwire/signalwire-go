# WIRED_MODES — load-bearing run-ci modes (plan 1.6 / D7)

The strict-mocks × Part-5 merge race silently DROPPED load-bearing env/mode lines from
individual ports' `scripts/run-ci.sh` (a strict export un-set, a gate then green-and-
vacuous). This manifest is the merge-coherence guard: each line below is a regex the
WIRED-MODES gate (`check_wired_modes.py`) requires to be present in `scripts/run-ci.sh`.
If a future merge drops one, the gate reds instead of shipping a vacuous strict/race lane.

Format (one required pattern per line): `` - `<python-regex>` — <why it is load-bearing> ``.
Prose/headers/comments are ignored, so this file doubles as human documentation.

- `MOCK_RELAY_STRICT=1` — RELAY strict mode: the RELAY test suite re-runs with the shared mock in 400-on-violation mode so a wire-shape regression the tolerant mock would swallow fails loud (STRICT-MOCKS gate).
- `export MOCK_SIGNALWIRE_STRICT` — REST 400 strict default (D3): the REST mock returns 400 on an unknown key / wrong type instead of tolerantly journaling it, exported fleet-wide so the REST-COVERAGE + TEST lanes catch the regression.
- `-race` — RELAY race detector (§2.16): the strict RELAY test pass runs under the Go race detector, so a data race in the context-cancelled WS loop / RWMutex-guarded connection / goroutine dispatch reds the gate instead of flaking.

# DUP_TREE_PAIRS — parallel-doc anti-drift manifest (plan 3.3)

go keeps top-level `rest/` and `relay/` docs+examples trees alongside the `pkg/rest/`
and `pkg/relay/` SDK packages. Any duplicate-basename README/doc must stay in sync: the
duplicate is EITHER byte-identical to the canonical OR a pointer stub linking to it. The
DUP-TREE gate (dup_tree.py) enforces this so the two trees can't silently re-diverge.
Declare each pair as `canonical -> duplicate`:

```
rest/README.md -> pkg/rest/README.md
relay/README.md -> pkg/relay/README.md
```

The repo-level `rest/` and `relay/` trees are the canonical, user-facing docs hubs; the
`pkg/*/README.md` files are pointer stubs into them (the same discipline the `rest` pair
established). The pkg-side stub carries no code fence, so no compile shim
(`_ = time.Second`) is needed there.

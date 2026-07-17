# DUP_TREE_PAIRS — parallel-doc anti-drift manifest (plan 3.3)

go keeps a top-level `rest/` docs+examples tree alongside the `pkg/rest/` SDK package.
Any duplicate-basename README/doc must stay in sync: the duplicate is EITHER byte-identical
to the canonical OR a pointer stub linking to it. The DUP-TREE gate (dup_tree.py) enforces
this so the two trees can't silently re-diverge. Declare each pair as `canonical -> duplicate`:

```
rest/README.md -> pkg/rest/README.md
```

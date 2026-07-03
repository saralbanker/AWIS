# EDR-001 — Module path (CONTRA-1 disposition, IMP §25)

V1 implements one module `github.com/awis/awis` with public package `github.com/awis/awis/sdk` (identical surface). The literal `github.com/awis/sdk` import path is satisfied in V2 via vanity-import redirect or SDK repo split — a one-line import change for applications.

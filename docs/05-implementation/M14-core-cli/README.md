# M14 — Core CLI
**Status:** Partitioned — materialize at entry · **Effort:** 2d (+F-5 scope) · **Window:** Weeks 4–5 (parallel)
**Objective:** TDS-07 first — **[F-3] MUST specify the CLI↔runtime interaction model** (Blueprint-§28-sanctioned local socket, or direct SQLite-WAL access with a recorded CONTRA-style refinement of §9 "no concurrent writers"; PID-file + SIGTERM for `stop`). Then: start (auto-discovery + startup header), stop, version, submit(+--input/--wait), signal, cancel, status(+--watch), trace(+--json/--full), workflow validate|list|show, shared what/where/what-now error renderer, --json everywhere (PP-6), **[F-5] minimal local-path `plugin install` + `plugin list`**, audit read path stays M17.
**Depends on:** M06, M07, M08, M10 (M12 for plugin-install wiring) · **Blocks:** M15, M17
**Primary sources:** IMP §27.M14, §15–16; PRD §15–20, §26–27; Verification F-3/F-5 · **Compilation spec:** IKB §4/M14
**Key ACs:** system test (real binary: start→register→submit→signal→trace→stop; kill -9 → rebuild-state → identical state); golden-output tests vs PRD §17/§19–20 fixtures.

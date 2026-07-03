# EDR-003 — SQLite driver

**Decision:** `modernc.org/sqlite` (pure Go, CGO_DISABLED)

Per PRD Implementation Dependencies and Blueprint §20: the pure-Go modernc driver eliminates the CGO toolchain requirement, enabling cross-compilation and hermetic CI with CGO_ENABLED=0. WAL journal mode is the default for all AWIS databases (enables concurrent readers with a single writer, critical for Step and EventLog access patterns). The require entry is added at M02 when first imported; version is pinned then.

# AWIS

AWIS is the workflow runtime (Step, EventLog, IntelligencePort, StoragePort); OIP is its first application, living at `apps/oip` as a separate Go module (`github.com/awis/oip`) that can only consume the public SDK (`github.com/awis/awis/sdk`); Go's `internal/` visibility rule makes crossing that boundary a compile error, providing a mechanical guarantee that the platform and the application remain decoupled.

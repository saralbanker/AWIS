# AWIS Project

This project was initialized with `awis init`.

## Quick Start

1. **Start the runtime:**
   ```
   awis start
   ```

2. **Submit a workflow instance:**
   ```
   awis submit hello-world --input name=Alice
   ```

3. **Check status:**
   ```
   awis status
   ```

4. **Trace an instance:**
   ```
   awis trace <instance-id>
   ```

## Project Structure

```
.
├── config.yaml              # AWIS configuration (namespace, tick, intelligence)
├── .gitignore               # Excludes .awis/ runtime data
├── workflows/               # Workflow YAML definitions
│   ├── hello-world.yaml     # Minimal linear native workflow
│   ├── with-signal.yaml     # WAIT/signal workflow
│   └── with-intelligence.yaml  # Intelligence step with fallback
├── handlers/                # Native step handler stubs
│   └── example_handler.go
└── README_AWIS.md           # This file
```

## Next Steps

- Edit `config.yaml` to set your namespace and intelligence key.
- Add your own workflow YAML files under `workflows/`.
- Implement handler functions in `handlers/` and register them with the runtime.
- Run `awis workflow validate workflows/*.yaml` to check your definitions.

## Reference

```
awis --help                   # Full command tree
awis workflow validate <file> # Validate a workflow YAML
awis history                  # Recent completed instances
awis logs                     # Runtime log stream
```

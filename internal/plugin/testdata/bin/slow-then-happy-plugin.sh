#!/bin/sh
# slow-then-happy-plugin.sh — Responds with name matching whatever is passed.
# Used as the replacement plugin after a crash in checkpoint tests.
# This version handshakes with name "slow-plugin" (for the checkpoint test).

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"slow-plugin","version":"1.0.0","capabilities":[{"id":"test.slow"}]}}\n' "$id"
            ;;
        execute)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"outputs":{"result":"ok"},"duration_ms":1}}\n' "$id"
            ;;
        shutdown)
            exit 0
            ;;
    esac
done

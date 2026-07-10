#!/bin/sh
# crash-then-happy-plugin.sh — Fixture for respawn after crash in engine test.
# Handshakes with name "crash-plugin" and responds successfully to execute.
# Used when re-registering the crash-plugin after the first crash to let retry succeed.

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"crash-plugin","version":"1.0.0","capabilities":[{"id":"test.cap"}]}}\n' "$id"
            ;;
        execute)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"outputs":{"result":"ok"},"duration_ms":1}}\n' "$id"
            ;;
        shutdown)
            exit 0
            ;;
    esac
done

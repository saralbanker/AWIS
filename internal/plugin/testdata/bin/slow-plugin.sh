#!/bin/sh
# slow-plugin.sh — Delays execute response to test timeout handling.
# On execute: sleeps indefinitely (until killed by the runtime).

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"slow-plugin","version":"1.0.0","capabilities":[{"id":"test.slow"}]}}\n' "$id"
            ;;
        execute)
            # Sleep indefinitely — the runtime will kill us on timeout.
            sleep 3600
            ;;
        shutdown)
            exit 0
            ;;
    esac
done

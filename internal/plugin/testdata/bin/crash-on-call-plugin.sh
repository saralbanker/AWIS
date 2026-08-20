#!/bin/sh
# crash-on-call-plugin.sh — Crashes (exits) immediately after handshake.
# Used to test: crash mid-call → plugin_crash + restart on next call.

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"crash-plugin","version":"1.0.0","capabilities":[{"id":"test.cap"}]}}\n' "$id"
            ;;
        execute)
            # Crash instead of responding.
            exit 1
            ;;
        shutdown)
            exit 0
            ;;
    esac
done

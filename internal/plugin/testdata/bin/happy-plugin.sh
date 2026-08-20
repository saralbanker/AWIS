#!/bin/sh
# happy-plugin.sh — NDJSON JSON-RPC 2.0 fixture: happy-path plugin.
# Speaks TDS-05 protocol: responds to handshake and execute correctly.
# Manifest name/version baked in; capability: "test.cap" with input key "value".

set -e

# Process requests from stdin line by line.
while IFS= read -r line; do
    # Parse method from the JSON line using basic string matching.
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"test-plugin","version":"1.0.0","capabilities":[{"id":"test.cap"}]}}\n' "$id"
            ;;
        execute)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"outputs":{"result":"ok"},"duration_ms":1}}\n' "$id"
            ;;
        shutdown)
            exit 0
            ;;
        *)
            printf '{"jsonrpc":"2.0","id":"%s","error":{"code":-32601,"message":"method not found"}}\n' "$id"
            ;;
    esac
done

#!/bin/sh
# git-context-stub.sh — JSON-RPC 2.0 stub for git-context-plugin capability.
# Speaks TDS-05 protocol: responds to handshake and execute for git.context.assemble.
# Returns a minimal context object sufficient for OIP QG-3 tests.

set -e

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"git-context-plugin","version":"1.0.0","capabilities":[{"id":"git.context.assemble"},{"id":"git.diff.fetch"}]}}\n' "$id"
            ;;
        execute)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"outputs":{"context":{"repo":"stub","branch":"main","commits":[]}},"duration_ms":1}}\n' "$id"
            ;;
        shutdown)
            exit 0
            ;;
        *)
            printf '{"jsonrpc":"2.0","id":"%s","error":{"code":-32601,"message":"method not found"}}\n' "$id"
            ;;
    esac
done

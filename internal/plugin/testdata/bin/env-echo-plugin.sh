#!/bin/sh
# env-echo-plugin.sh — NFR-S-02 env test fixture.
# On execute: returns {"env_var": <value of TEST_CANARY_PARENT_VAR or "absent">}.
# The test verifies the canary parent env var is NOT present in the child.

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"env-plugin","version":"1.0.0","capabilities":[{"id":"test.env"}]}}\n' "$id"
            ;;
        execute)
            # Check if the canary env var is present.
            canary="${TEST_CANARY_PARENT_VAR:-absent}"
            printf '{"jsonrpc":"2.0","id":"%s","result":{"outputs":{"env_var":"%s"},"duration_ms":1}}\n' "$id" "$canary"
            ;;
        shutdown)
            exit 0
            ;;
    esac
done

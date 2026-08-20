#!/bin/sh
# bad-handshake-plugin.sh — Returns wrong name in handshake response.
# Used to test handshake mismatch → plugin_handshake_error + crash count.

while IFS= read -r line; do
    method=$(printf '%s' "$line" | grep -o '"method":"[^"]*"' | head -1 | cut -d'"' -f4)
    id=$(printf '%s' "$line" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    case "$method" in
        handshake)
            # Return wrong name — validation failure.
            printf '{"jsonrpc":"2.0","id":"%s","result":{"name":"wrong-name","version":"1.0.0","capabilities":[{"id":"test.cap"}]}}\n' "$id"
            ;;
        shutdown)
            exit 0
            ;;
    esac
done

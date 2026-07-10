#!/bin/sh
# bad-protocol.sh: exits 0 but writes a response with wrong protocol value → protocol_error.
cat > /dev/null
printf '{"protocol":"awis-subprocess/0","outputs":{}}\n'

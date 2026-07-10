#!/bin/sh
# ok.sh: reads stdin (ignores it), writes a success envelope to stdout.
cat > /dev/null
printf '{"protocol":"awis-subprocess/1","outputs":{"result":"ok"}}\n'

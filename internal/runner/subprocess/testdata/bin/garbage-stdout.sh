#!/bin/sh
# garbage-stdout.sh: exits nonzero with garbage on stdout → subprocess_error.
cat > /dev/null
printf 'this is not json at all\n'
printf 'some stderr output\n' >&2
exit 2

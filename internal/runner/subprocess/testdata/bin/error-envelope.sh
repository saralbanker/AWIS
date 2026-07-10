#!/bin/sh
# error-envelope.sh: exits nonzero but writes a valid error envelope to stdout.
# Per TDS-04 §7 the envelope wins over the exit code.
cat > /dev/null
printf '{"protocol":"awis-subprocess/1","error":{"code":"validation_error","message":"records field must not be empty","details":{"field":"records"}}}\n'
exit 1

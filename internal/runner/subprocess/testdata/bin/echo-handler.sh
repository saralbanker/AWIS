#!/bin/sh
# echo-handler.sh: reads the request envelope from stdin, always emits a
# success response with a fixed output value so the harness route test can
# assert the value flowed to the next step.
cat > /dev/null
printf '{"protocol":"awis-subprocess/1","outputs":{"from_subprocess":"hello"}}\n'

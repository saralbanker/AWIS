#!/usr/bin/env bash
# EEOS docs-lint (AWIS_EEOS.md §1/§12.4; IKB §12): mechanical checks only.
#   1. Every docs/05-implementation/MXX-* dir has a README.md.
#   2. A materialized module dir (>1 .md file) has exactly the 7 contract .md files;
#      an optional cards/ subdirectory is permitted (EEOS ruling in 05-implementation/README.md).
#   3. No file under docs/ cites archive/ (supersession rule, IKB §1).
#   4. The EEOS execution ledger and V-COMMON exist.
set -u
fail=0
err() { echo "docs-lint: $1" >&2; fail=1; }

impl="docs/05-implementation"
contract="AI_EXECUTION_CONTEXT.md DEPENDENCY_MAP.md HANDOFF.md IMPLEMENTATION_SPEC.md README.md TRACEABILITY.md VALIDATION_CHECKLIST.md"

for d in "$impl"/M[0-9][0-9]-*/; do
  [ -f "$d/README.md" ] || err "$d missing README.md"
  n=$(find "$d" -maxdepth 1 -name '*.md' | wc -l)
  if [ "$n" -gt 1 ]; then
    have=$(find "$d" -maxdepth 1 -name '*.md' -exec basename {} \; | sort | tr '\n' ' ' | sed 's/ $//')
    want=$(echo "$contract" | tr ' ' '\n' | sort | tr '\n' ' ' | sed 's/ $//')
    [ "$have" = "$want" ] || err "$d materialized but files != 7-file contract: [$have]"
    extras=$(find "$d" -mindepth 1 -maxdepth 1 -type d ! -name cards)
    [ -z "$extras" ] || err "$d has non-cards subdirectories: $extras"
  fi
done

# Supersession registrars may DESCRIBE the archive; nothing else may cite it.
allow='^docs/(README\.md|02-architecture/README\.md|07-indices/canonical-reference-map\.md)$'
if grep -rl "archive/" docs/ --include='*.md' | grep -Ev "$allow" >/tmp/docs-lint-archive 2>/dev/null; then
  err "files cite archive/: $(tr '\n' ' ' </tmp/docs-lint-archive)"
fi

[ -f "$impl/STATE.md" ] || err "missing $impl/STATE.md (EEOS ledger)"
[ -f "$impl/V-COMMON.md" ] || err "missing $impl/V-COMMON.md"

[ "$fail" -eq 0 ] && echo "docs-lint: OK"
exit "$fail"
